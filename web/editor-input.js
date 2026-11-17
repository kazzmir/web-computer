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
