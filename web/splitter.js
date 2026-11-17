const main = document.querySelector('main');
const editor = document.querySelector('.editor-pane');
const divider = document.querySelector('#pane-divider');
const storageKey = 'web-computer.pane-split';
const defaultRatio = 1 / 2.15;
const minEditor = 280;
const minResult = 340;
let ratio = defaultRatio;
let drag = null;
try {
  const saved = Number(localStorage.getItem(storageKey));
  if (saved > 0 && saved < 1) ratio = saved;
} catch { /* Storage may be disabled. */ }

function availableWidth() { return main.clientWidth - divider.offsetWidth; }
function updateAccessibility() {
  if (!divider.offsetWidth) return; // Stacked mobile layout has no divider.
  const width = availableWidth();
  divider.setAttribute('aria-valuemin', Math.ceil(minEditor / width * 100));
  divider.setAttribute('aria-valuemax', Math.floor((width - minResult) / width * 100));
  const percent = Math.round(editor.getBoundingClientRect().width / width * 100);
  divider.setAttribute('aria-valuenow', percent);
  divider.setAttribute('aria-valuetext', `Editor ${percent}%, results ${100 - percent}%`);
}
function apply() {
  main.style.setProperty('--editor-width', `${ratio}fr`);
  main.style.setProperty('--result-width', `${1 - ratio}fr`);
  updateAccessibility();
}
function setWidth(width) {
  const available = availableWidth();
  if (available < minEditor + minResult) return;
  ratio = Math.max(minEditor, Math.min(available - minResult, width)) / available;
  apply();
}
function save() {
  try { localStorage.setItem(storageKey, ratio); } catch { /* Storage may be disabled. */ }
}
function finishDrag(event) {
  if (!drag || event.pointerId !== drag.pointerId) return;
  drag = null;
  document.body.classList.remove('resizing-panes');
  if (divider.hasPointerCapture(event.pointerId)) divider.releasePointerCapture(event.pointerId);
  save();
}
divider.addEventListener('pointerdown', (event) => {
  if (!event.isPrimary || event.button !== 0 || !divider.offsetWidth || drag) return;
  event.preventDefault();
  divider.focus();
  drag = { pointerId: event.pointerId, x: event.clientX, width: editor.getBoundingClientRect().width };
  divider.setPointerCapture(event.pointerId);
  document.body.classList.add('resizing-panes');
});
divider.addEventListener('pointermove', (event) => {
  if (drag && event.pointerId === drag.pointerId) setWidth(drag.width + event.clientX - drag.x);
});
for (const type of ['pointerup', 'pointercancel', 'lostpointercapture']) divider.addEventListener(type, finishDrag);
divider.addEventListener('keydown', (event) => {
  const width = availableWidth();
  const current = editor.getBoundingClientRect().width;
  const step = width * (event.shiftKey ? 0.1 : 0.02);
  const targets = { ArrowLeft: current - step, ArrowRight: current + step, Home: minEditor, End: width - minResult };
  if (!(event.key in targets)) return;
  event.preventDefault();
  setWidth(targets[event.key]);
  save();
});
divider.addEventListener('dblclick', () => {
  ratio = defaultRatio;
  apply();
  save();
});
new ResizeObserver(updateAccessibility).observe(main);
apply();
