function dottedName(tokens) {
  if (!tokens.length || tokens.length % 2 === 0 || tokens.some((token, i) => i % 2 === 0 ? token.kind !== 'identifier' : token.value !== '.')) return null;
  return tokens.map((token) => token.value).join('');
}

// Extract explicit import bindings without executing an import. Unsupported or
// incomplete syntax returns null so callers can conservatively invalidate it.
export function importBindings(tokens) {
  const from = tokens[0]?.value === 'from';
  if (!from && tokens[0]?.value !== 'import') return null;
  const importIndex = from ? tokens.findIndex((token) => token.value === 'import') : 0;
  if (importIndex < 0) return null;
  const module = from ? dottedName(tokens.slice(1, importIndex)) : '';
  if (from && !module) return null;
  let items = tokens.slice(importIndex + 1);
  if (items[0]?.value === '(' && items.at(-1)?.value === ')') items = items.slice(1, -1);
  if (from && items.some((token) => token.value === '*')) return { wildcard: true, bindings: [] };
  const clauses = [[]];
  for (const token of items) {
    if (token.value === ',') clauses.push([]);
    else clauses.at(-1).push(token);
  }
  const bindings = [];
  for (const clause of clauses) {
    if (!clause.length) continue;
    const as = clause.findIndex((token) => token.value === 'as');
    if (as >= 0 && (as !== clause.length - 2 || clause.at(-1).kind !== 'identifier')) return null;
    const pathTokens = as < 0 ? clause : clause.slice(0, as);
    const path = dottedName(pathTokens);
    if (!path || (from && pathTokens.length !== 1)) return null;
    const name = as < 0 ? pathTokens[0] : clause.at(-1);
    bindings.push({
      name: name.value,
      path: from ? `${module}.${path}` : as < 0 ? pathTokens[0].value : path,
      start: name.start,
    });
  }
  return bindings.length ? { wildcard: false, bindings } : null;
}
