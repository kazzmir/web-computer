import { indexBrackets, matchingBrackets } from './bracket-pairs.js';

const source = document.querySelector('#source');
const layer = document.querySelector('#bracket-layer');
const measure = document.createElement('canvas').getContext('2d');
let index = null;
let frame = null;

function position(offset, style) {
  let low = 0;
  let high = index.lineStarts.length - 1;
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (index.lineStarts[middle] <= offset) low = middle;
    else high = middle - 1;
  }
  const prefix = index.lines[low].slice(0, offset - index.lineStarts[low]);
  const tabWidth = measure.measureText(' ').width * (parseFloat(style.tabSize) || 4);
  let width = 0;
  const pieces = prefix.split('\t');
  for (let i = 0; i < pieces.length; i++) {
    if (i) width += tabWidth - width % tabWidth;
    width += measure.measureText(pieces[i]).width;
  }
  return {
    left: parseFloat(style.paddingLeft) + width - source.scrollLeft,
    top: parseFloat(style.paddingTop) + low * parseFloat(style.lineHeight) - source.scrollTop,
  };
}
function render() {
  frame = null;
  layer.replaceChildren();
  if (document.activeElement !== source) return;
  if (index?.text !== source.value) index = indexBrackets(source.value);
  const pair = matchingBrackets(index, source.selectionStart, source.selectionEnd);
  if (!pair) return;
  const style = getComputedStyle(source);
  measure.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
  layer.style.width = `${source.clientWidth}px`;
  layer.style.height = `${source.clientHeight}px`;
  for (const offset of pair) {
    const point = position(offset, style);
    const mark = document.createElement('span');
    mark.className = 'bracket-match';
    mark.dataset.offset = offset;
    mark.style.left = `${point.left}px`;
    mark.style.top = `${point.top}px`;
    mark.style.width = `${measure.measureText(index.text[offset]).width}px`;
    mark.style.height = style.lineHeight;
    layer.append(mark);
  }
}
function schedule() {
  if (frame === null) frame = requestAnimationFrame(render);
}
for (const type of ['input', 'select', 'selectionchange', 'keyup', 'pointerup', 'scroll', 'focus', 'blur']) source.addEventListener(type, schedule);
document.addEventListener('selectionchange', schedule);
new ResizeObserver(schedule).observe(source);
schedule();
