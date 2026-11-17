import './splitter.js';
import './result-splitters.js';
import './brackets.js';
import { refreshHighlighting } from './highlight.js';
import { newlineText, dedentLines } from './editor-input.js';
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
const callStack = document.querySelector('#call-stack');
const localsTitle = document.querySelector('#locals-title');
const debugLocation = document.querySelector('#debug-location');
const debugControls = [...document.querySelectorAll('[data-debug-command]')];
const breakButton = document.querySelector('#debug-break');
const evaluator = document.querySelector('#debug-evaluator');
const expressionInput = document.querySelector('#debug-expression');
const evaluateButton = document.querySelector('#evaluate-expression');
const breakpoints = new Set();
let syntaxDiagnostics = [];
source.addEventListener('syntax-diagnostics', ({ detail }) => {
  if (detail.source !== source.value) return;
  syntaxDiagnostics = detail.diagnostics;
  renderGutter();
});
let pausedLine = null;
let pausedFilename = null;
let debugReady = false;
let debugRunning = false;
let breakPending = false;
let canStepOut = false;
let stackFrames = [];
let selectedFrameId = 0;
let requestedFrameId = null;
let evaluationPending = null;
let evaluationSequence = 0;
let evaluationSupported = false;
function updateDebugControls() {
  const ready = debugging && pausedLine !== null && !variableEditor && requestedFrameId === null && !evaluationPending;
  for (const button of debugControls) button.disabled = !ready || (button.dataset.debugCommand === 'out' && !canStepOut);
  for (const button of callStack.querySelectorAll('button')) button.disabled = !ready;
  breakButton.disabled = !debugging || !debugReady || !debugRunning || breakPending;
  for (const button of locals.querySelectorAll('.local-variable')) button.disabled = !ready;
  expressionInput.disabled = evaluateButton.disabled = !ready || !evaluationSupported;
  const selected = stackFrames.find((frame) => frame.id === selectedFrameId);
  expressionInput.placeholder = debugReady && !evaluationSupported ? 'Rebuild runtime to enable evaluation' : ready && selected ? `Expression in ${selected.function}` : 'Pause debugging to evaluate';
}
function showLocals(items) {
  locals.replaceChildren(...items.map((local) => variableRow(local.name, local.value)));
  if (!items.length) locals.textContent = '(No local variables yet)';
  const frame = stackFrames.find((item) => item.id === selectedFrameId);
  localsTitle.textContent = frame ? `Locals — ${frame.function}` : 'Local variables';
}
function jumpToFrame(frame) {
  if (!frame || frame.filename !== program?.filename) return;
  const lines = source.value.split('\n');
  if (!Number.isInteger(frame.line) || frame.line < 1 || frame.line > lines.length) return;
  let offset = 0;
  for (let i = 0; i < frame.line - 1; i++) offset += lines[i].length + 1;
  offset += lines[frame.line - 1].match(/^[\t ]*/)[0].length;
  source.focus({ preventScroll: true });
  source.setSelectionRange(offset, offset);
  const style = getComputedStyle(source);
  source.scrollTop = Math.max(0, parseFloat(style.paddingTop) + (frame.line - 1) * parseFloat(style.lineHeight) - source.clientHeight / 3);
  source.scrollLeft = 0;
  source.dispatchEvent(new Event('scroll'));
}
function renderCallStack() {
  callStack.replaceChildren(...stackFrames.map((frame) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'stack-frame';
    button.dataset.frameId = frame.id;
    button.setAttribute('aria-pressed', String(frame.id === selectedFrameId));
    button.setAttribute('aria-label', `${frame.function}, ${frame.filename}, line ${frame.line}${frame.id === 0 ? ', executing frame' : ''}`);
    button.title = `${frame.function} — ${frame.filename}:${frame.line}`;
    const name = document.createElement('span');
    name.textContent = `${frame.id === 0 ? '▶ ' : ''}${frame.function}`;
    const location = document.createElement('span');
    location.className = 'frame-location';
    location.textContent = `${frame.filename}:${frame.line}`;
    button.append(name, location);
    button.addEventListener('click', () => {
      if (button.disabled) return;
      if (frame.id === selectedFrameId) { jumpToFrame(frame); return; }
      requestedFrameId = frame.id;
      updateDebugControls();
      debugCommand('select-frame', { frameId: frame.id });
    });
    return button;
  }));
}
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
  if (!debugging || pausedLine === null || variableEditor || requestedFrameId !== null || evaluationPending) return;
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
  const editor = { row, name, input, save, cancel, error, saving: false, frameId: selectedFrameId };
  variableEditor = editor;
  updateDebugControls();
  const close = () => {
    row.replaceWith(variableRow(name, value));
    variableEditor = null;
    updateDebugControls();
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
    debugCommand('set-variable', { name, value: input.value, frameId: editor.frameId });
  });
  input.focus();
  input.select();
}
function renderGutter() {
  const count = source.value.split('\n').length;
  gutter.style.setProperty('--gutter-digits', Math.max(3, String(count).length));
  const inspected = stackFrames.find((frame) => frame.id === selectedFrameId);
  const inspectedLine = debugging && pausedLine !== null && selectedFrameId !== 0 && inspected?.filename === program?.filename ? inspected.line : null;
  for (const line of breakpoints) if (line > count) breakpoints.delete(line);
  gutterLines.replaceChildren(...Array.from({ length: count }, (_, i) => {
    const line = i + 1;
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = line;
    button.dataset.line = line;
    button.classList.toggle('breakpoint', breakpoints.has(line));
    button.classList.toggle('current', pausedLine === line && pausedFilename === program?.filename);
    button.classList.toggle('inspected', inspectedLine === line);
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
function debugCommand(command, payload) {
  // Cached runners may still forward the older `variable` field to Go.
  frame?.contentWindow.postMessage({ channel: 'web-computer', type: 'debug-command', command, breakpoints: [...breakpoints], payload, variable: payload }, location.origin);
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
  evaluationPending = null;
  evaluationSupported = false;
  variableEditor = null;
  stackFrames = [];
  selectedFrameId = 0;
  requestedFrameId = null;
  callStack.textContent = 'Not paused';
  localsTitle.textContent = 'Local variables';
  locals.querySelectorAll('button, input').forEach((control) => { control.disabled = true; });
  debugging = false;
  pausedLine = null;
  pausedFilename = null;
  debugReady = debugRunning = breakPending = false;
  breakButton.disabled = true;
  canStepOut = false;
  debugControls.forEach((button) => { button.disabled = true; });
  document.querySelector('#debug-stop').disabled = true;
  source.readOnly = false;
  selector.disabled = resetButton.disabled = false;
  updateDebugControls();
  renderGutter();
}
evaluator.addEventListener('submit', (event) => {
  event.preventDefault();
  if (!debugging || pausedLine === null || variableEditor || requestedFrameId !== null || evaluationPending || !evaluationSupported) return;
  const expression = expressionInput.value.trim();
  if (!expression) { expressionInput.focus(); return; }
  evaluationPending = { requestId: ++evaluationSequence, frameId: selectedFrameId, expression };
  if (output.textContent && !output.textContent.endsWith('\n')) append('\n');
  append(`>>> ${expression}\n`);
  updateDebugControls();
  debugCommand('evaluate', evaluationPending);
});
evaluator.addEventListener('keydown', (event) => {
  if (event.key !== 'Enter') return;
  event.stopPropagation();
  if (event.ctrlKey || event.metaKey) { event.preventDefault(); evaluator.requestSubmit(); }
});
breakButton.addEventListener('click', () => {
  if (breakButton.disabled) return;
  breakPending = true;
  updateDebugControls();
  debugCommand('break');
  debugLocation.textContent = 'Debugger — pausing…';
  setStatus('Pausing…', 'running');
});
for (const button of debugControls) button.addEventListener('click', () => {
  if (button.disabled) return;
  debugCommand(button.dataset.debugCommand);
  pausedLine = null;
  pausedFilename = null;
  debugRunning = false; // Wait for Go to acknowledge the resume before enabling Break.
  breakPending = false;
  canStepOut = false;
  renderGutter();
  debugControls.forEach((control) => { control.disabled = true; });
  debugLocation.textContent = 'Debugger — running';
  locals.textContent = 'Running…';
  callStack.textContent = 'Running…';
  stackFrames = [];
  selectedFrameId = 0;
  requestedFrameId = null;
  localsTitle.textContent = 'Local variables';
  updateDebugControls();
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
  debugPanel.hidden = evaluator.hidden = !debug;
  expressionInput.value = '';
  locals.textContent = 'Waiting for a breakpoint…';
  callStack.textContent = 'Waiting for a breakpoint…';
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
    case 'running':
      debugRunning = debugging;
      updateDebugControls();
      setStatus('Running', 'running');
      break;
    case 'debug-ready':
      debugReady = debugging;
      evaluationSupported = message.canEvaluate === true;
      updateDebugControls();
      break;
    case 'debug-resumed':
      debugRunning = debugging && pausedLine === null;
      updateDebugControls();
      break;
    case 'paused': {
      if (!debugging) break;
      pausedLine = message.line;
      pausedFilename = message.filename ?? selected.filename;
      debugRunning = breakPending = false;
      canStepOut = message.canStepOut === true;
      setStatus(`Paused at line ${pausedLine}`, 'paused');
      debugLocation.textContent = `${pausedFilename}:${pausedLine} — ${message.function}`;
      stackFrames = message.frames ?? [{ id: 0, function: message.function, filename: selected.filename, line: message.line }];
      selectedFrameId = 0;
      requestedFrameId = null;
      renderCallStack();
      showLocals(message.locals);
      updateDebugControls();
      const lineHeight = parseFloat(getComputedStyle(source).lineHeight);
      const top = (pausedLine - 1) * lineHeight;
      if (pausedFilename === program?.filename && (top < source.scrollTop || top > source.scrollTop + source.clientHeight - 50)) source.scrollTop = Math.max(0, top - 50);
      renderGutter();
      break;
    }
    case 'frame-selected': {
      if (!debugging || pausedLine === null || requestedFrameId !== message.frameId) break;
      requestedFrameId = null;
      if (!message.error) {
        selectedFrameId = message.frameId;
        renderCallStack();
        showLocals(message.locals);
        jumpToFrame(stackFrames.find((frame) => frame.id === selectedFrameId));
        renderGutter();
      }
      updateDebugControls();
      break;
    }
    case 'evaluation-result': {
      const pending = evaluationPending;
      if (!pending || pending.requestId !== message.requestId || pending.frameId !== message.frameId || !debugging || pausedLine === null) break;
      evaluationPending = null;
      append(`${message.error ?? message.value}\n`);
      showLocals(message.locals);
      updateDebugControls();
      expressionInput.focus();
      expressionInput.select();
      break;
    }
    case 'variable-result': {
      const editor = variableEditor;
      if (!editor || editor.name !== message.name || editor.frameId !== (message.frameId ?? 0) || !debugging || pausedLine === null) break;
      if (message.error) {
        editor.saving = false;
        editor.input.disabled = editor.save.disabled = editor.cancel.disabled = false;
        editor.error.textContent = message.error;
        editor.input.focus();
      } else {
        editor.row.replaceWith(variableRow(message.name, message.value));
        variableEditor = null;
        updateDebugControls();
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
  if (event.key === 'Tab' && event.shiftKey) {
    event.preventDefault();
    const edit = dedentLines(source.value, source.selectionStart, source.selectionEnd);
    if (edit) {
      const direction = source.selectionDirection;
      source.setRangeText(edit.text, edit.start, edit.end, 'preserve');
      source.setSelectionRange(edit.selectionStart, edit.selectionEnd, direction);
      source.dispatchEvent(new Event('input', { bubbles: true }));
    }
    return;
  }
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
