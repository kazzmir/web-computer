// Optional: npm install --no-save --package-lock=false playwright
// Run with the built web/ directory served at BASE_URL (default localhost:8000).
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const waitStatus = (text) => page.locator('#status').filter({ hasText: text }).waitFor({ timeout: 60_000 });
const run = async (code) => { await page.locator('#source').fill(code); await page.locator('#run').click(); };
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await page.locator('#run').click();
  await waitStatus('Running');
  await page.waitForFunction(() => document.querySelector('#console').textContent.includes('Hello!'), null, { timeout: 60_000 });
  const canvas = page.frameLocator('#viewport iframe').locator('canvas');
  await canvas.waitFor();
  await page.waitForTimeout(500);
  const first = await canvas.screenshot();
  await page.waitForTimeout(180);
  assert(!first.equals(await canvas.screenshot()), 'canvas must animate');
  await page.locator('#stop').click();
  await waitStatus('Stopped');
  assert.equal(await page.locator('#viewport iframe').count(), 0);
  await run('import pygame\ns = pygame.display.set_mode((320, 240))\ns.fill((210, 40, 80))\npygame.display.flip()');
  await waitStatus('Finished');
  const snapshot = await page.frameLocator('#viewport iframe').locator('canvas').screenshot();
  const pixel = await page.evaluate(async (png) => {
    const image = new Image();
    image.src = `data:image/png;base64,${png}`;
    await image.decode();
    const canvas = document.createElement('canvas');
    canvas.width = image.width; canvas.height = image.height;
    const context = canvas.getContext('2d');
    context.drawImage(image, 0, 0);
    return Array.from(context.getImageData(image.width >> 1, image.height >> 1, 1, 1).data);
  }, snapshot.toString('base64'));
  assert.deepEqual(pixel, [210, 40, 80, 255], 'a one-flip program must retain its final frame');
  await run('print("hello λ", end="!")');
  await waitStatus('Finished');
  assert.equal(await page.locator('#console').textContent(), 'hello λ!');
  await run('def broken(:');
  await waitStatus('Exited (1)');
  assert.match(await page.locator('#console').textContent(), /SyntaxError/);
  await run('print(1 / 0)');
  await waitStatus('Exited (1)');
  assert.match(await page.locator('#console').textContent(), /ZeroDivisionError/);
  await run('x = 42\nprint(x)');
  await waitStatus('Finished');
  await run('print(x)');
  await waitStatus('Exited (1)');
  assert.match(await page.locator('#console').textContent(), /NameError/);
  await run('print("looping")\nwhile True:\n    pass');
  await page.waitForFunction(() => document.querySelector('#console').textContent.includes('looping'), null, { timeout: 60_000 });
  await page.locator('#stop').click({ timeout: 5000 });
  await waitStatus('Stopped');
  await run('print("after stop")');
  await waitStatus('Finished');
  assert.match(await page.locator('#console').textContent(), /after stop/);
  await page.reload();
  assert.equal(await page.locator('#source').inputValue(), 'print("after stop")');
  assert.deepEqual(errors, [], 'no callbacks should reenter an exited Go runtime');
  console.log('PASS: graphics, output, errors, reruns, Stop, and saved editor');
} finally {
  await browser.close();
}
