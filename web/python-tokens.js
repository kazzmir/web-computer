// Tolerant lexical tokens only; this never evaluates Python source. Strings and
// comments are single tokens, so consumers cannot mistake their text for code.
export const pythonTokenPattern = /(?<comment>#[^\n]*)|(?<string>(?:[rRuUbBfF]{1,2})?(?:"""(?:\\[\s\S]|(?!""")[^\\])*?(?:"""|$)|'''(?:\\[\s\S]|(?!''')[^\\])*?(?:'''|$)|"(?:\\[\s\S]|[^"\\\r\n])*(?:"|(?=\r?\n|$))|'(?:\\[\s\S]|[^'\\\r\n])*(?:'|(?=\r?\n|$))))|(?<number>\b(?:0[xX][\da-fA-F_]+|0[bB][01_]+|0[oO][0-7_]+|\d[\d_]*(?:\.[\d_]*)?(?:[eE][+-]?[\d_]+)?)[jJ]?|\.[\d_]+(?:[eE][+-]?[\d_]+)?[jJ]?)|(?<identifier>[\p{L}_][\p{L}\p{N}_]*)|(?<operator>[-+*/%=<>!&|^~:@]+)|(?<punctuation>[()[\]{},.;])/gu;

export function* pythonTokens(text, start = 0) {
  const lexer = new RegExp(pythonTokenPattern.source, pythonTokenPattern.flags);
  lexer.lastIndex = start;
  for (const match of text.matchAll(lexer)) {
    yield {
      start: match.index,
      end: match.index + match[0].length,
      value: match[0],
      kind: Object.keys(match.groups).find((key) => match.groups[key] !== undefined),
    };
  }
}
