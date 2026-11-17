// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const widths = () => page.evaluate(() => [document.querySelector('.editor-pane').getBoundingClientRect().width, document.querySelector('.result-pane').getBoundingClientRect().width]);
try {
  await page.goto(process.env.BASE_URL ?? 'http://localhost:8000');
  await page.locator('#run').click();
  await page.frameLocator('#viewport iframe').locator('canvas').waitFor({ timeout: 60_000 });
  const initial = await widths();
  const divider = page.getByRole('separator', { name: 'Resize editor and result panes' });
  const box = await divider.boundingBox();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + 180, box.y + box.height / 2, { steps: 12 });
  await page.mouse.up();
  const dragged = await widths();
  assert(dragged[0] > initial[0] + 170, 'drag right should widen the editor');
  assert(dragged[1] < initial[1] - 170, 'drag right should shrink the result pane');
  assert.equal(await page.locator('#status').textContent(), 'Running', 'resizing must not restart the runtime');
  assert.equal(await page.locator('body').evaluate((body) => body.classList.contains('resizing-panes')), false);
  await divider.press('ArrowLeft');
  assert((await widths())[0] < dragged[0]);
  await divider.press('Home');
  assert(Math.abs((await widths())[0] - 280) < 1, 'editor has a minimum width');
  await divider.press('End');
  assert(Math.abs((await widths())[1] - 340) < 1, 'result pane has a minimum width');
  await divider.press('ArrowLeft');
  const saved = await widths();
  await page.reload();
  await divider.waitFor();
  assert(Math.abs((await widths())[0] - saved[0]) < 1, 'split should survive reload');
  await divider.dblclick();
  assert(Math.abs((await widths())[0] - initial[0]) < 1, 'double-click should reset split');
  await page.setViewportSize({ width: 600, height: 900 });
  assert.equal(await divider.isVisible(), false, 'mobile keeps its stacked layout');
  const mobile = await page.evaluate(() => {
    const editor = document.querySelector('.editor-pane').getBoundingClientRect();
    const result = document.querySelector('.result-pane').getBoundingClientRect();
    return { editorBottom: editor.bottom, resultTop: result.top, scrollWidth: document.body.scrollWidth };
  });
  assert(mobile.resultTop >= mobile.editorBottom);
  assert(mobile.scrollWidth <= 600, 'mobile layout should not overflow horizontally');
  await page.setViewportSize({ width: 1400, height: 900 });
  assert.equal(await divider.isVisible(), true);
  assert.deepEqual(errors, []);
  console.log('Pane splitter browser tests passed');
} finally {
  await browser.close();
}
