// Serve web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  const source = page.locator('#source');
  await page.locator('#syntax-toggle').click();
  for (const [before, indentation] of [
    ['while f():', '    '],
    ['    while f():', '        '],
    ['    if ready: # comment', '        '],
    ['    elif ready:', '        '],
    ['    else:', '        '],
    ['    for x in values:', '        '],
    ['async def work():', '    '],
    ['class Thing:', '    '],
    ['try:', '    '],
    ['except ValueError:', '    '],
    ['finally:', '    '],
    ['with open("file") as f:', '    '],
    ['match value:', '    '],
    ['    case 1:', '        '],
    ['    value = 3', '    '],
    ['        print("text:")', '        '],
    ['    # while f():', '    '],
    ['    value = {1: "a"}', '    '],
    ['    if x: print("one-line block")', '    '],
    ['    if "unterminated:', '    '],
    ['    if f("#"):', '        '],
    ['    ', '    '],
    ['print("hello")', ''],
    ['\twhile f():', '\t    '],
    ['', ''],
  ]) {
    const original = 'first = 1\n' + before;
    await source.fill(original);
    await source.press('ControlOrMeta+End');
    await source.press('Enter');
    const expected = original + '\n' + indentation;
    assert.equal(await source.inputValue(), expected, `wrong indentation for ${JSON.stringify(before)}`);
    assert.deepEqual(await source.evaluate((element) => [element.selectionStart, element.selectionEnd]), [expected.length, expected.length]);
  }
  await source.fill('        value = 3');
  await source.press('End');
  await source.press('Shift+Tab');
  assert.equal(await source.inputValue(), '    value = 3');
  assert.deepEqual(await source.evaluate((element) => [element.selectionStart, element.selectionEnd]), [13, 13]);
  await source.press('Shift+Tab');
  assert.equal(await source.inputValue(), 'value = 3');
  await source.press('Shift+Tab');
  assert.equal(await source.inputValue(), 'value = 3', 'dedenting an unindented line must be a no-op');
  assert(await source.evaluate((element) => document.activeElement === element), 'Shift+Tab must keep editor focus');
  await source.press('Home');
  await source.press('Tab');
  assert.equal(await source.inputValue(), '    value = 3', 'ordinary Tab must still insert four spaces');
  await source.fill('    a\n        b\n    c');
  await source.evaluate((element) => element.setSelectionRange(0, 16, 'backward'));
  await source.press('Shift+Tab');
  assert.equal(await source.inputValue(), 'a\n    b\n    c', 'dedent selected lines, excluding the next line at the selection boundary');
  assert.deepEqual(await source.evaluate((element) => [element.selectionStart, element.selectionEnd, element.selectionDirection]), [0, 8, 'backward']);
  await source.fill('  \tvalue');
  await source.press('End');
  await source.press('Shift+Tab');
  assert.equal(await source.inputValue(), 'value', 'mixed leading whitespace should remove one visual indentation level');
  await source.fill('    result = before + after');
  await source.evaluate((element) => element.setSelectionRange(13, 19));
  await source.press('Enter');
  assert.equal(await source.inputValue(), '    result = \n     + after', 'Enter should replace the selected text and indent at its start');
  await page.waitForFunction(() => !document.querySelector('.source-editor').classList.contains('syntax-pending'));
  assert.equal(await page.locator('#syntax-code').textContent(), (await source.inputValue()) + '\n');
  assert.equal(await page.locator('#gutter button').count(), 2);
  const saved = await source.inputValue();
  await page.reload();
  assert.equal(await source.inputValue(), saved, 'autoindent must be saved through the normal input handler');
  await source.fill('print("shortcut")');
  await source.press('ControlOrMeta+Enter');
  await page.waitForFunction(() => document.querySelector('#status').textContent === 'Finished', null, { timeout: 60_000 });
  assert.equal(await source.inputValue(), 'print("shortcut")', 'run shortcut must not insert a newline');
  assert.equal(await page.locator('#console').textContent(), 'shortcut\n');
  assert.deepEqual(errors, []);
  console.log('Editor autoindent browser tests passed');
} finally {
  await browser.close();
}
