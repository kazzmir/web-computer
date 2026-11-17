// npm install --no-save --package-lock=false playwright
// Serve built web/ first; optionally set BASE_URL and PLAYWRIGHT_MODULE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
const breakpoint = (line) => page.locator(`#gutter button[data-line="${line}"]`).click();
const step = async (command, line) => {
  await page.locator(`[data-debug-command="${command}"]`).click();
  await status(`Paused at line ${line}`);
};
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await page.locator('#source').fill('def add(x):\n    y = x + 1\n    return y\na = 10\nb = add(a)\nc = add(b)\nprint(c)');
  await breakpoint(5);
  await page.locator('#debug').click();
  await status('Paused at line 5');
  assert.match(await page.locator('#locals').textContent(), /a = 10/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /b =/);
  assert.equal(await page.locator('#console').textContent(), '');
  assert.equal(await page.locator('#source').getAttribute('readonly'), '');
  await step('into', 2);
  assert.match(await page.locator('#locals').textContent(), /x = 10/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /y =/);
  await step('over', 3);
  assert.match(await page.locator('#locals').textContent(), /y = 11/);
  await step('over', 6);
  assert.match(await page.locator('#locals').textContent(), /b = 11/);
  await step('over', 7);
  assert.match(await page.locator('#locals').textContent(), /c = 12/);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '12\n');

  // Repeated hits, live breakpoint removal/addition, and stop while paused.
  await page.locator('#source').fill('i = 0\nwhile i < 3:\n    i += 1\nprint(i)');
  await breakpoint(3);
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.match(await page.locator('#locals').textContent(), /i = 0/);
  await step('continue', 3);
  assert.match(await page.locator('#locals').textContent(), /i = 1/);
  await breakpoint(3);
  await breakpoint(4);
  await step('continue', 4);
  assert.match(await page.locator('#locals').textContent(), /i = 3/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await page.locator('#viewport iframe').count(), 0);
  await page.locator('#run').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '3\n');
  assert.equal(await page.locator('#debugger').isVisible(), false);
  assert.deepEqual(errors, []);
  console.log('Debugger browser tests passed');
} finally {
  await browser.close();
}
