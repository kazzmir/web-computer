// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 1000 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const input = page.locator('#debug-expression');
const output = page.locator('#console');
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
async function evaluate(expression, key = 'Enter') {
  const before = (await output.textContent()).length;
  await input.fill(expression);
  await input.press(key);
  await page.waitForFunction(() => !document.querySelector('#debug-expression').disabled, null, { timeout: 60_000 });
  return (await output.textContent()).slice(before);
}
async function selectFrame(id) {
  await page.locator(`#call-stack [data-frame-id="${id}"]`).click();
  await page.waitForFunction((id) => document.querySelector(`#call-stack [data-frame-id="${id}"]`)?.getAttribute('aria-pressed') === 'true', id);
}
try {
  await page.addInitScript(() => {
    window.addEventListener('message', (event) => {
      if (window.holdEvaluation && event.data?.type === 'evaluation-result') {
        window.heldEvaluation = { data: event.data, origin: event.origin, source: event.source };
        event.stopImmediatePropagation();
      }
    }, true);
  });
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  assert.equal(await page.locator('#debug-evaluator').isVisible(), false);
  const code = 'global_value = 7\ndef twice(n):\n    return n * 2\ndef inner(value, items):\n    local = value + 1\n    result = local\n    return result\ndef outer():\n    value = 20\n    return inner(3, [])\nanswer = outer()\nprint(answer)';
  await page.locator('#source').fill(code);
  for (const line of [3, 6]) await page.locator(`#gutter button[data-line="${line}"]`).click();
  await page.locator('#debug').click();
  await status('Paused at line 6');
  assert.equal(await input.isDisabled(), false);
  assert.match(await evaluate('local + global_value'), /\n11\n$/);
  assert.match(await evaluate('[value, local]'), /\n\[3, 4\]\n$/);
  assert.match(await evaluate('(value, local)'), /\n\(3, 4\)\n$/);
  assert.match(await evaluate('{1: "a"}'), /\n\{1: "a"\}\n$/);
  assert.match(await evaluate('twice(value)'), /\n6\n$/, 'calls must not recursively stop at their source breakpoints');
  assert.equal(await page.locator('#status').textContent(), 'Paused at line 6');
  assert.equal(await page.locator('#call-stack button').count(), 3);
  assert.match(await evaluate('1 / 0'), /ZeroDivisionError/);
  assert.match(await evaluate('not_defined'), /NameError/);
  assert.match(await evaluate('local = 100'), /SyntaxError/);
  assert.match(await page.getByRole('button', { name: 'Edit variable local', exact: true }).textContent(), /local = 4$/);
  const invalid = await evaluate('print("must not run"); 3');
  assert.match(invalid, /SyntaxError/);
  assert.doesNotMatch(invalid, /\nmust not run\n/);
  assert.match(await evaluate('items.append(8) or len(items)'), /\n1\n$/);
  assert.match(await page.getByRole('button', { name: 'Edit variable items', exact: true }).textContent(), /items = \[8\]/, 'evaluation side effects must refresh inspected locals');
  await evaluate('"<img src=x onerror=alert(1)>"');
  assert.equal(await output.locator('img').count(), 0);
  await selectFrame(1);
  assert.match(await evaluate('value + global_value'), /\n27\n$/, 'evaluate in caller locals');
  await selectFrame(2);
  assert.match(await evaluate('global_value'), /\n7\n$/, 'evaluate in module globals');
  assert.match(await evaluate('value'), /NameError/, 'do not leak a different frame locals');
  await selectFrame(0);
  await page.getByRole('button', { name: 'Edit variable value', exact: true }).click();
  assert.equal(await input.isDisabled(), true);
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();
  assert.equal(await input.isDisabled(), false);

  await page.evaluate(() => { window.holdEvaluation = true; window.originalIframe = document.querySelector('#viewport iframe'); });
  await input.fill('2 + 3');
  await input.press('ControlOrMeta+Enter');
  await page.waitForFunction(() => !!window.heldEvaluation);
  assert.equal(await input.isDisabled(), true);
  assert(await page.locator('[data-debug-command], #call-stack button, #locals .local-variable').evaluateAll((nodes) => nodes.every((node) => node.disabled)), 'frame switches, variable edits and stepping must wait for the evaluation response');
  assert.equal(await page.locator('#debug-stop').isDisabled(), false);
  assert(await page.evaluate(() => window.originalIframe === document.querySelector('#viewport iframe')), 'Ctrl+Enter in evaluator must not rerun the program');
  await page.evaluate(() => { window.holdEvaluation = false; window.dispatchEvent(new MessageEvent('message', window.heldEvaluation)); });
  await page.waitForFunction(() => !document.querySelector('#debug-expression').disabled);
  assert.match(await output.textContent(), /\n5\n$/);
  assert.equal(await page.locator('#source').inputValue(), code);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await input.isDisabled(), true);
  assert.match(await output.textContent(), /\n4\n$/, 'program resumes normally after evaluations');

  // Even a nonterminating function called by the evaluator can be stopped.
  for (const line of [3, 6]) await page.locator(`#gutter button[data-line="${line}"]`).click();
  await page.locator('#source').fill('def hang():\n    while True:\n        pass\ndef main():\n    value = 1\n    return value\nmain()');
  await page.locator('#gutter button[data-line="5"]').click();
  await page.locator('#debug').click();
  await status('Paused at line 5');
  await input.fill('hang()');
  await input.press('Enter');
  await page.waitForTimeout(150);
  assert.equal(await input.isDisabled(), true);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await page.locator('#viewport iframe').count(), 0);
  assert.equal(await input.isDisabled(), true);
  await page.locator('#source').fill('print("normal run")');
  await page.locator('#run').click();
  await status('Finished');
  assert.equal(await page.locator('#debug-evaluator').isVisible(), false);
  assert.deepEqual(errors, []);
  console.log('Debugger expression evaluator browser tests passed');
} finally {
  await browser.close();
}
