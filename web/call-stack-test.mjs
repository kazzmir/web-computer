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
async function assertEditorLine(line) {
  const position = await page.locator('#source').evaluate((element) => {
    const before = element.value.slice(0, element.selectionStart);
    const row = before.split('\n').length;
    const style = getComputedStyle(element);
    const top = parseFloat(style.paddingTop) + (row - 1) * parseFloat(style.lineHeight) - element.scrollTop;
    return { row, collapsed: element.selectionStart === element.selectionEnd, visible: top >= 0 && top < element.clientHeight, focused: document.activeElement === element, scroll: element.scrollTop };
  });
  assert.equal(position.row, line);
  assert(position.collapsed && position.visible && position.focused, 'selected frame line must have a visible focused caret');
  assert.equal(await page.locator('#gutter').evaluate((element) => element.scrollTop), position.scroll);
}
async function edit(name, value) {
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).click();
  assert(await page.locator('#call-stack button').evaluateAll((nodes) => nodes.every((node) => node.disabled)), 'frame switching must be blocked during an edit');
  await page.getByRole('textbox', { name: `New value for ${name}`, exact: true }).fill(value);
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  await page.getByRole('button', { name: `Edit variable ${name}`, exact: true }).waitFor();
}
try {
  // Serve a runner with the older inbound field and no frame callback, as a
  // browser may do when runner.js is cached separately from the new WASM/UI.
  await page.route('**/runner.js', async (route) => {
    const response = await route.fetch();
    const body = (await response.text())
      .replace('message.payload ?? message.variable', 'message.variable')
      .replace("    globalThis.webComputerFrameSelected = (state) => send('frame-selected', state);", '');
    await route.fulfill({ response, body });
  });
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
  // Reproduce a stale runner missing callback globals. Frame selection, edits,
  // and subsequent pauses must use the message protocol without panicking.
  const callbacks = await page.frameLocator('#viewport iframe').locator('body').evaluate(() => {
    for (const name of ['webComputerFrameSelected', 'webComputerVariableResult', 'webComputerPaused']) delete globalThis[name];
    return typeof globalThis.webComputerFrameSelected;
  });
  assert.equal(callbacks, 'undefined');
  await page.evaluate(() => {
    window.inspectorSizes = [];
    window.inspectorObserver = new ResizeObserver(() => {
      const { width, height } = document.querySelector('#viewport').getBoundingClientRect();
      window.inspectorSizes.push([width, height]);
    });
    window.inspectorObserver.observe(document.querySelector('#viewport'));
  });
  await selectFrame(2);
  await assertEditorLine(9);
  assert.match(await page.locator('#locals-title').textContent(), /<module>/);
  assert.match(await page.locator('#locals').textContent(), /module_var = 7/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /value = 10/);
  assert.equal(await page.locator('[data-debug-command="out"]').isDisabled(), false, 'step out follows the stopped function, not the inspected module frame');
  await edit('module_var', '20');
  await selectFrame(1);
  await assertEditorLine(7);
  assert.equal(await page.locator('#gutter .inspected').getAttribute('data-line'), '7');
  assert.equal(await page.locator('#gutter .current').getAttribute('data-line'), '3', 'viewing a caller must not move the execution marker');
  assert.match(await page.locator('#locals-title').textContent(), /outer/);
  assert.match(await page.locator('#locals').textContent(), /outer_only = 10/);
  await edit('outer_only', '50');
  await selectFrame(0);
  await assertEditorLine(3);
  assert.equal(await page.locator('#gutter .inspected').count(), 0);
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
  // Navigate between distant frames, including clicking the already selected
  // active frame after manually scrolling elsewhere.
  await page.locator('#gutter .breakpoint').click();
  const distant = ['def first():', '    caller_value = 1', '    second()', '    return caller_value', ...Array.from({ length: 80 }, () => '# spacer'), 'def second():', '    inner_value = 2', '    return inner_value', 'first()'].join('\n');
  const activeLine = distant.slice(0, distant.indexOf('inner_value =')).split('\n').length;
  const moduleLine = distant.split('\n').length;
  await page.locator('#source').fill(distant);
  await page.locator(`#gutter button[data-line="${activeLine}"]`).click();
  await page.locator('#debug').click();
  await status(`Paused at line ${activeLine}`);
  await selectFrame(1);
  await assertEditorLine(3);
  await selectFrame(2);
  await assertEditorLine(moduleLine);
  await selectFrame(0);
  await assertEditorLine(activeLine);
  await page.locator('#source').evaluate((element) => { element.scrollTop = 0; });
  await frameButton(0).click();
  await assertEditorLine(activeLine);
  assert.equal(await page.locator('#status').textContent(), `Paused at line ${activeLine}`);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  await page.locator('#gutter .breakpoint').click();
  await page.locator('#source').fill('import pygame as pg\ndef collides(a, b):\n    result = False\n    return result\na = pg.sprite.Sprite()\na.rect = pg.Rect(0, 0, 1, 1)\nb = pg.sprite.Sprite()\nb.rect = pg.Rect(0, 0, 1, 1)\ngroup = pg.sprite.Group(b)\npg.sprite.spritecollide(a, group, False, collides)');
  await page.locator('#gutter button[data-line="3"]').click();
  await page.locator('#debug').click();
  await status('Paused at line 3');
  await frameButton(0).click();
  await assertEditorLine(3);
  const beforeLibrary = await page.locator('#source').evaluate((element) => element.selectionStart);
  await selectFrame(1);
  assert.match(await page.locator('#locals-title').textContent(), /_spritecollide/);
  assert.equal(await page.locator('#source').evaluate((element) => element.selectionStart), beforeLibrary, 'library frames must not jump to an unrelated line in user source');
  assert.equal(await page.locator('#gutter .inspected').count(), 0);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.deepEqual(errors, []);
  console.log('Debugger call stack browser tests passed');
} finally {
  await browser.close();
}
