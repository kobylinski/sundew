// Display timestamps in the browser's local timezone. Captured bodies stay literal.
export const time = (value) => new Intl.DateTimeFormat(undefined, {
  hour: '2-digit', minute: '2-digit',
}).format(new Date(value));
export const dateTime = (value) => new Intl.DateTimeFormat(undefined, {
  day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit',
  minute: '2-digit', second: '2-digit', timeZoneName: 'short',
}).format(new Date(value));
export function rawRequest({ exchange: x }) {
  if (!x.method) return '';
  return [x.method + ' ' + x.path + (x.query ? '?' + x.query : '') + ' HTTP/1.1',
    ...Object.entries(x.header).flatMap(([key, values]) => values.map((value) => key + ': ' + value)),
    '', x.body ?? ''].join('\n');
}
export function rawResponse({ exchange: x }) {
  if (!x.response_status) return '';
  return ['HTTP/1.1 ' + x.response_status,
    ...Object.entries(x.response_header).flatMap(([key, values]) => values.map((value) => key + ': ' + value)),
    '', x.response_body ?? ''].join('\n');
}
