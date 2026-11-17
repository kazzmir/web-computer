const source = document.querySelector('#source');
const editor = document.querySelector('.source-editor');
const layer = document.querySelector('#syntax-layer');
const code = document.querySelector('#syntax-code');
const toggle = document.querySelector('#syntax-toggle');
const storageKey = 'web-computer.syntax-highlighting';
const debounceMs = 200;
const keywords = new Set('and as assert async await break class continue def del elif else except finally for from global if import in is lambda nonlocal not or pass raise return try while with yield'.split(' '));
const constants = new Set(['True', 'False', 'None', 'Ellipsis', 'NotImplemented']);
const builtins = new Set('abs all any bool bytes callable chr classmethod complex dict dir divmod enumerate filter float format frozenset getattr hasattr hash help hex id input int isinstance issubclass iter len list map max min next object oct open ord pow print property range repr reversed round set setattr slice sorted staticmethod str sum super tuple type vars zip __import__ Exception ValueError TypeError RuntimeError'.split(' '));

// A tolerant lexer, not an evaluator: incomplete strings and code can still
// be colored while typing. All source is inserted as text, never as HTML.
const tokens = /(?<comment>#[^\n]*)|(?<string>(?:[rRuUbBfF]{1,2})?(?:"""(?:\\[\s\S]|(?!""")[^\\])*?(?:"""|$)|'''(?:\\[\s\S]|(?!''')[^\\])*?(?:'''|$)|"(?:\\[\s\S]|[^"\\\r\n])*(?:"|(?=\r?\n|$))|'(?:\\[\s\S]|[^'\\\r\n])*(?:'|(?=\r?\n|$))))|(?<number>\b(?:0[xX][\da-fA-F_]+|0[bB][01_]+|0[oO][0-7_]+|\d[\d_]*(?:\.[\d_]*)?(?:[eE][+-]?[\d_]+)?)[jJ]?|\.[\d_]+(?:[eE][+-]?[\d_]+)?[jJ]?)|(?<identifier>[\p{L}_][\p{L}\p{N}_]*)|(?<operator>[-+*/%=<>!&|^~:@]+)/gu;
let enabled = false;
let timer = null;
let composing = false;
let displayedText = null;
let highlightedTokens = [];
let dirtyOffset = null;
try { enabled = localStorage.getItem(storageKey) === 'true'; } catch { /* Storage may be disabled. */ }

function syncScroll() {
  if (!enabled) return;
  layer.style.width = `${source.clientWidth}px`;
  layer.style.height = `${source.clientHeight}px`;
  layer.scrollTop = source.scrollTop;
  layer.scrollLeft = source.scrollLeft;
}
function textPosition(offset) {
  let position = 0;
  for (let i = 0; i < code.childNodes.length; i++) {
    const node = code.childNodes[i];
    if (offset === position) return [code, i];
    const end = position + node.textContent.length;
    if (offset < end) return [node.nodeType === Node.TEXT_NODE ? node : node.firstChild, offset - position];
    position = end;
  }
  return [code, code.childNodes.length];
}

// Patch only the changed text. Spans on either side stay in the DOM and keep
// their colors while the lexer is debounced, even across several pending edits.
function prepareChange() {
  const text = source.value;
  if (displayedText === null) {
    displayedText = '';
    code.textContent = '\n'; // Keep the final empty line scrollable.
    dirtyOffset = 0;
  }
  if (text === displayedText) return;
  let changed = 0;
  const length = Math.min(text.length, displayedText.length);
  while (changed < length && text[changed] === displayedText[changed]) changed++;
  let oldEnd = displayedText.length;
  let newEnd = text.length;
  while (oldEnd > changed && newEnd > changed && displayedText[oldEnd - 1] === text[newEnd - 1]) { oldEnd--; newEnd--; }
  const delta = newEnd - oldEnd;
  let restart = 0;
  let replaceStart = changed;
  let replaceEnd = oldEnd;
  for (const token of highlightedTokens) {
    // Include the preceding token at an edit boundary: quotes, prefixes, and
    // digits can combine with it when the lexer eventually runs.
    if (token.end < changed) restart = token.end;
    const overlaps = oldEnd === changed
      ? token.start < changed && token.end > changed
      : token.end > changed && token.start < oldEnd;
    if (overlaps) {
      replaceStart = Math.min(replaceStart, token.start);
      replaceEnd = Math.max(replaceEnd, token.end);
    }
  }
  dirtyOffset = dirtyOffset === null ? restart : Math.min(dirtyOffset, restart);
  const range = document.createRange();
  range.setStart(...textPosition(replaceStart));
  range.setEnd(...textPosition(replaceEnd));
  range.deleteContents();
  range.insertNode(document.createTextNode(text.slice(replaceStart, replaceEnd + delta)));
  highlightedTokens = highlightedTokens.filter((token) => {
    if (token.end <= replaceStart) return true;
    if (token.start >= replaceEnd) {
      token.start += delta;
      token.end += delta;
      return true;
    }
    return false;
  });
  displayedText = text;
  editor.classList.add('syntax-pending');
  syncScroll();
}
function render() {
  timer = null;
  if (!enabled || dirtyOffset === null) return;
  const text = displayedText;
  const fragment = document.createDocumentFragment();
  const retained = highlightedTokens.filter((token) => token.end <= dirtyOffset);
  const reusable = new Map(highlightedTokens.filter((token) => token.start >= dirtyOffset).map((token) => [token.start, token]));
  const previous = retained.at(-1);
  let end = previous?.end ?? 0;
  let definition = previous?.definition ?? false;
  // Detach the affected tail once, reusing unchanged spans as it is recolored.
  const oldTail = document.createDocumentFragment();
  while (previous ? previous.node.nextSibling : code.firstChild) oldTail.append(previous ? previous.node.nextSibling : code.firstChild);
  highlightedTokens = retained;
  tokens.lastIndex = end;
  for (const match of text.matchAll(tokens)) {
    fragment.append(document.createTextNode(text.slice(end, match.index)));
    let kind = Object.keys(match.groups).find((key) => match.groups[key] !== undefined);
    const value = match[0];
    if (kind === 'identifier') {
      if (keywords.has(value)) kind = 'keyword';
      else if (constants.has(value)) kind = 'constant';
      else if (definition) kind = 'function';
      else if (builtins.has(value)) kind = 'builtin';
    }
    const old = reusable.get(match.index);
    let span = old?.kind === kind && old.end === match.index + value.length && old.node.textContent === value ? old.node : null;
    if (!span) {
      span = document.createElement('span');
      span.className = `syntax-${kind}`;
      span.textContent = value;
    }
    fragment.append(span);
    definition = kind === 'keyword' && (value === 'def' || value === 'class');
    end = match.index + value.length;
    highlightedTokens.push({ start: match.index, end, kind, definition, node: span });
  }
  // An extra newline preserves the final empty line for scroll alignment.
  fragment.append(document.createTextNode(text.slice(end) + '\n'));
  code.append(fragment);
  dirtyOffset = null;
  editor.classList.remove('syntax-pending');
  syncScroll();
}
export function refreshHighlighting(immediate = false) {
  clearTimeout(timer);
  if (!enabled) return;
  prepareChange();
  if (immediate) render();
  else if (!composing) timer = setTimeout(render, debounceMs);
}
function applyEnabled() {
  clearTimeout(timer);
  editor.classList.toggle('syntax-enabled', enabled);
  editor.classList.remove('syntax-pending');
  toggle.setAttribute('aria-pressed', String(enabled));
  if (enabled) refreshHighlighting(true);
  else {
    code.replaceChildren();
    highlightedTokens = [];
    displayedText = null;
    dirtyOffset = null;
  }
}
toggle.addEventListener('click', () => {
  enabled = !enabled;
  applyEnabled();
  try { localStorage.setItem(storageKey, enabled); } catch { /* Storage may be disabled. */ }
});
source.addEventListener('input', () => refreshHighlighting());
source.addEventListener('scroll', syncScroll);
source.addEventListener('compositionstart', () => { composing = true; clearTimeout(timer); });
source.addEventListener('compositionend', () => { composing = false; refreshHighlighting(); });
new ResizeObserver(syncScroll).observe(source);
applyEnabled();
