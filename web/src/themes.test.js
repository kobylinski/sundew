import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import postcss from 'postcss';

// postcss is already in the pinned Vite toolchain; no test dependency is added.
const accepted = postcss.parse(await readFile(new URL('../../prototypes/web-ui/src/style.css', import.meta.url), 'utf8'));
// src/ -> web/ -> repository root for both the source and embedded build.
const output = new URL('../../internal/ui/static/assets/', import.meta.url);
const cssNames = (await readdir(output)).filter(name => name.endsWith('.css'));
const built = postcss.parse((await Promise.all(cssNames.map(name => readFile(new URL(name, output), 'utf8')))).join('\n'));
function canonical(value) {
  return value.replace(/#[0-9a-f]{3,8}\b/gi, hex => {
    const digits = hex.slice(1).toLowerCase();
    return '#' + (digits.length === 3 || digits.length === 4 ? [...digits].map(c => c+c).join('') : digits);
  }).replace(/\s+/g, '');
}
function palette(css, selector, dark = false, source = false) {
  let properties;
  css.walkRules(selector, rule => {
    let inDarkMedia = false;
    for (let parent = rule.parent; parent; parent = parent.parent) {
      if (parent.type === 'atrule' && parent.name === 'media' && /prefers-color-scheme\s*:\s*dark/.test(parent.params)) inDarkMedia = true;
    }
    if (source || inDarkMedia === dark) {
      const tokens = {};
      rule.walkDecls(/^--/, declaration => {
        // Vite emits empty/initial custom properties for color-scheme support;
        // they are compiler switches, not palette colors.
        if (declaration.value.trim() && declaration.value.trim() !== 'initial') tokens[declaration.prop] = canonical(declaration.value);
      });
      if (Object.keys(tokens).length) properties = tokens;
    }
  });
  return properties;
}
test('built CSS has complete automatic palettes equal to the accepted prototype', () => {
  const expectedLight = palette(accepted, '.app', false, true);
  const expectedDark = palette(accepted, '.app[data-theme="dark"]', true, true);
  const light = palette(built, '.app'); const dark = palette(built, '.app', true);
  assert.ok(light && dark, 'both palettes must be present; dark must be inside a system media query');
  assert.deepEqual(Object.keys(light).sort(), Object.keys(dark).sort(), 'every palette token has an opposite-theme value');
  assert.deepEqual(light, expectedLight); assert.deepEqual(dark, expectedDark);
  for (const status of ['queued', 'sent', 'delivered', 'failed', 'received']) {
    const expected = []; const actual = [];
    accepted.walkRules('[data-theme="dark"] .status-' + status, rule => rule.walkDecls(d => expected.push([d.prop, canonical(d.value)])));
    built.walkRules('.status-' + status, rule => {
      if (rule.parent.type === 'atrule' && /prefers-color-scheme\s*:\s*dark/.test(rule.parent.params)) rule.walkDecls(d => actual.push([d.prop, canonical(d.value)]));
    });
    assert.deepEqual(actual, expected, status + ' dark status colors');
  }
});
