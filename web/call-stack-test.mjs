// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
const frameButton = (id) => page.locator(`#call-stack [data-frame-id="${id}"]`);
async function selectFrame(id) {
  await frameButton(id).click();
  await page.waitForFunction((id) => document.querySelector(`#call-stack [data-frame-id="${id}"]`)?.getAttribute('aria-pressed') === 'true', id);
}
async function edit(name, value) {
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).click();
  assert(await page.locator('#call-stack button').evaluateAll((nodes) => nodes.every((node) => node.disabled)), 'frame switching must be blocked during an edit');
  await page.getByRole('textbox', { name: `New value for ${name}`, exact: true }).fill(value);
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).waitFor();
}
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await page.locator('#source').fill('module_var = 7\ndef inner(value):\n    inner_only = value + 1\n    return inner_only\ndef outer():\n    outer_only = 10\n    result = inner(outer_only)\n    return result + outer_only\nanswer = outer()\nprint(answer, module_var)');
  await page.locator('#gutter button[data-line="3"]').click();
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.equal(await page.locator('#call-stack button').count(), 3);
  assert.match(await frameButton(0).textContent(), /inner.*editor.py:3/);
  assert.match(await frameButton(1).textContent(), /outer.*editor.py:7/);
  assert.match(await frameButton(2).textContent(), /<module>.*editor.py:9/);
  assert.match(await page.locator('#locals').textContent(), /value = 10/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /outer_only/);
  await page.evaluate(() => {
    window.inspectorSizes = [];
    window.inspectorObserver = new ResizeObserver(() => {
      const { width, height } = document.querySelector('#viewport').getBoundingClientRect();
      window.inspectorSizes.push([width, height]);
    });
    window.inspectorObserver.observe(document.querySelector('#viewport'));
  });
  await selectFrame(2);
  assert.match(await page.locator('#locals-title').textContent(), /<module>/);
  assert.match(await page.locator('#locals').textContent(), /module_var = 7/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /value = 10/);
  assert.equal(await page.locator('[data-debug-command="out"]').isDisabled(), false, 'step out follows the stopped function, not the inspected module frame');
  await edit('module_var', '20');
  await selectFrame(1);
  assert.match(await page.locator('#locals-title').textContent(), /outer/);
  assert.match(await page.locator('#locals').textContent(), /outer_only = 10/);
  await edit('outer_only', '50');
  await selectFrame(0);
  assert.match(await page.locator('#locals').textContent(), /value = 10/, 'caller edits must not overwrite the active frame arguments');
  assert.equal(await page.locator('#status').textContent(), 'Paused at line 3', 'selecting frames must not resume or move execution');
  await selectFrame(1);
  await page.locator('[data-debug-command="over"]').click();
  await status('Paused at line 4');
  assert.equal(await frameButton(0).getAttribute('aria-pressed'), 'true', 'every new pause defaults to the active frame');
  assert.match(await page.locator('#locals-title').textContent(), /inner/);
  assert.match(await page.locator('#locals').textContent(), /inner_only = 11/);
  await page.locator('[data-debug-command="out"]').click();
  await status('Paused at line 8');
  assert.equal(await page.locator('#call-stack button').count(), 2);
  assert.match(await page.locator('#locals').textContent(), /outer_only = 50/);
  await page.locator('[data-debug-command="out"]').click();
  await status('Paused at line 10');
  assert.equal(await page.locator('#call-stack button').count(), 1);
  assert.match(await page.locator('#locals').textContent(), /answer = 61/);
  assert.equal(await page.locator('[data-debug-command="out"]').isDisabled(), true);
  await page.waitForTimeout(100);
  const sizes = await page.evaluate(() => { window.inspectorObserver.disconnect(); return window.inspectorSizes; });
  for (const size of sizes) assert.deepEqual(size, sizes[0], 'stack/locals changes must not resize the result window');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '61 20\n', 'edits to selected caller and module frames must affect execution');
  assert.equal(await page.locator('#call-stack button').count(), 0);

  await page.locator('#gutter .breakpoint').click();
  await page.locator('#source').fill('def count(n):\n    if n == 0:\n        value = 0\n        return value\n    value = count(n - 1)\n    value += 1\n    return value\ntotal = count(2)\nprint(total)');
  await page.locator('#gutter button[data-line="3"]').click();
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.equal(await page.locator('#call-stack button').count(), 4);
  for (const [id, n] of [[0, 0], [1, 1], [2, 2]]) {
    await selectFrame(id);
    assert.match(await page.locator('#locals').textContent(), new RegExp(`n = ${n}`), 'recursive frames must have distinct locals');
  }
  await page.locator('[data-debug-command="out"]').click();
  await status('Paused at line 6');
  assert.equal(await frameButton(0).getAttribute('aria-pressed'), 'true');
  assert.match(await page.locator('#locals').textContent(), /n = 1/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await page.locator('#call-stack button').count(), 0);
  assert.deepEqual(errors, []);
  console.log('Debugger call stack browser tests passed');
} finally {
  await browser.close();
}
