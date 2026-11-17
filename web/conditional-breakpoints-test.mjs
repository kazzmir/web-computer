// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
const point = (line) => page.locator(`#gutter button[data-line="${line}"]`);
const conditionButton = (line) => page.locator(`#gutter [data-condition-line="${line}"]`);
const conditionInput = page.locator('#breakpoint-condition');
async function condition(line, text, key = null) {
  await conditionButton(line).click();
  await conditionInput.fill(text);
  if (key) await conditionInput.press(key); else await page.getByRole('button', { name: 'Save condition', exact: true }).click();
  assert.equal(await page.locator('#breakpoint-condition-editor').isVisible(), false);
  assert.match(await point(line).getAttribute('class'), /breakpoint/, 'editing a condition must not remove its breakpoint');
}
const value = () => page.getByRole('button', { name: 'Edit variable i', exact: true }).textContent();
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  const loop = 'i = 0\nwhile i < 7:\n    i += 1\nprint(i)';
  await page.locator('#source').fill(loop);
  await point(3).click();
  assert.equal(await conditionButton(3).isVisible(), true);
  await condition(3, 'i == 2', 'ControlOrMeta+Enter');
  assert.equal(await page.locator('#viewport iframe').count(), 0, 'saving with Ctrl+Enter must not run the program');
  assert.match(await conditionButton(3).getAttribute('title'), /i == 2/);
  await conditionButton(3).click();
  await conditionInput.fill('i == 1000');
  await conditionInput.press('Escape');
  assert.match(await conditionButton(3).getAttribute('title'), /i == 2/, 'cancel must preserve the previous expression');
  // A condition edited while the runtime downloads must not be lost before
  // Go's command callback exists.
  let releaseLoad;
  let markLoading;
  const loading = new Promise((resolve) => { markLoading = resolve; });
  const proceed = new Promise((resolve) => { releaseLoad = resolve; });
  await page.route('**/web-computer.wasm', async (route) => { markLoading(); await proceed; await route.continue(); }, { times: 1 });
  await condition(3, 'False');
  await page.locator('#debug').click();
  await loading;
  await condition(3, 'i == 2');
  releaseLoad();
  await status('Paused at line 3');
  assert.match(await value(), /i = 2$/, 'condition must be tested before executing the line');
  assert.equal(await page.locator('#console').textContent(), '');
  await condition(3, 'i == 4');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Paused at line 3');
  assert.match(await value(), /i = 4$/, 'live condition changes must reach the interpreter');
  await condition(3, '');
  assert.equal(await conditionButton(3).evaluate((button) => button.classList.contains('has-condition')), false);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Paused at line 3');
  assert.match(await value(), /i = 5$/, 'empty condition restores an ordinary breakpoint');
  await point(3).click();
  assert.equal(await conditionButton(3).count(), 0);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '7\n');

  // Stepping and manual Break still work when a breakpoint condition is false.
  await page.locator('#source').fill('i = 0\nwhile True:\n    i += 1');
  await point(3).click();
  await condition(3, 'False');
  await page.locator('#debug').click();
  await page.waitForFunction(() => !document.querySelector('#debug-break').disabled, null, { timeout: 60_000 });
  await page.waitForTimeout(150);
  assert.equal(await page.locator('#status').textContent(), 'Running');
  await condition(3, 'i > 2');
  await status('Paused at line 3');
  await condition(3, 'False');
  await page.locator('[data-debug-command="continue"]').click();
  await page.waitForFunction(() => !document.querySelector('#debug-break').disabled);
  await page.locator('#debug-break').click();
  await page.waitForFunction(() => document.querySelector('#status').dataset.state === 'paused');
  await page.locator('[data-debug-command="over"]').click();
  await page.waitForFunction(() => document.querySelector('#status').dataset.state === 'paused');
  await page.locator('#debug-stop').click();
  await status('Stopped');

  // Condition exceptions pause visibly and can be corrected without a rerun.
  await point(3).click();
  await page.locator('#source').fill(loop);
  await point(3).click();
  await condition(3, 'missing_name > 0');
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.match(await value(), /i = 0$/);
  assert.match(await page.locator('#console').textContent(), /Breakpoint condition error[\s\S]*NameError/);
  await condition(3, 'i == 2');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Paused at line 3');
  assert.match(await value(), /i = 2$/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  await condition(3, 'i = 2');
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.match(await page.locator('#console').textContent(), /Breakpoint condition error[\s\S]*SyntaxError/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  await point(3).click();
  await point(3).click();
  assert.equal(await conditionButton(3).evaluate((button) => button.classList.contains('has-condition')), false, 'removing a breakpoint must discard its condition');
  await page.locator('#reset').click();
  assert.equal(await conditionButton(3).count(), 0, 'example reset clears breakpoint conditions');
  assert.deepEqual(errors, []);
  console.log('Conditional breakpoint browser tests passed');
} finally {
  await browser.close();
}
