// Definitive: extract class tokens from UU vendored files, generate CSS with
// the real config, exact-match escaped selectors (handles ::before/[attr] tails
// and classes that share a prefix, e.g. outline vs outline-2).
const fs = require('fs');
const postcss = require('postcss');
const tailwind = require('tailwindcss');

const FILES = [
  'src/components/marketing/banners/banner-dual-action-brand-full-width.tsx',
  'src/components/base/buttons/button.tsx',
  'src/components/base/buttons/close-button.tsx',
  'src/components/foundations/featured-icon/featured-icon.tsx',
];

// Noise words that appear in quoted strings but are not utility classes.
const NOISE = new Set([
  'client', 'loading', 'use', 'round', 'none', 'function', 'true', 'href',
  'slot', 'button', 'color', 'gradient', 'error', 'warning', 'success',
  'primary', 'secondary', 'light', 'dark', 'modern', 'modern-neue',
  'xs', 'sm', 'md', 'lg', 'xl', 'link-color', 'link-destructive', 'link-gray',
  'primary-destructive', 'secondary-destructive', 'tertiary-destructive',
  'react-aria-components',
]);

const tokens = new Set();
for (const f of FILES) {
  const src = fs.readFileSync(f, 'utf8');
  for (const m of src.matchAll(/"([^"]*)"|'([^']*)'/g)) {
    const s = m[1] ?? m[2] ?? '';
    for (const part of s.split(/\s+/)) {
      if (!part) continue;
      if (!/^[a-z&![]/i.test(part)) continue;
      if (NOISE.has(part)) continue;
      if (!/[-:]/.test(part)) continue;
      tokens.add(part);
    }
  }
}
const list = [...tokens].sort();

const esc = (c) => c.replace(/[^A-Za-z0-9_-]/g, (ch) => '\\' + ch);

(async () => {
  const raw = list.map((c) => `<div class="${c}"></div>`).join('');
  const res = await postcss([
    tailwind({ config: './tailwind.config.ts', content: [{ raw, extension: 'html' }] }),
  ]).process('@tailwind utilities;', { from: undefined });

  const css = res.css;
  const missing = [];
  for (const c of list) {
    const needle = '.' + esc(c);
    let found = false;
    let idx = css.indexOf(needle);
    while (idx !== -1) {
      const next = css[idx + needle.length];
      // boundary after the class: selector tail or rule start — but NOT when
      // the match is just the prefix of a longer class (outline vs outline-2).
      if (next === undefined || /[\s:{},[>.~+()]/.test(next)) {
        found = true;
        break;
      }
      idx = css.indexOf(needle, idx + 1);
    }
    if (!found) missing.push(c);
  }
  console.log('TOTAL tokens:', list.length);
  console.log('MISSING (' + missing.length + '):');
  missing.forEach((c) => console.log('  ' + c));
})();
