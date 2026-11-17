const source = document.querySelector('#source');
const status = document.querySelector('#syntax-status');
let worker = null;
let debounce = null;
let watchdog = null;
let revision = 0;
let diagnostics = [];
let textForRevision = '';

function publish(state, items, message) {
  source.dataset.syntaxState = state;
  status.dataset.state = state;
  if (status.textContent !== message) status.textContent = message;
  status.title = message;
  const changed = diagnostics.length || items.length;
  diagnostics = items;
  if (changed) source.dispatchEvent(new CustomEvent('syntax-diagnostics', { detail: { source: source.value, diagnostics } }));
}
function fail(message) {
  clearTimeout(watchdog);
  worker?.terminate();
  worker = null;
  publish('unavailable', [], `Syntax checks unavailable: ${message}`);
}
function check(id, text) {
  if (id !== revision || text !== source.value) return;
  try {
    if (!worker) {
      const instance = worker = new Worker(new URL('./syntax-worker.js', import.meta.url));
      instance.addEventListener('message', ({ data }) => {
        if (instance !== worker || data.id !== revision || textForRevision !== source.value) return;
        clearTimeout(watchdog);
        if (data.error) { fail(data.error); return; }
        if (!Array.isArray(data.diagnostics)) { fail('Invalid compiler response'); return; }
        const items = data.diagnostics;
        const first = items[0];
        publish(first ? 'error' : 'valid', items, first ? `Line ${first.line}, column ${first.column}: ${first.message}` : 'Syntax OK');
      });
      instance.addEventListener('error', (event) => {
        event.preventDefault();
        if (instance === worker) fail(event.message || 'Background compiler failed; try rebuilding with ./web/build.sh.');
      });
      instance.addEventListener('messageerror', () => { if (instance === worker) fail('Invalid compiler response'); });
    }
    textForRevision = text;
    worker.postMessage({ id, source: text });
    watchdog = setTimeout(() => { if (id === revision) fail('Compiler timed out'); }, 20000);
  } catch (error) { fail(String(error?.message ?? error)); }
}
export function refreshSyntaxErrors() {
  const id = ++revision;
  clearTimeout(debounce);
  clearTimeout(watchdog);
  publish('checking', [], 'Checking syntax…');
  const text = source.value;
  debounce = setTimeout(() => check(id, text), 300);
}
source.addEventListener('input', refreshSyntaxErrors);
