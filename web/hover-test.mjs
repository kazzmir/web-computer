// Serve web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const tooltip = page.getByRole('tooltip');
async function fill(text) {
  await page.locator('#source').fill(text);
  await page.waitForTimeout(250); // The documentation index is debounced too.
}
async function hover(offset) {
  const point = await page.locator('#source').evaluate((source, offset) => {
    const style = getComputedStyle(source);
    const before = source.value.slice(0, offset);
    const lines = before.split('\n');
    const row = lines.length - 1;
    const height = parseFloat(style.lineHeight);
    source.scrollTop = Math.max(0, row * height - 100);
    const canvas = document.createElement('canvas');
    const measure = canvas.getContext('2d');
    measure.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    const text = lines.at(-1);
    const tabWidth = measure.measureText(' ').width * 4;
    let x = 0;
    for (const char of text) x += char === '\t' ? tabWidth - x % tabWidth : measure.measureText(char).width;
    source.scrollLeft = Math.max(0, x - 100);
    const rect = source.getBoundingClientRect();
    return { x: rect.left + parseFloat(style.paddingLeft) + x - source.scrollLeft + measure.measureText('M').width / 2, y: rect.top + parseFloat(style.paddingTop) + row * height - source.scrollTop + height / 2 };
  }, offset);
  await page.mouse.move(0, 0);
  await page.mouse.move(point.x, point.y);
}
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await fill('print(len([1, 2]))');
  await hover(1);
  await tooltip.waitFor();
  assert.match(await tooltip.locator('.function-signature').textContent(), /^print\(/);
  assert.match(await tooltip.locator('.function-documentation').textContent(), /Write objects/);
  assert.match(await page.locator('#source').getAttribute('aria-describedby'), /function-tooltip/);
  await page.keyboard.press('Escape');
  assert.equal(await tooltip.isVisible(), false);

  const code = `def greet(name, count=2):
    """Greet a person.

    Returns:
        A greeting, not <img src=x onerror=alert(1)>.
    """
    return name * count

greet("World")
unknown(3)
# print("comment")
text = "print(3)"
thing.print(1)
`;
  await fill(code);
  await hover(code.indexOf('greet("World")') + 2);
  await tooltip.waitFor();
  assert.equal(await tooltip.locator('.function-signature').textContent(), 'greet(name, count=2)');
  assert.match(await tooltip.locator('.function-documentation').textContent(), /Greet a person\.\n\nReturns:\n    A greeting/);
  assert.equal(await tooltip.locator('img').count(), 0, 'docstrings must be displayed as text');
  assert.equal(await tooltip.locator('.function-origin').textContent(), 'This file');
  await page.locator('#syntax-toggle').click();
  await hover(code.indexOf('greet("World")') + 2);
  await tooltip.waitFor();
  assert.equal(await tooltip.locator('.function-signature').textContent(), 'greet(name, count=2)', 'hover works with highlighting enabled');
  for (const offset of [code.indexOf('unknown'), code.indexOf('# print') + 3, code.indexOf('"print(3)"') + 3, code.indexOf('thing.print') + 7]) {
    await hover(offset);
    await page.waitForTimeout(420);
    assert.equal(await tooltip.isVisible(), false, 'unknown methods, strings, comments, and unknown names must not show guessed docs');
  }
  for (const shadowed of ['print = 3\nprint', 'def f(print):\n    print(1)', 'from somewhere import *\nprint(1)']) {
    await fill(shadowed);
    await hover(shadowed.lastIndexOf('print') + 1);
    await page.waitForTimeout(420);
    assert.equal(await tooltip.isVisible(), false, 'ambiguous bindings must have no tooltip');
  }
  const scrolled = Array.from({ length: 70 }, (_, i) => `# line ${i}`).join('\n') + '\n\tprint("scroll")';
  await fill(scrolled);
  await hover(scrolled.lastIndexOf('print') + 2);
  await tooltip.waitFor();
  assert.match(await tooltip.locator('.function-signature').textContent(), /^print\(/, 'hover must account for scroll and tabs');
  await page.locator('#source').press('End');
  assert.equal(await tooltip.isVisible(), false, 'typing/navigation should dismiss the tooltip');
  await fill('print(1)');
  await hover(1);
  await tooltip.waitFor();
  await page.locator('#source').press('ControlOrMeta+A');
  await page.locator('#source').press('Backspace');
  assert.equal(await tooltip.isVisible(), false, 'edits should dismiss stale documentation immediately');
  assert.deepEqual(errors, []);
  console.log('Function hover browser tests passed');
} finally {
  await browser.close();
}
