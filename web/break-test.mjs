// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const button = page.locator('#debug-break');
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
async function start(code) {
  await page.locator('#source').fill(code);
  await page.locator('#debug').click();
  await page.waitForFunction(() => !document.querySelector('#debug-break').disabled, null, { timeout: 60_000 });
}
async function pause() {
  await button.click();
  await page.waitForFunction(() => document.querySelector('#status').dataset.state === 'paused', null, { timeout: 60_000 });
  assert.equal(await button.isDisabled(), true);
}
try {
  await page.addInitScript(() => {
    window.addEventListener('message', (event) => {
      if (window.holdResumeAck && event.data?.type === 'debug-resumed') {
        window.heldResume = { data: event.data, origin: event.origin, source: event.source };
        event.stopImmediatePropagation();
      }
    }, true);
  });
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  assert.equal(await button.isDisabled(), true);
  await start('counter = 0\nwhile True:\n    counter += 1');
  await pause();
  assert.equal(await page.locator('#gutter .breakpoint').count(), 0, 'Break must not add a persistent breakpoint');
  assert.equal(await page.locator('#call-stack button').count(), 1);
  assert.equal(await page.locator('[data-debug-command="out"]').isDisabled(), true);
  const counter = async () => Number((await page.getByRole('button', { name: 'Edit variable counter', exact: true }).textContent()).match(/counter = (\d+)/)[1]);
  const first = await counter();
  assert(first >= 0);
  assert.match(await page.locator('#debug-location').textContent(), /editor.py:/);
  assert.equal(await page.locator('#gutter .current').count(), 1);
  // A break request delivered while already paused must not poison Continue.
  await page.evaluate(() => document.querySelector('#viewport iframe').contentWindow.postMessage({ channel: 'web-computer', type: 'debug-command', command: 'break' }, location.origin));
  await page.waitForTimeout(100);
  await page.evaluate(() => { window.holdResumeAck = true; });
  await page.locator('[data-debug-command="continue"]').click();
  await status('Running');
  await page.waitForFunction(() => !!window.heldResume);
  assert.equal(await button.isDisabled(), true, 'Break must wait until the interpreter acknowledges Continue');
  await page.evaluate(() => {
    window.holdResumeAck = false;
    window.dispatchEvent(new MessageEvent('message', window.heldResume));
  });
  await page.waitForTimeout(150);
  assert.equal(await page.locator('#status').textContent(), 'Running', 'Continue should not immediately re-break');
  assert.equal(await button.isDisabled(), false);
  await pause();
  assert(await counter() > first, 'interpreter must actually resume between Break requests');
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await button.isDisabled(), true);

  await start('def spin():\n    counter = 0\n    while True:\n        counter += 1\nspin()');
  await pause();
  assert.match(await page.locator('#locals-title').textContent(), /spin/);
  assert.equal(await page.locator('[data-debug-command="out"]').isDisabled(), false);
  assert.equal(await page.locator('#call-stack button').count(), 2);
  await page.locator('#debug-stop').click();
  await status('Stopped');

  // A manual break may pause in another Python file. Do not highlight an
  // unrelated editor line, but keep the stack and locals fully inspectable.
  const helper = 'def spin():\n    counter = 0\n    while True:\n        counter += 1\n';
  await start(`namespace = {}\nexec(compile(${JSON.stringify(helper)}, "helper.py", "exec"), namespace)\nnamespace["spin"]()`);
  await pause();
  assert.match(await page.locator('#debug-location').textContent(), /helper.py:/);
  assert.equal(await page.locator('#gutter .current').count(), 0, 'library pauses must not highlight the wrong user-source line');
  assert.match(await page.locator('#locals').textContent(), /counter =/);
  await page.locator('#call-stack [data-frame-id="1"]').click();
  await page.waitForFunction(() => document.querySelector('#call-stack [data-frame-id="1"]')?.getAttribute('aria-pressed') === 'true');
  assert.equal(await page.locator('#gutter .inspected').getAttribute('data-line'), '3');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Running');
  await pause();
  assert.match(await page.locator('#debug-location').textContent(), /helper.py:/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  await page.locator('#source').fill('print("done")');
  await page.locator('#debug').click();
  await status('Finished');
  assert.equal(await button.isDisabled(), true, 'Break must be disabled after exit');
  await page.locator('#run').click();
  await status('Finished');
  assert.equal(await page.locator('#debugger').isVisible(), false);
  assert.equal(await button.isDisabled(), true);
  assert.deepEqual(errors, []);
  console.log('Debugger manual break browser tests passed');
} finally {
  await browser.close();
}
