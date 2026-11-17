// Serve built web/ first. Optional: BASE_URL, PLAYWRIGHT_MODULE, CHROMIUM_EXECUTABLE.
import assert from 'node:assert/strict';
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? 'playwright');
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_EXECUTABLE, headless: true, args: ['--no-sandbox', '--enable-unsafe-swiftshader'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
const errors = [];
page.on('pageerror', (error) => errors.push(error.message));
const url = process.env.BASE_URL ?? 'http://localhost:8000';
const state = (page, value) => page.waitForFunction((value) => document.querySelector('#source').dataset.syntaxState === value, value, { timeout: 60_000 });
try {
  await page.addInitScript(() => {
    window.syntaxRequests = [];
    const NativeWorker = window.Worker;
    window.Worker = class extends NativeWorker {
      postMessage(data, ...args) {
        if (typeof data?.source === 'string') window.syntaxRequests.push(data);
        return super.postMessage(data, ...args);
      }
    };
  });
  await page.goto(url);
  await state(page, 'valid');
  assert.equal(await page.locator('#gutter .syntax-error').count(), 0);
  assert.equal(await page.locator('#console').textContent(), '');
  await page.locator('#source').fill('x = 1\ndef broken(:\n    pass');
  await state(page, 'error');
  const marker = page.locator('#gutter button[data-line="2"]');
  assert.match(await marker.getAttribute('class'), /syntax-error/);
  assert.match(await marker.getAttribute('title'), /SyntaxError: invalid syntax/);
  assert.match(await marker.getAttribute('aria-label'), /SyntaxError/);
  await marker.click();
  assert.match(await marker.getAttribute('class'), /breakpoint/);
  assert.match(await marker.getAttribute('class'), /syntax-error/, 'syntax markers must coexist with breakpoints');
  await page.locator('#source').fill('x = 1\ny = 2');
  assert.equal(await page.locator('#gutter .syntax-error').count(), 0, 'stale markers clear as soon as source changes');
  await state(page, 'valid');
  assert.match(await page.locator('#gutter button[data-line="2"]').getAttribute('class'), /breakpoint/);
  await page.locator('#source').fill('x = 1\nreturn 3');
  await state(page, 'error');
  assert.match(await page.locator('#syntax-status').textContent(), /outside function/);
  await page.locator('#source').fill('raise ValueError("not executed by checker")');
  await state(page, 'valid');
  assert.equal(await page.locator('#console').textContent(), '', 'checking must never execute the program');
  const requests = await page.evaluate(() => window.syntaxRequests.length);
  await page.locator('#source').evaluate(async (source) => {
    for (const text of ['x = (', 'x = [', 'x = "("']) {
      source.value = text;
      source.dispatchEvent(new Event('input', { bubbles: true }));
      await new Promise((resolve) => setTimeout(resolve, 40));
    }
  });
  assert.equal(await page.evaluate(() => window.syntaxRequests.length), requests, 'checks should be debounced');
  await state(page, 'valid');
  assert.equal(await page.evaluate(() => window.syntaxRequests.length), requests + 1);

  // Editing diagnostics are independent of a running interpreter.
  await page.locator('#source').fill('print("looping")\nwhile True:\n    pass');
  await page.locator('#run').click();
  await page.waitForFunction(() => document.querySelector('#console').textContent.includes('looping'), null, { timeout: 60_000 });
  await page.locator('#source').fill('def broken(:');
  await state(page, 'error');
  assert.equal(await page.locator('#status').textContent(), 'Running');
  assert.equal(await page.locator('#viewport iframe').count(), 1);
  await page.locator('#stop').click();
  await page.locator('#reset').click();
  await state(page, 'valid');
  assert.equal(await page.locator('#gutter .syntax-error').count(), 0, 'example replacement must refresh diagnostics');

  // Explicitly deliver out-of-order responses to test stale-result protection.
  const stale = await browser.newPage();
  await stale.addInitScript(() => {
    window.syntaxWorkers = [];
    window.Worker = class extends EventTarget {
      constructor() { super(); this.requests = []; window.syntaxWorkers.push(this); }
      postMessage(data) { this.requests.push(data); }
      terminate() {}
      respond(data) { this.dispatchEvent(new MessageEvent('message', { data })); }
    };
  });
  await stale.goto(url);
  await stale.waitForFunction(() => window.syntaxWorkers[0]?.requests.length === 1);
  await stale.locator('#source').fill('x = 2');
  await stale.waitForFunction(() => window.syntaxWorkers[0]?.requests.length === 2);
  await stale.evaluate(() => {
    const worker = window.syntaxWorkers[0];
    worker.respond({ id: worker.requests[0].id, diagnostics: [{ line: 1, column: 1, message: 'stale error' }] });
  });
  assert.equal(await stale.locator('#gutter .syntax-error').count(), 0);
  assert.equal(await stale.locator('#source').getAttribute('data-syntax-state'), 'checking');
  await stale.evaluate(() => {
    const worker = window.syntaxWorkers[0];
    worker.respond({ id: worker.requests[1].id, diagnostics: [] });
  });
  await state(stale, 'valid');
  await stale.close();

  const unavailable = await browser.newPage();
  await unavailable.route('**/syntax-check.wasm', (route) => route.fulfill({ status: 404, body: 'not built' }));
  await unavailable.goto(url);
  await state(unavailable, 'unavailable');
  assert.match(await unavailable.locator('#syntax-status').textContent(), /build.sh/);
  assert.equal(await unavailable.locator('#gutter .syntax-error').count(), 0, 'a missing checker is not a syntax error');
  await unavailable.locator('#source').fill('print("still works")');
  await unavailable.locator('#run').click();
  await unavailable.waitForFunction(() => document.querySelector('#status').textContent === 'Finished', null, { timeout: 60_000 });
  assert.equal(await unavailable.locator('#console').textContent(), 'still works\n');
  await unavailable.close();
  assert.deepEqual(errors, []);
  console.log('Gutter syntax diagnostics browser tests passed');
} finally {
  await browser.close();
}
