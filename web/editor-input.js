// Work from the text before the selection so splitting a line or replacing a
// selection indents the new line relative to where the newline is inserted.
export function newlineText(text, cursor) {
  const prefix = text.slice(text.lastIndexOf('\n', cursor - 1) + 1, cursor);
  const indentation = prefix.match(/^[\t ]*/)[0];
  // Ignore strings and trailing comments when identifying the header's colon.
  const structural = prefix.replace(/"(?:\\.|[^"\\])*(?:"|$)|'(?:\\.|[^'\\])*(?:'|$)|#[^\n]*/g, '');
  const block = /^\s*(?:async\s+)?(?:if|elif|else|for|while|def|class|try|except|finally|with|match|case)\b/.test(structural)
    && structural.trimEnd().endsWith(':');
  return '\n' + indentation + (block ? '    ' : '');
}

export function dedentLines(text, selectionStart, selectionEnd) {
  const first = selectionStart === 0 ? 0 : text.lastIndexOf('\n', selectionStart - 1) + 1;
  const removals = [];
  let last = first;
  for (let start = first; start <= text.length;) {
    // A selection ending at the next line's start does not include that line.
    if (start !== first && start >= selectionEnd) break;
    const newline = text.indexOf('\n', start);
    const end = newline < 0 ? text.length : newline;
    let count = 0;
    let columns = 0;
    while (start + count < end && columns < 4) {
      const char = text[start + count];
      if (char === ' ') columns++;
      else if (char === '\t') columns += 4 - columns % 4;
      else break;
      count++;
    }
    if (count) removals.push({ start, end: start + count });
    last = end;
    if (newline < 0) break;
    start = newline + 1;
  }
  if (!removals.length) return null;
  let replacement = text.slice(first, last);
  for (const removal of [...removals].reverse()) {
    replacement = replacement.slice(0, removal.start - first) + replacement.slice(removal.end - first);
  }
  const adjusted = (position) => {
    let removed = 0;
    for (const removal of removals) {
      if (position <= removal.start) break;
      if (position < removal.end) return removal.start - removed;
      removed += removal.end - removal.start;
    }
    return position - removed;
  };
  return { start: first, end: last, text: replacement, selectionStart: adjusted(selectionStart), selectionEnd: adjusted(selectionEnd) };
}
