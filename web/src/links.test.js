import { test } from 'node:test';
import assert from 'node:assert/strict';
import { linkParts, webHref } from './links.js';

test('only absolute HTTP(S) links; HTML and unsafe schemes stay text', () => {
  for (const value of ['javascript:alert(1)', 'data:text/html,x', '//evil.test', '/api', '<b>x</b>', 'https://']) assert.equal(webHref(value), null);
  assert.equal(webHref('HTTPS://Example.COM/path'), 'https://example.com/path');
  const text = '<img onerror=x> javascript:alert(1) https://example.test/a?q=1&b=2.';
  const parts = linkParts(text);
  assert.equal(parts.map(part => part.text).join(''), text);
  assert.deepEqual(parts.filter(part => part.href).map(part => part.text), ['https://example.test/a?q=1&b=2']);
});
test('sentence punctuation stays literal; balanced URL parentheses survive', () => {
  const text = 'See (https://example.test/a_(b)), then http://example.test/x!';
  const parts = linkParts(text);
  assert.equal(parts.map(p => p.text).join(''), text);
  assert.deepEqual(parts.filter(p => p.href).map(p => p.text), ['https://example.test/a_(b)', 'http://example.test/x']);
});
