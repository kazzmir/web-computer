import { runtimeLifecycle } from './lifecycle.js';

// Each run owns its Go runtime, DOM, canvas, event handlers, and audio state.
let started = false;
let exited = false;
const send = (type, fields = {}) => parent.postMessage({ channel: 'web-computer', type, ...fields }, location.origin);
const fail = (error) => { if (!exited) send('error', { text: String(error?.message ?? error) }); };
window.addEventListener('error', (event) => fail(event.error ?? event.message));
window.addEventListener('unhandledrejection', (event) => fail(event.reason));
window.addEventListener('message', async (event) => {
  if (event.source !== parent || event.origin !== location.origin || started) return;
  const message = event.data;
  if (!message || message.channel !== 'web-computer' || message.type !== 'run' || typeof message.source !== 'string') return;
  started = true;
  const decoders = new Map([[1, new TextDecoder()], [2, new TextDecoder()]]);
  const flush = () => {
    for (const decoder of decoders.values()) {
      const text = decoder.decode();
      if (text) send('output', { text });
    }
  };
  let cleanup = () => {};
  try {
    if (typeof Go !== 'function') throw new Error('wasm_exec.js is missing. Run ./web/build.sh first.');
    // Capture bytes rather than console lines: print(..., end="") and split
    // UTF-8 writes must work too. wasm_exec.js supplies the browser fs shim.
    const originalWrite = globalThis.fs.writeSync;
    globalThis.fs.writeSync = (fd, bytes) => {
      const decoder = decoders.get(fd);
      if (!decoder) return originalWrite(fd, bytes);
      const text = decoder.decode(bytes, { stream: true });
      if (text) send('output', { text });
      return bytes.length;
    };
    globalThis.webComputerSource = message.source;
    cleanup = runtimeLifecycle();
    const go = new Go();
    go.argv = ['web-computer.wasm', 'editor.py'];
    go.exit = (code) => { flush(); exited = true; cleanup(); send('exit', { code }); };
    const response = await fetch('web-computer.wasm');
    if (!response.ok) throw new Error(`Cannot load web-computer.wasm (${response.status}). Run ./web/build.sh first.`);
    // The bytes fallback also supports servers without application/wasm MIME.
    let result;
    if (WebAssembly.instantiateStreaming) {
      try { result = await WebAssembly.instantiateStreaming(response.clone(), go.importObject); }
      catch { result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject); }
    } else {
      result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
    }
    const focusCanvas = () => {
      const canvas = document.querySelector('canvas');
      if (canvas) { canvas.focus(); observer.disconnect(); }
    };
    const observer = new MutationObserver(focusCanvas);
    observer.observe(document.body, { childList: true, subtree: true });
    send('running');
    await go.run(result.instance);
    observer.disconnect();
    if (!exited) { flush(); exited = true; cleanup(); send('exit', { code: 0 }); }
  } catch (error) {
    cleanup();
    flush();
    fail(error);
  }
});
if (parent !== window) send('ready');
