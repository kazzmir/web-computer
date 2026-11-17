// npm install --no-save --package-lock=false playwright
// Serve built web/ first; optionally set BASE_URL and PLAYWRIGHT_MODULE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const status = (text) => page.waitForFunction((text) => document.querySelector('#status').textContent === text, text, { timeout: 60_000 });
const breakpoint = (line) => page.locator(`#gutter button[data-line="${line}"]`).click();
const step = async (command, line) => {
  await page.locator(`[data-debug-command="${command}"]`).click();
  await status(`Paused at line ${line}`);
};
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await page.locator('#source').fill('def add(x):\n    y = x + 1\n    return y\na = 10\nb = add(a)\nc = add(b)\nprint(c)');
  await breakpoint(5);
  await page.locator('#debug').click();
  await status('Paused at line 5');
  assert.match(await page.locator('#locals').textContent(), /a = 10/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /b =/);
  assert.equal(await page.locator('#console').textContent(), '');
  assert.equal(await page.locator('#source').getAttribute('readonly'), '');
  await step('into', 2);
  assert.match(await page.locator('#locals').textContent(), /x = 10/);
  assert.doesNotMatch(await page.locator('#locals').textContent(), /y =/);
  await step('over', 3);
  assert.match(await page.locator('#locals').textContent(), /y = 11/);
  await step('over', 6);
  assert.match(await page.locator('#locals').textContent(), /b = 11/);
  await step('over', 7);
  assert.match(await page.locator('#locals').textContent(), /c = 12/);
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '12\n');

  // A paused interpreter must keep presenting the last flip, not the
  // subsequently modified display surface or Ebitengine's cleared screen.
  await breakpoint(5);
  await page.locator('#source').fill('import pygame\ns = pygame.display.set_mode((320, 240))\ns.fill((210, 40, 80))\npygame.display.flip()\ns.fill((0, 0, 0))\nprint("done")');
  await breakpoint(6);
  await page.locator('#debug').click();
  await status('Paused at line 6');
  const canvas = page.frameLocator('#viewport iframe').locator('canvas');
  const centerPixel = async () => {
    const png = await canvas.screenshot();
    return page.evaluate(async (png) => {
      const image = new Image();
      image.src = `data:image/png;base64,${png}`;
      await image.decode();
      const target = document.createElement('canvas');
      target.width = image.width; target.height = image.height;
      const context = target.getContext('2d');
      context.drawImage(image, 0, 0);
      return Array.from(context.getImageData(image.width >> 1, image.height >> 1, 1, 1).data);
    }, png.toString('base64'));
  };
  await page.waitForTimeout(300);
  assert.deepEqual(await centerPixel(), [210, 40, 80, 255], 'paused frame retains the last flip');
  await page.waitForTimeout(300);
  assert.deepEqual(await centerPixel(), [210, 40, 80, 255], 'idle draws must not clear the frame');
  await page.locator('[data-debug-command="continue"]').click();
  await status('Finished');
  assert.deepEqual(await centerPixel(), [210, 40, 80, 255], 'unflipped changes stay invisible after exit');

  // Repeated hits, live breakpoint removal/addition, and stop while paused.
  await page.locator('#source').fill('i = 0\nwhile i < 3:\n    i += 1\nprint(i)');
  await breakpoint(3);
  await page.locator('#debug').click();
  await status('Paused at line 3');
  assert.match(await page.locator('#locals').textContent(), /i = 0/);
  await step('continue', 3);
  assert.match(await page.locator('#locals').textContent(), /i = 1/);
  await breakpoint(3);
  await breakpoint(4);
  await step('continue', 4);
  assert.match(await page.locator('#locals').textContent(), /i = 3/);
  await page.locator('#debug-stop').click();
  await status('Stopped');
  assert.equal(await page.locator('#viewport iframe').count(), 0);
  await page.locator('#run').click();
  await status('Finished');
  assert.equal(await page.locator('#console').textContent(), '3\n');
  assert.equal(await page.locator('#debugger').isVisible(), false);
  // Locals must scroll rather than resize the viewport while stepping.
  await breakpoint(4);
  await page.locator('#source').fill([
    'def slow():',
    '    for i in range(100000):',
    '        pass',
    ...Array.from({ length: 12 }, (_, i) => `a${i} = ${i}`),
    'slow()',
    'print("done")',
  ].join('\n'));
  await breakpoint(16);
  await page.locator('#debug').click();
  await status('Paused at line 16');
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  await page.evaluate(() => {
    const viewport = document.querySelector('#viewport');
    window.debugViewportSizes = [];
    const record = () => {
      const { width, height } = viewport.getBoundingClientRect();
      window.debugViewportSizes.push([width, height]);
    };
    record();
    window.debugResizeObserver = new ResizeObserver(record);
    window.debugResizeObserver.observe(viewport);
  });
  await step('over', 17);
  await page.waitForTimeout(100);
  const sizes = await page.evaluate(() => {
    window.debugResizeObserver.disconnect();
    return window.debugViewportSizes;
  });
  assert(sizes.length > 0);
  for (const size of sizes) assert.deepEqual(size, sizes[0], 'stepping must not resize the result window');
  assert(await page.locator('#locals').evaluate((element) => element.scrollHeight > element.clientHeight), 'large locals lists should scroll');
  await page.locator('#debug-stop').click();
  assert.deepEqual(errors, []);
  console.log('Debugger browser tests passed');
} finally {
  await browser.close();
}
