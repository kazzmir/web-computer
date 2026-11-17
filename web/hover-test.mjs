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
  return point;
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
  for (const functions of [
    'def f(value):\n    """Documentation for f."""\n    return value\ndef g():\n    """Documentation for g."""\n    f(3)\n    unknown()\ng()\n',
    'def g():\n    """Documentation for g."""\n    f(3)\n    unknown()\ndef f(value):\n    """Documentation for f."""\n    return value\ng()\n',
  ]) {
    await fill(functions);
    await hover(functions.lastIndexOf('g()'));
    await tooltip.waitFor();
    assert.equal(await tooltip.locator('.function-signature').textContent(), 'g()');
    await hover(functions.indexOf('f(3)'));
    await tooltip.waitFor();
    assert.equal(await tooltip.locator('.function-signature').textContent(), 'f(value)', 'hover on f() inside g() must show f, not g');
    assert.equal(await tooltip.locator('.function-documentation').textContent(), 'Documentation for f.');
    await hover(functions.indexOf('unknown'));
    await page.waitForTimeout(420);
    assert.equal(await tooltip.isVisible(), false, 'unknown calls inside g() must not fall back to g documentation');
  }
  const pygameCode = 'import pygame as pg\ndef main():\n    """Run the game."""\n    for event in pg.event.get():\n        print(event)\nmain()\n';
  await fill(pygameCode);
  await hover(pygameCode.lastIndexOf('main') + 1);
  await tooltip.waitFor();
  assert.equal(await tooltip.locator('.function-signature').textContent(), 'main()');
  for (const offset of [pygameCode.indexOf('pg.event.get'), pygameCode.indexOf('event.get') + 1]) {
    await hover(offset);
    await page.waitForTimeout(420);
    assert.equal(await tooltip.isVisible(), false, 'pygame attributes must not inherit main documentation');
  }
  await hover(pygameCode.indexOf('get()') + 1);
  await tooltip.waitFor();
  assert.equal(await tooltip.locator('.function-signature').textContent(), 'pygame.event.get()');
  assert.equal(await tooltip.locator('.function-origin').textContent(), 'Pygame (browser API)');
  await hover(pygameCode.indexOf('print(event)') + 1);
  await tooltip.waitFor();
  assert.match(await tooltip.locator('.function-signature').textContent(), /^print\(/, 'builtins in main must resolve to their own docs');
  const pygameFunctions = 'import pygame as pg\ndef main():\n    pg.transform.flip(surface, True, False)\n    pg.image.load("image.png")\n    pg.display.flip()\n    pg.mixer.music.load("music.ogg")\nmain()';
  await fill(pygameFunctions);
  for (const [expression, signature, description] of [
    ['transform.flip', 'pygame.transform.flip(surface, x, y)', /mirrored horizontally/],
    ['image.load', 'pygame.image.load(filename)', /Load an image file/],
    ['display.flip', 'pygame.display.flip()', /Present the current display/],
    ['music.load', 'pygame.mixer.music.load(filename)', /Load a music file/],
  ]) {
    await hover(pygameFunctions.indexOf(expression) + expression.indexOf('.') + 1);
    await tooltip.waitFor();
    assert.equal(await tooltip.locator('.function-signature').textContent(), signature);
    assert.match(await tooltip.locator('.function-documentation').textContent(), description);
  }
  const directImport = 'from pygame.transform import flip as mirror\nmirror(surface, True, False)';
  await fill(directImport);
  await hover(directImport.lastIndexOf('mirror'));
  await tooltip.waitFor();
  assert.equal(await tooltip.locator('.function-signature').textContent(), 'pygame.transform.flip(surface, x, y)');
  const shadowedPygame = 'import pygame as pg\npg = other\npg.image.load("x")';
  await fill(shadowedPygame);
  await hover(shadowedPygame.lastIndexOf('load'));
  await page.waitForTimeout(420);
  assert.equal(await tooltip.isVisible(), false, 'reassigned pygame aliases must not show stale API docs');
  // Double-click navigation targets the callee's own definition, not the
  // enclosing def, and works in either direction across a scrolled document.
  const spacer = Array.from({ length: 70 }, (_, i) => `# spacer ${i}`).join('\n');
  for (const navigationCode of [
    `def f(value):\n    return value\n${spacer}\ndef g():\n    f(3)\ng()`,
    `def g():\n    f(3)\n${spacer}\ndef f(value):\n    return value\ng()`,
  ]) {
    await fill(navigationCode);
    const point = await hover(navigationCode.indexOf('f(3)'));
    await page.mouse.dblclick(point.x, point.y);
    const definition = navigationCode.indexOf('f(value)');
    const state = await page.locator('#source').evaluate((element, definition) => {
      const row = element.value.slice(0, definition).split('\n').length - 1;
      const style = getComputedStyle(element);
      const top = parseFloat(style.paddingTop) + row * parseFloat(style.lineHeight) - element.scrollTop;
      return { selection: [element.selectionStart, element.selectionEnd], top, height: element.clientHeight, focused: document.activeElement === element, scroll: element.scrollTop };
    }, definition);
    assert.deepEqual(state.selection, [definition, definition], 'double-click must place the cursor at the callee definition');
    assert(state.top >= 0 && state.top < state.height, 'definition must be visible');
    assert(state.focused);
    assert.equal(await page.locator('#source').inputValue(), navigationCode, 'navigation must not edit the source');
    assert.equal(await page.locator('#gutter').evaluate((element) => element.scrollTop), state.scroll);
    assert.equal(await page.locator('#syntax-layer').evaluate((element) => element.scrollTop), state.scroll);
    assert.equal(await tooltip.isVisible(), false);
  }
  for (const classCode of [
    `class Widget:\n    def __init__(self, value):\n        self.value = value\n${spacer}\nWidget(3)`,
    `def main():\n    Widget(3)\n${spacer}\nclass Widget:\n    def __init__(self, value):\n        self.value = value\nmain()`,
  ]) {
    await fill(classCode);
    const point = await hover(classCode.indexOf('Widget(3)'));
    await page.mouse.dblclick(point.x, point.y);
    const definition = classCode.indexOf('class Widget') + 'class '.length;
    assert.deepEqual(await page.locator('#source').evaluate((element) => [element.selectionStart, element.selectionEnd]), [definition, definition], 'constructors must navigate to the class name, not __init__ or an enclosing function');
    const visible = await page.locator('#source').evaluate((element, definition) => {
      const row = element.value.slice(0, definition).split('\n').length - 1;
      const style = getComputedStyle(element);
      const top = parseFloat(style.paddingTop) + row * parseFloat(style.lineHeight) - element.scrollTop;
      return top >= 0 && top < element.clientHeight;
    }, definition);
    assert(visible, 'the class declaration must be scrolled into view');
    assert.equal(await page.locator('#source').inputValue(), classCode);
  }
  for (const [text, name] of [
    ['print("hello")', 'print'],
    ['list([1, 2])', 'list'],
    ['import pygame as pg\npg.Surface((10, 10))', 'Surface'],
    ['class Widget: pass\nWidget = other\nWidget()', 'Widget'],
    ['import pygame as pg\npg.image.load("x")', 'load'],
    ['unknown(3)', 'unknown'],
  ]) {
    await fill(text);
    const point = await hover(text.lastIndexOf(name));
    await page.mouse.dblclick(point.x, point.y);
    assert.equal(await page.locator('#source').evaluate((element) => element.value.slice(element.selectionStart, element.selectionEnd)), name, 'non-user functions should retain ordinary word selection');
  }
  await page.locator('#syntax-toggle').click();
  const plainNavigation = 'def plain():\n    pass\nplain()';
  await fill(plainNavigation);
  const plainPoint = await hover(plainNavigation.lastIndexOf('plain'));
  await page.mouse.dblclick(plainPoint.x, plainPoint.y);
  assert.deepEqual(await page.locator('#source').evaluate((element) => [element.selectionStart, element.selectionEnd]), [4, 4], 'navigation must also work with highlighting disabled');
  await page.locator('#syntax-toggle').click();
  // Dispatch input and double-click in one task: no debounce timeout can run
  // before the navigation handler rebuilds its stale index.
  await page.locator('#source').evaluate((element) => {
    element.value = 'def fresh():\n    pass\nfresh()';
    element.dispatchEvent(new Event('input', { bubbles: true }));
    element.scrollTop = element.scrollLeft = 0;
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    element.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, clientX: rect.left + parseFloat(style.paddingLeft) + 4, clientY: rect.top + parseFloat(style.paddingTop) + 2.5 * parseFloat(style.lineHeight) }));
  });
  assert.deepEqual(await page.locator('#source').evaluate((element) => [element.selectionStart, element.selectionEnd]), [4, 4], 'navigation must work before analysis debounce completes');
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
  console.log('Function hover and definition navigation browser tests passed');
} finally {
  await browser.close();
}
