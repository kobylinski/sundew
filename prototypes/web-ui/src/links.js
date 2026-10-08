// Prototype text rendering only. Never interpret a message body as HTML.
export function webHref(value) {
  if (typeof value !== 'string' || !/^https?:\/\//i.test(value)) return null;
  try {
    const parsed = new URL(value);
    return ['http:', 'https:'].includes(parsed.protocol) ? parsed.href : null;
  } catch {
    return null;
  }
}

export function linkParts(text) {
  const parts = [];
  let cursor = 0;
  for (const match of text.matchAll(/https?:\/\/[^\s<>"'`]+/gi)) {
    let label = match[0].replace(/[.,!?;:]+$/, '');
    // Sentence punctuation stays outside the link; balanced URL brackets stay in it.
    for (const [open, close] of [['(', ')'], ['[', ']'], ['{', '}']]) {
      while (label.endsWith(close) && label.split(close).length > label.split(open).length) label = label.slice(0, -1);
    }
    const href = webHref(label);
    if (!href) continue;
    if (match.index > cursor) parts.push({ text: text.slice(cursor, match.index) });
    parts.push({ text: label, href });
    cursor = match.index + label.length;
  }
  if (cursor < text.length) parts.push({ text: text.slice(cursor) });
  return parts;
}
