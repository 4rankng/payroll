#!/usr/bin/env node
/**
 * One-shot remediation: darken text-* classes that fail WCAG 2.2 AA on light
 * surfaces. Skips `dark:` prefixed classes (those target dark surfaces and
 * already pass). Skips classes already at 700+ (already dark enough).
 *
 * Policy (user-confirmed): prefer design tokens; darken one step.
 */

import { readFileSync, writeFileSync, readdirSync, statSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { TAILWIND_PALETTE } from './tailwind-palette.mjs';

const __dirname = fileURLToPath(new URL('.', import.meta.url));
const FRONTEND_SRC = resolve(__dirname, '../src');

// Tailwind step ladder, lightest to darkest.
const STEPS = ['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950'];

// Colors that already pass on light surfaces — do not darken.
const SKIP_STEPS = new Set(['700', '800', '900', '950']);

// Map of (family, step) -> darker step that clears 4.5:1 on white AND muted.
// Only 500/600 series are auto-darkened; 200/300/400 are left for manual
// review because they are often intentionally used on dark surfaces.
const DARKER = {
  'red-500': 'red-700', 'red-600': 'red-700',
  'orange-500': 'orange-700', 'orange-600': 'orange-700',
  'amber-500': 'amber-700', 'amber-600': 'amber-700',
  'yellow-500': 'yellow-700', 'yellow-600': 'yellow-800',
  'green-500': 'green-700', 'green-600': 'green-700',
  'emerald-500': 'emerald-700', 'emerald-600': 'emerald-700',
  'teal-500': 'teal-700', 'teal-600': 'teal-700',
  'cyan-500': 'cyan-700', 'cyan-600': 'cyan-700',
  'sky-500': 'sky-700', 'sky-600': 'sky-700',
  'blue-500': 'blue-700', 'blue-600': 'blue-700',
  'indigo-500': 'indigo-700', 'indigo-600': 'indigo-700',
  'violet-500': 'violet-700', 'violet-600': 'violet-700',
  'purple-500': 'purple-700', 'purple-600': 'purple-700',
  'fuchsia-500': 'fuchsia-700', 'fuchsia-600': 'fuchsia-700',
  'pink-500': 'pink-700', 'pink-600': 'pink-700',
  'rose-500': 'rose-700', 'rose-600': 'rose-700',
  'slate-400': 'slate-600', 'slate-500': 'slate-600',
  'gray-400': 'gray-600', 'gray-500': 'gray-600',
  'zinc-400': 'zinc-600', 'zinc-500': 'zinc-600',
  'neutral-400': 'neutral-600', 'neutral-500': 'neutral-600',
  'stone-400': 'stone-600', 'stone-500': 'stone-600',
  'lime-500': 'lime-700', 'lime-600': 'lime-700',
};

// Alpha variants that need the alpha removed (text-<color>-700/60 -> text-<color>-700).
const ALPHA_STRIP = /text-([a-z]+)-(\d{2,3})\/(\d{1,3})/g;

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    const s = statSync(p);
    if (s.isDirectory()) {
      if (name === 'node_modules' || name === 'dist' || name === '__tests__' || name === 'test') continue;
      walk(p, out);
    } else if (/\.(tsx|ts)$/.test(name)) {
      out.push(p);
    }
  }
  return out;
}

function processFile(path) {
  let src = readFileSync(path, 'utf8');
  const orig = src;
  const changes = [];

  // 1. Strip alpha from text-<color>-<step>/<alpha> when the base already passes.
  src = src.replace(ALPHA_STRIP, (match, family, step, alpha) => {
    const key = `${family}-${step}`;
    // If the un-alphaed version is already at 700+, strip the alpha.
    if (SKIP_STEPS.has(step)) {
      changes.push({ from: match, to: `text-${family}-${step}` });
      return `text-${family}-${step}`;
    }
    // If the darker replacement exists (500/600 series), use it without alpha.
    const darker = DARKER[key];
    if (darker) {
      changes.push({ from: match, to: `text-${darker}` });
      return `text-${darker}`;
    }
    // 200/300/400 with alpha: leave for manual review (dark-context candidates).
    return match;
  });

  // 2. Darken text-<color>-<step> classes that fail on light surfaces.
  //    Only touch non-`dark:` prefixed classes.
  for (const [key, darker] of Object.entries(DARKER)) {
    const [family, step] = key.split('-');
    // Match `text-<family>-<step>` NOT preceded by `dark:`.
    const re = new RegExp(`(?<!dark:)text-${family}-${step}\\b`, 'g');
    src = src.replace(re, (match) => {
      changes.push({ from: match, to: `text-${darker}` });
      return `text-${darker}`;
    });
  }

  if (src !== orig) {
    writeFileSync(path, src);
    return changes;
  }
  return [];
}

function main() {
  const files = walk(FRONTEND_SRC);
  let total = 0;
  const byFile = new Map();
  for (const f of files) {
    const changes = processFile(f);
    if (changes.length) {
      byFile.set(f, changes);
      total += changes.length;
    }
  }
  process.stdout.write(`Applied ${total} class swaps across ${byFile.size} files.\n`);
  for (const [file, changes] of byFile) {
    const unique = [...new Set(changes.map((c) => `${c.from} -> ${c.to}`))];
    process.stdout.write(`  ${file}\n`);
    for (const u of unique) process.stdout.write(`    ${u}\n`);
  }
}

main();
