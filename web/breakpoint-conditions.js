export function setupBreakpointConditions({ source, gutter, breakpoints, conditions, onChange }) {
  const form = document.querySelector('#breakpoint-condition-editor');
  const title = document.querySelector('#breakpoint-condition-title');
  const input = document.querySelector('#breakpoint-condition');
  let line = null;
  function close(restoreFocus = false) {
    const previous = line;
    form.hidden = true;
    line = null;
    if (restoreFocus) gutter.querySelector(`[data-condition-line="${previous}"]`)?.focus();
  }
  function position() {
    if (form.hidden) return;
    const anchor = gutter.querySelector(`[data-condition-line="${line}"]`);
    if (!anchor) { close(); return; }
    const point = anchor.getBoundingClientRect();
    const box = form.getBoundingClientRect();
    form.style.left = `${Math.max(8, Math.min(point.left, innerWidth - box.width - 8))}px`;
    form.style.top = `${Math.max(8, Math.min(point.bottom + box.height + 6 < innerHeight ? point.bottom + 6 : point.top - box.height - 6, innerHeight - box.height - 8))}px`;
  }
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    if (!breakpoints.has(line)) { close(); return; }
    const condition = input.value.trim();
    if (condition) conditions.set(line, condition); else conditions.delete(line);
    onChange();
    close(true);
  });
  document.querySelector('#cancel-breakpoint-condition').addEventListener('click', () => close(true));
  form.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(true); }
    if (event.key === 'Enter') {
      event.stopPropagation();
      if (event.ctrlKey || event.metaKey) { event.preventDefault(); form.requestSubmit(); }
    }
  });
  document.addEventListener('pointerdown', (event) => {
    if (!form.hidden && !form.contains(event.target) && !event.target.closest?.('[data-condition-line]')) close();
  });
  source.addEventListener('input', () => close());
  source.addEventListener('scroll', () => close());
  window.addEventListener('resize', position);
  new MutationObserver(position).observe(gutter, { childList: true, subtree: true });
  return (number) => {
    if (!breakpoints.has(number)) return;
    line = number;
    title.textContent = `Breakpoint at line ${line}`;
    input.value = conditions.get(line) ?? '';
    form.hidden = false;
    position();
    input.focus({ preventScroll: true });
    input.select();
  };
}
