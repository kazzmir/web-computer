const windows = document.querySelector('#result-windows');
const runtime = document.querySelector('#viewport');
const debuggerPane = document.querySelector('#debugger');
const debuggerHeading = debuggerPane.querySelector('.console-heading');
const consolePane = document.querySelector('#console-pane');
const evaluator = document.querySelector('#debug-evaluator');
const upperDivider = document.querySelector('#runtime-debug-divider');
const lowerDivider = document.querySelector('#console-divider');
const storageKey = 'web-computer.result-splits';
const defaults = { normal: [.72, .28], debug: [.48, .28, .24] };
let sizes = { normal: [...defaults.normal], debug: [...defaults.debug] };
let drag = null;
try {
  const saved = JSON.parse(localStorage.getItem(storageKey));
  for (const mode of ['normal', 'debug']) {
    const values = saved?.[mode];
    if (Array.isArray(values) && values.length === defaults[mode].length && values.every((value) => Number.isFinite(value) && value > 0)) sizes[mode] = values;
  }
} catch { /* Disabled storage or invalid saved layout. */ }

function mode() { return debuggerPane.hidden ? 'normal' : 'debug'; }
function panes() { return mode() === 'debug' ? [runtime, debuggerPane, consolePane] : [runtime, consolePane]; }
function minimum(pane) {
  if (pane === runtime) return 100;
  if (pane === consolePane) return Math.max(80, Math.ceil(consolePane.querySelector('.console-heading').getBoundingClientRect().height + evaluator.getBoundingClientRect().height) + 40);
  return Math.ceil(debuggerHeading.getBoundingClientRect().height) + 60;
}
function pair(divider) { return divider === upperDivider ? 0 : panes().length - 2; }
function save() {
  try { localStorage.setItem(storageKey, JSON.stringify(sizes)); } catch { /* Storage may be disabled. */ }
}
function updateAccessibility() {
  const items = panes();
  for (const divider of [upperDivider, lowerDivider]) {
    if (divider.hidden) continue;
    const index = pair(divider);
    const upper = items[index];
    const lower = items[index + 1];
    const upperHeight = upper.getBoundingClientRect().height;
    const total = upperHeight + lower.getBoundingClientRect().height;
    const usable = total >= minimum(upper) + minimum(lower);
    const upperName = upper === runtime ? 'Runtime' : 'Debugger';
    const lowerName = lower === debuggerPane ? 'debugger' : 'console';
    const percent = total ? Math.round(upperHeight / total * 100) : 50;
    divider.setAttribute('aria-label', `Resize ${upperName.toLowerCase()} and ${lowerName}`);
    divider.setAttribute('aria-controls', `${upper.id} ${lower.id}`);
    divider.setAttribute('aria-disabled', String(!usable));
    divider.setAttribute('aria-valuemin', usable ? Math.floor(minimum(upper) / total * 100) : 0);
    divider.setAttribute('aria-valuemax', usable ? Math.ceil((total - minimum(lower)) / total * 100) : 100);
    divider.setAttribute('aria-valuenow', percent);
    divider.setAttribute('aria-valuetext', `${upperName} ${percent}%, ${lowerName} ${100 - percent}%`);
  }
}
function apply() {
  const debug = mode() === 'debug';
  windows.classList.toggle('with-debugger', debug);
  upperDivider.hidden = !debug;
  windows.style.setProperty('--debugger-min', `${minimum(debuggerPane)}px`);
  const weights = sizes[mode()];
  windows.style.setProperty('--runtime-height', `${weights[0]}fr`);
  windows.style.setProperty('--console-height', `${weights.at(-1)}fr`);
  if (debug) windows.style.setProperty('--debugger-height', `${weights[1]}fr`);
  updateAccessibility();
}
function resize(divider, desired, original = panes().map((pane) => pane.getBoundingClientRect().height)) {
  const items = panes();
  const index = pair(divider);
  const total = original[index] + original[index + 1];
  if (total < minimum(items[index]) + minimum(items[index + 1])) return;
  const upper = Math.max(minimum(items[index]), Math.min(total - minimum(items[index + 1]), desired));
  const heights = [...original];
  heights[index] = upper;
  heights[index + 1] = total - upper;
  const sum = heights.reduce((a, b) => a + b, 0);
  sizes[mode()] = heights.map((height) => height / sum);
  apply();
}
function finish(event) {
  if (!drag || event.pointerId !== drag.pointerId) return;
  const divider = drag.divider;
  drag = null;
  document.body.classList.remove('resizing-results');
  if (divider.hasPointerCapture(event.pointerId)) divider.releasePointerCapture(event.pointerId);
  save();
}
for (const divider of [upperDivider, lowerDivider]) {
  divider.addEventListener('pointerdown', (event) => {
    if (divider.hidden || !event.isPrimary || event.button !== 0 || drag || divider.getAttribute('aria-disabled') === 'true') return;
    event.preventDefault();
    divider.focus();
    drag = { divider, y: event.clientY, heights: panes().map((pane) => pane.getBoundingClientRect().height), mode: mode() };
    divider.setPointerCapture(event.pointerId);
    drag.pointerId = event.pointerId;
    document.body.classList.add('resizing-results');
  });
  divider.addEventListener('pointermove', (event) => {
    if (!drag || drag.divider !== divider || drag.pointerId !== event.pointerId || drag.mode !== mode()) return;
    resize(divider, drag.heights[pair(divider)] + event.clientY - drag.y, drag.heights);
  });
  for (const type of ['pointerup', 'pointercancel', 'lostpointercapture']) divider.addEventListener(type, finish);
  divider.addEventListener('keydown', (event) => {
    if (divider.hidden) return;
    const items = panes();
    const index = pair(divider);
    const current = items[index].getBoundingClientRect().height;
    const total = current + items[index + 1].getBoundingClientRect().height;
    const step = event.shiftKey ? 50 : 10;
    const targets = { ArrowUp: current - step, ArrowDown: current + step, Home: minimum(items[index]), End: total - minimum(items[index + 1]) };
    if (!(event.key in targets)) return;
    event.preventDefault();
    resize(divider, targets[event.key]);
    save();
  });
  divider.addEventListener('dblclick', () => { sizes[mode()] = [...defaults[mode()]]; apply(); save(); });
}
new MutationObserver(apply).observe(debuggerPane, { attributes: true, attributeFilter: ['hidden'] });
const observer = new ResizeObserver(apply);
observer.observe(windows);
observer.observe(debuggerHeading);
observer.observe(evaluator);
apply();
