const source = document.querySelector('#source');
const viewport = document.querySelector('#viewport');
const output = document.querySelector('#console');
const status = document.querySelector('#status');
const stop = document.querySelector('#stop');
const storageKey = 'web-computer.python';
const selector = document.querySelector('#examples');
const runButton = document.querySelector('#run');
const resetButton = document.querySelector('#reset');
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
  runButton.disabled = resetButton.disabled = true;
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
    showSelection();
    save();
  } catch (error) {
    if (request !== loading) return;
    showSelection();
    append(`${error.message}\n`);
    setStatus('Example unavailable', 'error');
  } finally {
    if (request === loading) {
      runButton.disabled = resetButton.disabled = false;
      source.readOnly = false;
    }
  }
}
selector.addEventListener('change', () => loadExample(selector.value));
if (savedSource === null && selected.id !== 'starter') loadExample(selected.id);
source.addEventListener('input', save);
function setStatus(text, state = '') { status.textContent = text; status.dataset.state = state; }
function append(text) {
  output.textContent = (output.textContent + text).slice(-100_000);
  output.scrollTop = output.scrollHeight;
}
function dispose() {
  if (frame) frame.remove();
  frame = null;
  stop.disabled = true;
}
function run() {
  if (runButton.disabled) return;
  if (location.protocol === 'file:') {
    setStatus('HTTP server required', 'error');
    append('Serve web/ over HTTP: python3 -m http.server 8000 --directory web\n');
    return;
  }
  dispose();
  save();
  program = { source: source.value, filename: selected.filename, args: selected.args ?? [] };
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
      frame.contentWindow.postMessage({ channel: 'web-computer', type: 'run', ...program }, location.origin);
      break;
    case 'running': setStatus('Running', 'running'); break;
    case 'output': if (typeof message.text === 'string') append(message.text); break;
    case 'exit':
      setStatus(message.code === 0 ? 'Finished' : `Exited (${message.code})`, message.code === 0 ? '' : 'error');
      stop.disabled = true;
      break;
    case 'error':
      append(`${message.text}\n`);
      setStatus('Error', 'error');
      stop.disabled = true;
      break;
  }
});
runButton.addEventListener('click', run);
stop.addEventListener('click', () => {
  dispose();
  viewport.replaceChildren(placeholder);
  setStatus('Stopped');
  append('\n[Program stopped]\n');
});
document.querySelector('#clear').addEventListener('click', () => { output.textContent = ''; });
resetButton.addEventListener('click', () => loadExample(selected.id));
source.addEventListener('keydown', (event) => {
  if (source.readOnly) return;
  if (event.key === 'Tab' && !event.ctrlKey && !event.metaKey) {
    event.preventDefault();
    source.setRangeText('    ', source.selectionStart, source.selectionEnd, 'end');
    save();
  }
});
document.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) { event.preventDefault(); run(); }
});
