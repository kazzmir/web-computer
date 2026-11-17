// A separate Go/WASM compiler: syntax checks never execute Python or share the
// running program's interpreter, canvas, globals, or browser main thread.
let initialized;
async function initialize() {
  importScripts('wasm_exec.js');
  let resolveReady, rejectReady;
  const ready = new Promise((resolve, reject) => { resolveReady = resolve; rejectReady = reject; });
  globalThis.syntaxCheckerReady = resolveReady;
  const go = new Go();
  go.argv = ['syntax-check.wasm'];
  const response = await fetch('syntax-check.wasm');
  if (!response.ok) throw new Error('Cannot load syntax-check.wasm; run ./web/build.sh.');
  const { instance } = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  go.run(instance).catch(rejectReady);
  await ready;
}
self.addEventListener('message', async ({ data }) => {
  if (!data || typeof data.source !== 'string') return;
  try {
    initialized ??= initialize();
    await initialized;
    const result = globalThis.checkPythonSyntax(data.source);
    self.postMessage({ id: data.id, ...result });
  } catch (error) {
    self.postMessage({ id: data.id, error: String(error?.message ?? error) });
  }
});
