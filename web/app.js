import './splitter.js';
import { refreshHighlighting } from './highlight.js';
import { newlineText } from './editor-input.js';
import { refreshFunctionIndex } from './hover.js';
import { refreshSyntaxErrors } from './syntax-check.js';

const source = document.querySelector('#source');
const viewport = document.querySelector('#viewport');
const output = document.querySelector('#console');
const status = document.querySelector('#status');
const stop = document.querySelector('#stop');
const storageKey = 'web-computer.python';
const selector = document.querySelector('#examples');
const runButton = document.querySelector('#run');
const resetButton = document.querySelector('#reset');
const debugButton = document.querySelector('#debug');
const gutter = document.querySelector('#gutter');
const gutterLines = document.querySelector('#gutter-lines');
const debugPanel = document.querySelector('#debugger');
const locals = document.querySelector('#locals');
const debugLocation = document.querySelector('#debug-location');
const debugControls = [...document.querySelectorAll('[data-debug-command]')];
const breakpoints = new Set();
let syntaxDiagnostics = [];
source.addEventListener('syntax-diagnostics', ({ detail }) => {
  if (detail.source !== source.value) return;
  syntaxDiagnostics = detail.diagnostics;
  renderGutter();
});
let pausedLine = null;
let debugging = false;
let variableEditor = null;
function variableRow(name, value) {
  const row = document.createElement('div');
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'local-variable';
  button.setAttribute('aria-label', `Edit variable ${name}`);
  const label = document.createElement('span');
  label.className = 'local-name';
  label.textContent = name;
  button.append(label, document.createTextNode(` = ${value}`));
  button.addEventListener('click', () => editVariable(row, name, value));
  row.append(button);
  return row;
}
function editVariable(row, name, value) {
  if (!debugging || pausedLine === null || variableEditor) return;
  debugControls.forEach((button) => { button.disabled = true; });
  const form = document.createElement('form');
  form.className = 'variable-editor';
  const label = document.createElement('label');
  label.textContent = `${name} = `;
  const input = document.createElement('input');
  input.setAttribute('aria-label', `New value for ${name}`);
  input.autocomplete = 'off';
  input.spellcheck = false;
  input.value = value.includes('…') || value.startsWith('<') ? '' : value;
  input.placeholder = '"text", 3, 8.2, [1, 2], {1: "a"}';
  label.append(input);
  const save = document.createElement('button');
  save.type = 'submit';
  save.textContent = 'Save';
  const cancel = document.createElement('button');
  cancel.type = 'button';
  cancel.className = 'secondary';
  cancel.textContent = 'Cancel';
  const error = document.createElement('div');
  error.className = 'variable-error';
  error.setAttribute('role', 'alert');
  form.append(label, save, cancel, error);
  row.replaceChildren(form);
  const editor = { row, name, input, save, cancel, error, saving: false };
  variableEditor = editor;
  const close = () => {
    row.replaceWith(variableRow(name, value));
    variableEditor = null;
    debugControls.forEach((button) => { button.disabled = false; });
  };
  cancel.addEventListener('click', close);
  input.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !editor.saving) { event.preventDefault(); close(); }
  });
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    if (editor.saving) return;
    editor.saving = true;
    input.disabled = save.disabled = cancel.disabled = true;
    error.textContent = '';
    debugCommand('set-variable', { name, value: input.value });
  });
  input.focus();
  input.select();
}
function renderGutter() {
  const count = source.value.split('\n').length;
  for (const line of breakpoints) if (line > count) breakpoints.delete(line);
  gutterLines.replaceChildren(...Array.from({ length: count }, (_, i) => {
    const line = i + 1;
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = line;
    button.dataset.line = line;
    button.classList.toggle('breakpoint', breakpoints.has(line));
    button.classList.toggle('current', pausedLine === line);
    const error = syntaxDiagnostics.find((item) => item.line === line);
    button.classList.toggle('syntax-error', !!error);
    const action = `Line ${line}: ${breakpoints.has(line) ? 'remove' : 'add'} breakpoint`;
    button.setAttribute('aria-label', error ? `${action}. ${error.message}, column ${error.column}` : action);
    if (error) {
      button.title = `Line ${line}, column ${error.column}: ${error.message}`;
      button.setAttribute('aria-describedby', 'syntax-status');
    }
    button.setAttribute('aria-pressed', String(breakpoints.has(line)));
    return button;
  }));
  gutter.scrollTop = source.scrollTop;
}
function debugCommand(command, variable) {
  frame?.contentWindow.postMessage({ channel: 'web-computer', type: 'debug-command', command, breakpoints: [...breakpoints], variable }, location.origin);
}
gutterLines.addEventListener('click', (event) => {
  const line = Number(event.target.closest('button')?.dataset.line);
  if (!line) return;
  if (breakpoints.has(line)) breakpoints.delete(line); else breakpoints.add(line);
  renderGutter();
  if (debugging) debugCommand('breakpoints');
});
source.addEventListener('scroll', () => { gutter.scrollTop = source.scrollTop; });
function finishDebug() {
  variableEditor = null;
  locals.querySelectorAll('button, input').forEach((control) => { control.disabled = true; });
  debugging = false;
  pausedLine = null;
  debugControls.forEach((button) => { button.disabled = true; });
  document.querySelector('#debug-stop').disabled = true;
  source.readOnly = false;
  selector.disabled = resetButton.disabled = false;
  renderGutter();
}
for (const button of debugControls) button.addEventListener('click', () => {
  debugCommand(button.dataset.debugCommand);
  pausedLine = null;
  renderGutter();
  debugControls.forEach((control) => { control.disabled = true; });
  debugLocation.textContent = 'Debugger — running';
  locals.textContent = 'Running…';
  setStatus('Running', 'running');
});
const example = `import pygame

pygame.init()
screen = pygame.display.set_mode((640, 480))
pygame.display.set_caption("Hello from Python")
font = pygame.font.Font(None, 30)
label = font.render("Python is running in your browser!", True, "white")
clock = pygame.time.Clock()
x = 0
running = True

print("Hello! Click the window; Escape exits.")
while running:
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            running = False
        if event.type == pygame.KEYDOWN and event.key == pygame.K_ESCAPE:
            running = False

    screen.fill((18, 27, 43))
    screen.blit(label, (32, 40))
    screen.fill((110, 194, 255), (x, 180, 80, 80))
    x = (x + 3) % 640
    pygame.display.flip()
    clock.tick(60)

pygame.quit()
`;
const examples = [
  { id: 'starter', label: 'Welcome animation', filename: 'editor.py', help: 'Moving square · Escape exits' },
  { id: 'chimp', label: 'Chimp — sprites & sound', filename: 'chimp.py', help: 'Move the mouse and click to punch · Escape exits' },
  { id: 'aliens', label: 'Aliens — arcade game', filename: 'aliens.py', help: 'Arrows move · Space shoots · F toggles fullscreen · Escape exits' },
  { id: 'liquid', label: 'Liquid — image animation', filename: 'liquid.py', help: 'Any key or mouse click exits' },
  { id: 'scroll', label: 'Scroll — image viewer', filename: 'scroll.py', help: 'Arrows or on-screen triangles scroll · Escape exits' },
  { id: 'scaletest', label: 'Scaletest — image scaling', filename: 'scaletest.py', args: ['data/chimp.png'], help: 'Starts fullscreen · Escape leaves fullscreen/exits · Arrows resize · Uses data/chimp.png' },
  { id: 'glcube', label: 'GL cube — 3D rendering', filename: 'glcube.py', help: 'F toggles fullscreen · Escape exits' },
  { id: 'video', label: 'Video — SDL2 renderer panels', filename: 'video.py', help: 'Enter confirms the dialog · Arrows move · Space changes color · Escape exits' },
];
for (const info of examples) selector.add(new Option(info.label, info.id));
let selected = examples[0];
let loading = 0;
let frame = null;
let program = null;
const placeholder = viewport.firstElementChild;
let savedSource = null;
try {
  selected = examples.find((info) => info.id === localStorage.getItem(`${storageKey}.example`)) ?? selected;
  savedSource = localStorage.getItem(storageKey);
} catch { /* Private browsing / disabled storage. */ }
source.value = savedSource ?? example;
renderGutter();
refreshHighlighting(true);
refreshFunctionIndex();
refreshSyntaxErrors();
function showSelection() {
  selector.value = selected.id;
  document.querySelector('#editor-title').textContent = selected.filename;
  document.querySelector('#example-help').textContent = selected.help;
}
showSelection();
function save() {
  try {
    localStorage.setItem(storageKey, source.value);
    localStorage.setItem(`${storageKey}.example`, selected.id);
  } catch { /* Private browsing / disabled storage. */ }
}
async function loadExample(id) {
  const info = examples.find((item) => item.id === id);
  if (!info) return;
  const request = ++loading;
  runButton.disabled = debugButton.disabled = resetButton.disabled = true;
  source.readOnly = true;
  try {
    let text = example;
    if (info.id !== 'starter') {
      const response = await fetch(`examples/${info.filename}`);
      if (!response.ok) throw new Error(`Cannot load ${info.filename} (${response.status}). Run ./web/build.sh to copy the examples.`);
      text = await response.text();
    }
    if (request !== loading) return;
    selected = info;
    source.value = text;
    refreshHighlighting(true);
    refreshFunctionIndex();
    refreshSyntaxErrors();
    breakpoints.clear();
    renderGutter();
    showSelection();
    save();
  } catch (error) {
    if (request !== loading) return;
    showSelection();
    append(`${error.message}\n`);
    setStatus('Example unavailable', 'error');
  } finally {
    if (request === loading) {
      runButton.disabled = debugButton.disabled = resetButton.disabled = false;
      source.readOnly = false;
    }
  }
}
selector.addEventListener('change', () => loadExample(selector.value));
if (savedSource === null && selected.id !== 'starter') loadExample(selected.id);
source.addEventListener('input', () => { save(); renderGutter(); });
function setStatus(text, state = '') { status.textContent = text; status.dataset.state = state; }
function append(text) {
  output.textContent = (output.textContent + text).slice(-100_000);
  output.scrollTop = output.scrollHeight;
}
function dispose() {
  if (frame) frame.remove();
  frame = null;
  stop.disabled = true;
  finishDebug();
}
function run(debug = false) {
  if (runButton.disabled) return;
  if (location.protocol === 'file:') {
    setStatus('HTTP server required', 'error');
    append('Serve web/ over HTTP: python3 -m http.server 8000 --directory web\n');
    return;
  }
  dispose();
  save();
  debugging = debug;
  debugPanel.hidden = !debug;
  locals.textContent = 'Waiting for a breakpoint…';
  debugLocation.textContent = 'Debugger — running';
  document.querySelector('#debug-stop').disabled = !debug;
  source.readOnly = debug;
  selector.disabled = resetButton.disabled = debug;
  program = { source: source.value, filename: selected.filename, args: selected.args ?? [], debug, breakpoints: [...breakpoints] };
  output.textContent = '';
  setStatus('Loading runtime…');
  stop.disabled = false;
  frame = document.createElement('iframe');
  frame.title = 'Python program window';
  frame.setAttribute('sandbox', 'allow-scripts allow-same-origin allow-downloads allow-pointer-lock');
  frame.setAttribute('allow', 'autoplay; fullscreen');
  frame.src = 'runner.html';
  viewport.replaceChildren(frame);
}
window.addEventListener('message', (event) => {
  if (!frame || event.source !== frame.contentWindow || event.origin !== location.origin) return;
  const message = event.data;
  if (!message || message.channel !== 'web-computer') return;
  switch (message.type) {
    case 'ready':
      program.breakpoints = [...breakpoints];
      frame.contentWindow.postMessage({ channel: 'web-computer', type: 'run', ...program }, location.origin);
      break;
    case 'running': setStatus('Running', 'running'); break;
    case 'paused': {
      if (!debugging) break;
      pausedLine = message.line;
      setStatus(`Paused at line ${pausedLine}`, 'paused');
      debugLocation.textContent = `${selected.filename}:${pausedLine} — ${message.function}`;
      debugControls.forEach((button) => { button.disabled = false; });
      locals.replaceChildren();
      for (const local of message.locals) locals.append(variableRow(local.name, local.value));
      if (!message.locals.length) locals.textContent = '(No local variables yet)';
      const lineHeight = parseFloat(getComputedStyle(source).lineHeight);
      const top = (pausedLine - 1) * lineHeight;
      if (top < source.scrollTop || top > source.scrollTop + source.clientHeight - 50) source.scrollTop = Math.max(0, top - 50);
      renderGutter();
      break;
    }
    case 'variable-result': {
      const editor = variableEditor;
      if (!editor || editor.name !== message.name || !debugging || pausedLine === null) break;
      if (message.error) {
        editor.saving = false;
        editor.input.disabled = editor.save.disabled = editor.cancel.disabled = false;
        editor.error.textContent = message.error;
        editor.input.focus();
      } else {
        editor.row.replaceWith(variableRow(message.name, message.value));
        variableEditor = null;
        debugControls.forEach((button) => { button.disabled = false; });
      }
      break;
    }
    case 'output': if (typeof message.text === 'string') append(message.text); break;
    case 'exit':
      setStatus(message.code === 0 ? 'Finished' : `Exited (${message.code})`, message.code === 0 ? '' : 'error');
      stop.disabled = true;
      finishDebug();
      break;
    case 'error':
      append(`${message.text}\n`);
      setStatus('Error', 'error');
      stop.disabled = true;
      finishDebug();
      break;
  }
});
runButton.addEventListener('click', () => run());
debugButton.addEventListener('click', () => run(true));
document.querySelector('#debug-stop').addEventListener('click', () => stop.click());
stop.addEventListener('click', () => {
  dispose();
  viewport.replaceChildren(placeholder);
  setStatus('Stopped');
  append('\n[Program stopped]\n');
});
document.querySelector('#clear').addEventListener('click', () => { output.textContent = ''; });
resetButton.addEventListener('click', () => loadExample(selected.id));
source.addEventListener('keydown', (event) => {
  if (source.readOnly || event.isComposing || event.ctrlKey || event.metaKey || event.altKey) return;
  let insertion;
  if (event.key === 'Tab') insertion = '    ';
  else if (event.key === 'Enter') insertion = newlineText(source.value, source.selectionStart);
  else return;
  event.preventDefault();
  source.setRangeText(insertion, source.selectionStart, source.selectionEnd, 'end');
  source.dispatchEvent(new Event('input', { bubbles: true }));
});
document.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) { event.preventDefault(); run(); }
});
