// Serve web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const ready = () => page.waitForFunction(() => !document.querySelector('.source-editor').classList.contains('syntax-pending'));
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  const source = page.locator('#source');
  const toggle = page.getByRole('button', { name: 'Syntax highlighting', exact: true });
  assert.equal(await toggle.getAttribute('aria-pressed'), 'false');
  const sample = `# comment\ndef greet(name):\n    count = 8.2\n    value = [1, 0xFF, True]\n    text = "<img src=x onerror=alert(1)>"\n    multi = '''hello\nworld'''\n    return print(name, text)\n`;
  await source.fill(sample);
  await toggle.click();
  assert.equal(await toggle.getAttribute('aria-pressed'), 'true');
  assert.equal(await page.locator('#syntax-code').textContent(), sample + '\n');
  for (const kind of ['keyword', 'identifier', 'function', 'builtin', 'string', 'number', 'constant', 'comment', 'operator']) {
    assert(await page.locator(`.syntax-${kind}`).count() > 0, `${kind} should be colored`);
  }
  assert.equal(await page.locator('#syntax-code img').count(), 0, 'source must never be interpreted as HTML');
  assert.equal(await source.inputValue(), sample, 'highlighting must not change source');

  // Several input events should cause just one recolor after typing stops.
  await page.evaluate(async () => {
    const source = document.querySelector('#source');
    window.highlightRenders = 0;
    window.highlightObserver = new MutationObserver((records) => {
      if (records.some((record) => [...record.addedNodes].some((node) => node.nodeType === Node.ELEMENT_NODE))) window.highlightRenders++;
    });
    window.highlightObserver.observe(document.querySelector('#syntax-code'), { childList: true });
    for (const value of ['x = 1', 'x = 12', 'x = 123']) {
      source.value = value;
      source.dispatchEvent(new Event('input', { bubbles: true }));
      await new Promise((resolve) => setTimeout(resolve, 40));
    }
  });
  assert.equal(await page.evaluate(() => window.highlightRenders), 0, 'recoloring should be debounced');
  await ready();
  assert.equal(await page.evaluate(() => window.highlightRenders), 1);
  assert.equal(await page.locator('#syntax-code').textContent(), 'x = 123\n');
  await page.evaluate(() => window.highlightObserver.disconnect());

  // Earlier spans must stay colored and retain their DOM identity both while
  // an edit is pending and after the affected tail has been recolored.
  await source.fill(sample);
  await ready();
  const prefix = await page.evaluate(() => {
    const nodes = [...document.querySelectorAll('#syntax-code span')];
    window.stableTokens = nodes.slice(0, 6);
    window.stableSuffix = nodes.slice(nodes.findIndex((node) => node.textContent === '8.2') + 1);
    window.suffixColors = window.stableSuffix.map((node) => [node.textContent, getComputedStyle(node).color]);
    return window.stableTokens.map((node) => [node.textContent, getComputedStyle(node).color]);
  });
  await source.fill(sample.replace('8.2', '199.25'));
  assert.equal(await page.locator('#syntax-code').textContent(), sample.replace('8.2', '199.25') + '\n', 'edited text should appear immediately');
  const checkPrefix = async () => {
    const result = await page.evaluate(() => ({
      connected: window.stableTokens.every((node) => node.isConnected),
      colors: window.stableTokens.map((node) => [node.textContent, getComputedStyle(node).color]),
      visible: getComputedStyle(document.querySelector('#syntax-layer')).visibility,
      textareaColor: getComputedStyle(document.querySelector('#source')).color,
      suffixConnected: window.stableSuffix.every((node) => node.isConnected),
      suffixColors: window.stableSuffix.map((node) => [node.textContent, getComputedStyle(node).color]),
      originalSuffixColors: window.suffixColors,
    }));
    assert(result.connected, 'unchanged prefix spans must not be recreated');
    assert.deepEqual(result.colors, prefix);
    assert(result.suffixConnected, 'unchanged suffix spans must not be recreated');
    assert.deepEqual(result.suffixColors, result.originalSuffixColors, 'unchanged suffix must keep its colors');
    assert.equal(result.visible, 'visible');
    assert.equal(result.textareaColor, 'rgba(0, 0, 0, 0)', 'typing must not switch the whole editor to white text');
  };
  await checkPrefix();
  await ready();
  await checkPrefix();
  // More edits before the debounce fires must keep shifting the same suffix,
  // not discard it or leave its text at an old position.
  await source.evaluate((element, sample) => {
    for (const number of ['12345', '7', '-8.25']) {
      element.value = sample.replace('8.2', number);
      element.dispatchEvent(new Event('input', { bubbles: true }));
    }
  }, sample);
  assert.equal(await page.locator('#syntax-code').textContent(), sample.replace('8.2', '-8.25') + '\n');
  await checkPrefix();
  await ready();
  await checkPrefix();

  // Incremental lexing must agree with a full recolor, even when edits alter
  // multiline-string state, token boundaries, and function-name context.
  const coloredTokens = () => page.locator('#syntax-code span').evaluateAll((nodes) => nodes.map((node) => [node.className, node.textContent]));
  for (const [before, after] of [
    ['head = 1\ntext = """first\nsecond\n"""\nend = 2', 'head = 1\ntext = ""first\nsecond\n"""\nend = 2'],
    ['def name(x):\n    return x', 'def renamed(x):\n    return x'],
    ['x = 1', 'x = 12.5'],
    ['x = ""', 'x = """'],
    ['a = 1\nb = 2\nc = 3', 'a = 1\nc = 3'],
    ['head = 1\nvalue = 2', 'head = 1\n# value = 2'],
  ]) {
    await source.fill(before);
    await ready();
    await source.fill(after);
    await ready();
    const incremental = await coloredTokens();
    await toggle.click();
    await toggle.click();
    assert.deepEqual(await coloredTokens(), incremental, `incremental coloring differs for ${after}`);
  }

  // Horizontal/vertical scrolling and layout resizing must keep both layers aligned.
  await source.fill(Array.from({ length: 100 }, (_, i) => `line_${i} = "${'x'.repeat(200)}"`).join('\n'));
  await ready();
  await source.evaluate((element) => { element.scrollTop = 500; element.scrollLeft = 300; element.dispatchEvent(new Event('scroll')); });
  const alignment = () => page.evaluate(() => {
    const source = document.querySelector('#source');
    const layer = document.querySelector('#syntax-layer');
    return { top: Math.abs(source.scrollTop - layer.scrollTop), left: Math.abs(source.scrollLeft - layer.scrollLeft), width: Math.abs(source.clientWidth - layer.clientWidth), height: Math.abs(source.clientHeight - layer.clientHeight) };
  });
  for (const difference of Object.values(await alignment())) assert(difference < 1, 'overlay must match textarea geometry and scroll');
  await page.getByRole('separator').press('ArrowRight');
  await page.waitForTimeout(50);
  for (const difference of Object.values(await alignment())) assert(difference < 1, 'resizing must preserve alignment');

  await source.fill('x = 1\n');
  await source.press('End');
  await source.press('Tab');
  await ready();
  assert.equal(await page.locator('#syntax-code').textContent(), (await source.inputValue()) + '\n', 'Tab insertion must refresh colors');
  await source.fill('print("highlighted", 8.2)');
  await ready();
  await page.locator('#run').click();
  await page.waitForFunction(() => document.querySelector('#status').textContent === 'Finished', null, { timeout: 60_000 });
  assert.equal(await page.locator('#console').textContent(), 'highlighted 8.2\n');
  await page.reload();
  assert.equal(await toggle.getAttribute('aria-pressed'), 'true', 'preference should be saved');
  assert.equal(await page.locator('#syntax-code').textContent(), (await source.inputValue()) + '\n');
  await page.locator('#reset').click();
  await page.waitForFunction(() => document.querySelector('#source').value.startsWith('import pygame'));
  assert.equal(await page.locator('#syntax-code').textContent(), (await source.inputValue()) + '\n', 'example changes must refresh highlighting');
  await toggle.click();
  assert.equal(await page.locator('#syntax-layer').isVisible(), false);
  assert.equal(await page.locator('#syntax-code span').count(), 0);
  await page.reload();
  assert.equal(await toggle.getAttribute('aria-pressed'), 'false');
  assert.deepEqual(errors, []);
  console.log('Syntax highlighting browser tests passed');
} finally {
  await browser.close();
}
