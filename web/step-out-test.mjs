// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const out = page.locator('[data-debug-command="out"]');
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
const breakpoint = (line) => page.locator(`#gutter button[data-line="${line}"]`).click();
async function debug(text, line) {
  for (const line of await page.locator('#gutter .breakpoint').evaluateAll((nodes) => nodes.map((node) => Number(node.dataset.line)))) await breakpoint(line);
  await page.locator('#source').fill(text);
  await breakpoint(line);
  await page.locator('#debug').click();
  await status(`Paused at line ${line}`);
}
async function stepOut(line) {
  assert.equal(await out.isDisabled(), false);
  await out.click();
  await status(`Paused at line ${line}`);
}
async function edit(name, save) {
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).click();
  assert.equal(await out.isDisabled(), true, 'stepping is disabled during variable edits');
  await page.getByRole('button', { name: save ? 'Save' : 'Cancel', exact: true }).click();
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).waitFor();
}
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await debug('def inner(x):\n    y = x + 1\n    return y\ndef outer():\n    value = inner(3)\n    result = value + 1\n    return result\nanswer = outer()\nprint(answer)', 2);
  await edit('x', false);
  assert.equal(await out.isDisabled(), false, 'cancelling an edit restores function stepping');
  await edit('x', true);
  assert.equal(await out.isDisabled(), false, 'saving an edit restores function stepping');
  await breakpoint(3);
  await stepOut(3); // Breakpoints still take precedence over step out.
  await breakpoint(3);
  await stepOut(6);
  assert.match(await page.locator('#locals').textContent(), /value = 4/);
  await stepOut(9);
  assert.match(await page.locator('#locals').textContent(), /answer = 5/);
  assert.equal(await out.isDisabled(), true, 'step out must be disabled at module level');
  await edit('answer', false);
  assert.equal(await out.isDisabled(), true, 'cancelling a top-level edit must not enable step out');
  await edit('answer', true);
  assert.equal(await out.isDisabled(), true, 'saving a top-level edit must not enable step out');
  // Enforce the restriction in the interpreter too, not just the button.
  await page.evaluate(() => document.querySelector('#viewport iframe').contentWindow.postMessage({ channel: 'web-computer', type: 'debug-command', command: 'out' }, location.origin));
  await page.waitForTimeout(200);
  assert.equal(await page.locator('#status').textContent(), 'Paused at line 9');
  assert.equal(await page.locator('#console').textContent(), '');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '5\n');
  assert.equal(await out.isDisabled(), true);

  await debug('def count(n):\n    if n == 0:\n        value = 0\n        return value\n    value = count(n - 1)\n    value += 1\n    return value\ntotal = count(2)\nprint(total)', 3);
  await stepOut(6);
  assert.match(await page.locator('#locals').textContent(), /n = 1/);
  await stepOut(6);
  assert.match(await page.locator('#locals').textContent(), /n = 2/);
  await stepOut(9);
  assert.equal(await out.isDisabled(), true);
  assert.match(await page.locator('#locals').textContent(), /total = 2/);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');

  await debug('def f():\n    try:\n        value = 1\n        return value\n    finally:\n        print("cleanup")\nresult = f()\nprint(result)', 3);
  await stepOut(8);
  assert.equal(await page.locator('#console').textContent(), 'cleanup\n', 'step out must execute finally blocks');
  assert.equal(await out.isDisabled(), true);
  await page.locator('#debug-stop').click();
  await status('Stopped');

  await debug('def f():\n    value = 1\nf()', 2);
  await out.click();
  await status('Finished');
  assert.equal(await out.isDisabled(), true, 'step out may finish the program when no caller line remains');

  await debug('class C:\n    field = 3\nprint(C.field)', 2);
  assert.equal(await out.isDisabled(), true, 'a class body is not a function despite its nested frame depth');
  await page.locator('#debug-stop').click();
  await status('Stopped');
  await debug('x = 1\nprint(x)', 1);
  assert.equal(await out.isDisabled(), true, 'step out is disabled at an initial top-level breakpoint');
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await out.isDisabled(), true);
  assert.deepEqual(errors, []);
  console.log('Debugger step-out browser tests passed');
} finally {
  await browser.close();
}
