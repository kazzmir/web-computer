import { indexFunctions } from './function-index.js';

const source = document.querySelector('#source');
const tooltip = document.querySelector('#function-tooltip');
const signature = tooltip.querySelector('.function-signature');
const documentation = tooltip.querySelector('.function-documentation');
const origin = tooltip.querySelector('.function-origin');
const measure = document.createElement('canvas').getContext('2d');
let index = null;
let analysisTimer = null;
let hoverTimer = null;
let target = null;

function hide() {
  clearTimeout(hoverTimer);
  target = null;
  tooltip.hidden = true;
  const describedBy = source.getAttribute('aria-describedby')?.split(' ').filter((id) => id !== tooltip.id);
  source.setAttribute('aria-describedby', describedBy?.join(' ') ?? '');
}
export function refreshFunctionIndex() {
  hide();
  clearTimeout(analysisTimer);
  index = null;
  analysisTimer = setTimeout(() => { index = indexFunctions(source.value); }, 200);
}
function offsetAtPoint(event) {
  const bounds = source.getBoundingClientRect();
  const style = getComputedStyle(source);
  const left = parseFloat(style.paddingLeft);
  const top = parseFloat(style.paddingTop);
  const localX = event.clientX - bounds.left;
  const localY = event.clientY - bounds.top;
  if (localX < left || localY < top || localX >= source.clientWidth || localY >= source.clientHeight) return null;
  const row = Math.floor((localY - top + source.scrollTop) / parseFloat(style.lineHeight));
  const { lines, lineStarts } = index;
  if (row < 0 || row >= lines.length) return null;
  let offset = lineStarts[row];
  measure.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
  const tabWidth = measure.measureText(' ').width * (parseFloat(style.tabSize) || 4);
  const x = localX - left + source.scrollLeft;
  let position = 0;
  for (const character of lines[row]) {
    const width = character === '\t' ? tabWidth - position % tabWidth : measure.measureText(character).width;
    if (x >= position && x < position + width) return offset;
    position += width;
    offset += character.length;
  }
  return null;
}
source.addEventListener('pointermove', (event) => {
  if (event.pointerType === 'touch' || event.buttons) { hide(); return; }
  // Programmatic changes also invalidate the cache, even without an input event.
  if (index?.text !== source.value) {
    if (index) refreshFunctionIndex();
    return;
  }
  const offset = offsetAtPoint(event);
  const info = offset === null ? null : index.known.find((token) => token.start <= offset && offset < token.end);
  if (!info) { hide(); return; }
  if (target === info) return;
  hide();
  target = info;
  const x = event.clientX;
  const y = event.clientY;
  hoverTimer = setTimeout(() => {
    if (target !== info || index?.text !== source.value) return;
    signature.textContent = info.signature;
    documentation.textContent = info.documentation;
    origin.textContent = info.origin;
    tooltip.hidden = false;
    const box = tooltip.getBoundingClientRect();
    tooltip.style.left = `${Math.max(8, Math.min(x + 12, innerWidth - box.width - 8))}px`;
    tooltip.style.top = `${Math.max(8, y + 20 + box.height <= innerHeight - 8 ? y + 20 : y - box.height - 12)}px`;
    const ids = new Set(source.getAttribute('aria-describedby')?.split(' ').filter(Boolean));
    ids.add(tooltip.id);
    source.setAttribute('aria-describedby', [...ids].join(' '));
  }, 350);
});
source.addEventListener('input', refreshFunctionIndex);
for (const type of ['pointerleave', 'pointerdown', 'scroll', 'blur']) source.addEventListener(type, hide);
document.addEventListener('keydown', hide);
window.addEventListener('resize', hide);
refreshFunctionIndex();
