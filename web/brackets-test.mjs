// Serve web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const source = page.locator('#source');
const marks = page.locator('#bracket-layer .bracket-match');
const paint = () => page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
const marked = async () => (await marks.evaluateAll((nodes) => nodes.map((node) => Number(node.dataset.offset)))).sort((a, b) => a - b);
async function caret(start, end = start) {
  await source.focus();
  await source.evaluate((element, { start, end }) => { element.setSelectionRange(start, end); element.dispatchEvent(new Event('select')); }, { start, end });
  await paint();
}
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  for (const [text, open, close] of [['foo(bar)', 3, 7], ['items[2]', 5, 7], ['data = {1: 2}', 7, 12], ['f([1, {2: 3}])', 2, 12]]) {
    await source.fill(text);
    for (const offset of [open, close, close + 1]) {
      await caret(offset);
      // When the caret is on another bracket (e.g. the next outer closer),
      // it intentionally takes precedence over the preceding inner closer.
      if (offset === close + 1 && ')]}'.includes(text[offset] ?? ' ')) continue;
      assert.deepEqual(await marked(), [open, close], `wrong pair for ${text} at ${offset}`);
    }
    assert.equal(await source.inputValue(), text, 'bracket matching must not edit the source');
  }
  await source.fill('(abc)');
  await caret(0);
  assert.deepEqual(await marked(), [0, 4]);
  await source.press('ArrowRight');
  await paint();
  assert.deepEqual(await marked(), [0, 4], 'match the bracket just before the caret');
  await source.press('ArrowRight');
  await paint();
  assert.equal(await marks.count(), 0, 'moving away should clear highlights');
  await caret(0, 5);
  assert.equal(await marks.count(), 0, 'selected text should not leave stale matches');
  await caret(4);
  await page.locator('#run').focus();
  await paint();
  assert.equal(await marks.count(), 0, 'blur should clear highlights');

  for (const text of ['"(abc)"', '# (abc)', '"""(abc)"""', '([)]', '(abc']) {
    await source.fill(text);
    await caret(text.indexOf('('));
    assert.equal(await marks.count(), 0, 'ignore quoted/comment brackets and mismatched or incomplete pairs');
  }
  await source.fill('(abc');
  await source.press('ControlOrMeta+End');
  await source.press(')');
  await paint();
  assert.deepEqual(await marked(), [0, 4], 'typing a closing bracket updates the pair');
  await source.press('Backspace');
  await paint();
  assert.equal(await marks.count(), 0, 'deleting the closer clears the pair');

  // Matching is independent of the syntax-color toggle.
  await page.locator('#syntax-toggle').click();
  await source.fill('x = {"a": [1, 2]}');
  await caret(4);
  assert.deepEqual(await marked(), [4, 16]);
  await page.locator('#syntax-toggle').click();
  await caret(4);
  assert.deepEqual(await marked(), [4, 16]);

  const lines = Array.from({ length: 65 }, (_, i) => i === 40 ? '\tx = ' + ' '.repeat(140) + '[1, 2]' : '# filler');
  const text = lines.join('\n');
  const open = text.indexOf('[');
  await source.fill(text);
  await caret(open);
  await source.evaluate((element) => {
    element.scrollTop = 40 * parseFloat(getComputedStyle(element).lineHeight) - 100;
    element.scrollLeft = 1000;
    element.dispatchEvent(new Event('scroll'));
  });
  await paint();
  assert.deepEqual(await marked(), [open, open + 5]);
  const difference = await source.evaluate((element, open) => {
    const style = getComputedStyle(element);
    const measure = document.createElement('canvas').getContext('2d');
    measure.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    const prefix = element.value.slice(0, open).split('\n').at(-1);
    const width = measure.measureText('    ' + prefix.slice(1)).width;
    const mark = document.querySelector(`#bracket-layer [data-offset="${open}"]`).getBoundingClientRect();
    const bounds = element.getBoundingClientRect();
    return {
      x: Math.abs(mark.left - (bounds.left + parseFloat(style.paddingLeft) + width - element.scrollLeft)),
      y: Math.abs(mark.top - (bounds.top + parseFloat(style.paddingTop) + 40 * parseFloat(style.lineHeight) - element.scrollTop)),
    };
  }, open);
  assert(difference.x < 1 && difference.y < 1, 'markers must track tabs and horizontal/vertical scrolling');
  await page.getByRole('separator', { name: 'Resize editor and result panes' }).press('ArrowRight');
  await caret(open);
  assert.deepEqual(await marked(), [open, open + 5], 'resizing must preserve matching');
  assert.deepEqual(errors, []);
  console.log('Bracket matching browser tests passed');
} finally {
  await browser.close();
}
