const source = document.querySelector('#source');
const viewport = document.querySelector('#viewport');
const output = document.querySelector('#console');
const status = document.querySelector('#status');
const stop = document.querySelector('#stop');
const storageKey = 'web-computer.python';
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
let frame = null;
let programSource = '';
const placeholder = viewport.firstElementChild;
try { source.value = localStorage.getItem(storageKey) ?? example; }
catch { source.value = example; }
function save() { try { localStorage.setItem(storageKey, source.value); } catch { /* Private browsing / disabled storage. */ } }
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
  if (location.protocol === 'file:') {
    setStatus('HTTP server required', 'error');
    append('Serve web/ over HTTP: python3 -m http.server 8000 --directory web\n');
    return;
  }
  dispose();
  save();
  programSource = source.value;
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
      frame.contentWindow.postMessage({ channel: 'web-computer', type: 'run', source: programSource }, location.origin);
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
document.querySelector('#run').addEventListener('click', run);
stop.addEventListener('click', () => {
  dispose();
  viewport.replaceChildren(placeholder);
  setStatus('Stopped');
  append('\n[Program stopped]\n');
});
document.querySelector('#clear').addEventListener('click', () => { output.textContent = ''; });
document.querySelector('#reset').addEventListener('click', () => { source.value = example; save(); source.focus(); });
source.addEventListener('keydown', (event) => {
  if (event.key === 'Tab' && !event.ctrlKey && !event.metaKey) {
    event.preventDefault();
    source.setRangeText('    ', source.selectionStart, source.selectionEnd, 'end');
    save();
  }
});
document.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) { event.preventDefault(); run(); }
});
