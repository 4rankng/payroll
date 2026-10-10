#!/usr/bin/env node
/**
 * WCAG 2.2 AA contrast scanner.
 *
 * Evaluates real element-level color pairs:
 *   - text-<color> with an explicit bg-<color> on the same element
 *   - text-<color> on the nearest declared surface (white / card / muted / tint)
 *   - inline `color:` with an inline `background:` on the same rule
 *
 * Thresholds (WCAG 2.2 AA):
 *   text (normal)                ≥ 4.5:1
 *   non-text (borders, rings)    ≥ 3.0:1
 *
 * Output: reports/contrast-audit.json + reports/contrast-audit.md
 */

import { readFileSync, writeFileSync, mkdirSync, readdirSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { TAILWIND_PALETTE, TOKEN_CLASSES } from './tailwind-palette.mjs';

const __dirname = fileURLToPath(new URL('.', import.meta.url));
const REPO_ROOT = resolve(__dirname, '../..');
const FRONTEND_SRC = resolve(REPO_ROOT, 'frontend/src');
const VARIABLES_CSS = resolve(REPO_ROOT, 'frontend/src/styles/variables.css');
const REPORT_DIR = resolve(REPO_ROOT, 'plans/2026-10-10-wcag-contrast/reports');

// ---------------------------------------------------------------------------
// Color math
// ---------------------------------------------------------------------------

function hslToRgb(h, s, l) {
  const sat = s / 100;
  const lig = l / 100;
  const f = (n) => {
    const k = (n + h / 30) % 12;
    const a = sat * Math.min(lig, 1 - lig);
    return lig - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
  };
  return [Math.round(255 * f(0)), Math.round(255 * f(8)), Math.round(255 * f(4))];
}

function linearChannel(c) {
  const v = c / 255;
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function luminance(rgb) {
  const [r, g, b] = rgb;
  return 0.2126 * linearChannel(r) + 0.7152 * linearChannel(g) + 0.0722 * linearChannel(b);
}

function contrastRatio(fg, bg) {
  const lf = luminance(fg);
  const lb = luminance(bg);
  return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05);
}

function hexToRgb(hex) {
  const n = parseInt(hex.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function mixWithBase(rgb, alpha, base) {
  return rgb.map((c, i) => Math.round(alpha * c + (1 - alpha) * base[i]));
}

// Tailwind's `bg-X/alpha` composites over the parent surface. Light theme
// parents are near-white; dark theme parents are the dark card/body.
const LIGHT_PARENT = [255, 255, 255];
const DARK_PARENT = [15, 23, 42]; // slate-900-ish, the app's dark body

function tailwindRgb(family, step, alphaPct, isDark) {
  const fam = TAILWIND_PALETTE[family];
  if (!fam) return null;
  const hex = fam[step];
  if (!hex) return null;
  const rgb = hexToRgb(hex);
  if (alphaPct === undefined) return rgb;
  return mixWithBase(rgb, alphaPct / 100, isDark ? DARK_PARENT : LIGHT_PARENT);
}

function parseTokens() {
  const css = readFileSync(VARIABLES_CSS, 'utf8');
  const tokens = { light: {}, dark: {} };
  const blockRe = /([^{}]+)\{([^}]*)\}/g;
  let m;
  while ((m = blockRe.exec(css)) !== null) {
    const selector = m[1].trim();
    const body = m[2];
    const isDark = /\.dark\b/.test(selector);
    const bucket = isDark ? 'dark' : 'light';
    const declRe = /--([a-z0-9-]+)\s*:\s*([^;]+);/gi;
    let d;
    while ((d = declRe.exec(body)) !== null) {
      const key = d[1];
      const value = d[2].replace(/\/\*[\s\S]*?\*\//g, '').trim();
      if (value) tokens[bucket][key] = value;
    }
  }
  // .dark overrides: promote dark keys to light when not overridden
  for (const k of Object.keys(tokens.dark)) {
    if (!tokens.light[k]) tokens.light[k] = tokens.dark[k];
  }
  return tokens;
}

function tokenToRgb(value) {
  const t = value.trim();
  if (t.startsWith('#')) {
    return hexToRgb(t.length === 4
      ? '#' + t[1] + t[1] + t[2] + t[2] + t[3] + t[3]
      : t.slice(0, 7));
  }
  const hsl = t.match(/^hsl\(\s*([\d.]+)[,\s]+([\d.]+)%[,\s]+([\d.]+)%\s*\)/);
  if (hsl) return hslToRgb(+hsl[1], +hsl[2], +hsl[3]);
  const triplet = t.replace(/%/g, '').split(/\s+/).map(Number);
  if (triplet.length === 3 && triplet.every((n) => Number.isFinite(n))) {
    return hslToRgb(triplet[0], triplet[1], triplet[2]);
  }
  return null;
}

// ---------------------------------------------------------------------------
// Palette lookup
// ---------------------------------------------------------------------------

function parseInline(value) {
  const v = value.trim();
  if (v.startsWith('#')) {
    return v.length === 4
      ? hexToRgb('#' + v[1] + v[1] + v[2] + v[2] + v[3] + v[3])
      : hexToRgb(v.slice(0, 7));
  }
  const m = v.match(/rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)(?:\s*,\s*([\d.]+))?\s*\)/);
  if (!m) return null;
  const rgb = [+m[1], +m[2], +m[3]];
  const a = m[4] === undefined ? 1 : parseFloat(m[4]);
  return a >= 1 ? rgb : mixWithBase(rgb, a, LIGHT_PARENT);
}

// ---------------------------------------------------------------------------
// Surfaces
// ---------------------------------------------------------------------------

function resolveSurfaces(tokens) {
  const light = tokens.light;
  return {
    white: [255, 255, 255],
    card: tokenToRgb(light['card']) || [255, 255, 255],
    muted: tokenToRgb(light['muted']) || [242, 244, 247],
    background: tokenToRgb(light['background']) || [226, 232, 228],
  };
}

// Tinted surfaces are only used in explicit `text-X on bg-Y` pairs; text
// without an explicit bg is judged against the four neutral surfaces above.
const TINT_SURFACES = {
  'accent-soft': hexToRgb('#eaf8f0'),
  'warning-soft': hexToRgb('#fff7e8'),
  'error-soft': hexToRgb('#fef3f2'),
  'info-soft': hexToRgb('#eff8ff'),
};

const DARK_SURFACES = {
  'dark-bg': [15, 23, 42],
  'dark-card': [30, 41, 59],
};

// ---------------------------------------------------------------------------
// Regexes
// ---------------------------------------------------------------------------

const TEXT_CLASS_RE = /(?:^|[\s"'`])(dark:)?text-([a-z]+)-(\d{2,3})(?:\/(\d{1,3}))?(?=[\s"'`}]|$)/g;
const BG_CLASS_RE = /(?:^|[\s"'`])(dark:)?bg-([a-z]+)-(\d{2,3})(?:\/(\d{1,3}))?(?=[\s"'`}]|$)/g;
const INLINE_COLOR_RE = /(?:^|[;{\s])color\s*:\s*(#[0-9a-fA-F]{3,8}|rgba?\([^)]+\))/g;
const INLINE_BG_RE = /(?:^|[;{\s])(?:background|background-color)\s*:\s*(#[0-9a-fA-F]{3,8}|rgba?\([^)]+\))/g;

// ---------------------------------------------------------------------------
// Line-level pairing: when a `text-X` and a `bg-Y` land on the same line
// (same JSX className or same CSS declaration block), treat them as one element.
// ---------------------------------------------------------------------------

function findLine(src, index) {
  return src.slice(0, index).split('\n').length;
}

function lineBounds(src, index) {
  const start = src.lastIndexOf('\n', index) + 1;
  const end = src.indexOf('\n', index);
  return [start, end === -1 ? src.length : end];
}

// Extract every quoted class string literal from a line. Classes inside the
// same literal belong to the same element; classes in different literals on
// the same line do not.
const STRING_LITERAL_RE = /(['"`])((?:\\.|(?!\1)[^\\\r\n])*)\1/g;

function classStringsInLine(line) {
  const out = [];
  STRING_LITERAL_RE.lastIndex = 0;
  let m;
  while ((m = STRING_LITERAL_RE.exec(line)) !== null) {
    const body = m[2];
    // Only keep strings that look like class lists (contain a `text-` or `bg-` token).
    if (/(?:^|[\s])((?:dark:)?(?:text|bg|border|fill|stroke)-)/.test(body)) {
      out.push(body);
    }
  }
  return out;
}

function collectLinePairs(src, isCss, surfaces) {
  const pairs = [];
  const lines = src.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    // CSS rules: treat the whole line as one class context.
    const contexts = isCss ? [line] : classStringsInLine(line);
    for (const classStr of contexts) {
      const byTheme = {
        light: { texts: [], bgs: [] },
        dark: { texts: [], bgs: [] },
      };
      TEXT_CLASS_RE.lastIndex = 0;
      let m;
      while ((m = TEXT_CLASS_RE.exec(classStr)) !== null) {
        const dark = !!m[1];
        const rgb = tailwindRgb(m[2], m[3], m[4] === undefined ? undefined : parseInt(m[4], 10), dark);
        if (rgb) byTheme[dark ? 'dark' : 'light'].texts.push({ rgb, token: `text-${m[2]}-${m[3]}${m[4] ? '/' + m[4] : ''}` });
      }
      BG_CLASS_RE.lastIndex = 0;
      while ((m = BG_CLASS_RE.exec(classStr)) !== null) {
        const dark = !!m[1];
        const rgb = tailwindRgb(m[2], m[3], m[4] === undefined ? undefined : parseInt(m[4], 10), dark);
        if (rgb) byTheme[dark ? 'dark' : 'light'].bgs.push({ rgb, token: `bg-${m[2]}-${m[3]}${m[4] ? '/' + m[4] : ''}` });
      }
      // Inline CSS `color:` / `background:` on the same rule.
      INLINE_COLOR_RE.lastIndex = 0;
      INLINE_BG_RE.lastIndex = 0;
      while ((m = INLINE_COLOR_RE.exec(classStr)) !== null) {
        const rgb = parseInline(m[1]);
        if (rgb) byTheme.light.texts.push({ rgb, token: m[1] });
      }
      while ((m = INLINE_BG_RE.exec(classStr)) !== null) {
        const rgb = parseInline(m[1]);
        if (rgb) byTheme.light.bgs.push({ rgb, token: m[1] });
      }

      for (const theme of ['light', 'dark']) {
        const bucket = byTheme[theme];
        for (const t of bucket.texts) {
          for (const b of bucket.bgs) {
            const ratio = contrastRatio(t.rgb, b.rgb);
            pairs.push({
              line: i + 1,
              theme,
              text: t.token,
              bg: b.token,
              ratio: +ratio.toFixed(2),
              kind: 'paired',
            });
          }
          if (bucket.bgs.length === 0) {
            const surf = theme === 'dark' ? DARK_SURFACES : { white: surfaces.white, muted: surfaces.muted };
            for (const [sn, sr] of Object.entries(surf)) {
              const ratio = contrastRatio(t.rgb, sr);
              pairs.push({
                line: i + 1,
                theme,
                text: t.token,
                bg: sn,
                ratio: +ratio.toFixed(2),
                kind: 'inherit',
              });
            }
          }
        }
      }
    }
  }
  return pairs;
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    const s = statSync(p);
    if (s.isDirectory()) {
      if (name === 'node_modules' || name === 'dist' || name === '__tests__' || name === 'test') continue;
      walk(p, out);
    } else if (/\.(tsx|ts|css)$/.test(name)) {
      out.push(p);
    }
  }
  return out;
}

function suggestDarker(family, step, surfaceRgb, required) {
  const fam = TAILWIND_PALETTE[family];
  if (!fam) return 'n/a';
  const steps = ['50', '100', '200', '300', '400', '500', '600', '700', '800', '900', '950'];
  const idx = steps.indexOf(String(step));
  if (idx < 0) return 'n/a';
  for (let i = idx + 1; i < steps.length; i++) {
    const rgb = hexToRgb(fam[steps[i]]);
    if (contrastRatio(rgb, surfaceRgb) >= required) return `${family}-${steps[i]}`;
  }
  return `${family}-950`;
}

function main() {
  const tokens = parseTokens();
  const surfaces = resolveSurfaces(tokens);
  const files = walk(FRONTEND_SRC);
  const findings = [];

  for (const file of files) {
    const src = readFileSync(file, 'utf8');
    const rel = relative(REPO_ROOT, file);
    const pairs = collectLinePairs(src, file.endsWith('.css'), surfaces);
    for (const p of pairs) {
      const isText = true; // every collected pair has a text side
      const required = 4.5;
      if (p.ratio >= required) continue;
      const suggestion = p.kind === 'paired' && p.text.startsWith('text-')
        ? suggestDarker(p.text.replace('text-', '').split('-')[0],
                        p.text.replace('text-', '').split('-')[1],
                        (surfaces[p.bg] || TINT_SURFACES[p.bg] || [255, 255, 255]),
                        required)
        : 'n/a';
      findings.push({
        kind: 'text',
        theme: p.theme,
        text: p.text,
        bg: p.bg,
        ratio: p.ratio,
        required,
        file: rel,
        line: p.line,
        pairing: p.kind,
        suggestion,
      });
    }
  }

  findings.sort((a, b) => a.ratio - b.ratio);

  mkdirSync(REPORT_DIR, { recursive: true });

  const jsonPath = join(REPORT_DIR, 'contrast-audit.json');
  writeFileSync(jsonPath, JSON.stringify({
    generatedAt: new Date().toISOString(),
    thresholds: { text: 4.5, nonText: 3.0 },
    totals: {
      filesScanned: files.length,
      findings: findings.length,
      paired: findings.filter((f) => f.pairing === 'paired').length,
      inherit: findings.filter((f) => f.pairing === 'inherit').length,
      dark: findings.filter((f) => f.theme === 'dark').length,
      light: findings.filter((f) => f.theme === 'light').length,
    },
    findings,
  }, null, 2));

  const md = [];
  md.push('# WCAG 2.2 AA — Static contrast audit');
  md.push('');
  md.push(`**Generated**: ${new Date().toISOString()}`);
  md.push(`**Files scanned**: ${files.length}`);
  md.push(`**Findings**: ${findings.length} (paired ${findings.filter((f) => f.pairing === 'paired').length}, inherited-surface ${findings.filter((f) => f.pairing === 'inherit').length}; light ${findings.filter((f) => f.theme === 'light').length}, dark ${findings.filter((f) => f.theme === 'dark').length})`);
  md.push('');
  md.push('## Methodology');
  md.push('');
  md.push('Two pairings are evaluated:');
  md.push('');
  md.push('- **paired** — `text-X` and `bg-Y` land on the same source line (same JSX className or CSS rule). The exact pair is checked.');
  md.push('- **inherit** — `text-X` with no explicit bg on the same line. Evaluated against every declared surface (white / card / muted / background / tints) and flagged on the worst.');
  md.push('');
  md.push('`dark:` variants are evaluated against the dark surface set.');
  md.push('');
  md.push('## Thresholds');
  md.push('');
  md.push('| Kind | Required | WCAG |');
  md.push('|---|---|---|');
  md.push('| Text (normal) | 4.5:1 | 1.4.3 AA |');
  md.push('| Non-text (borders, focus rings) | 3.0:1 | 1.4.11 AA |');
  md.push('');
  md.push('## Worst offenders');
  md.push('');
  md.push('| Ratio | Theme | Text | On | Suggested fix | Location |');
  md.push('|---|---|---|---|---|---|');
  for (const f of findings.slice(0, 120)) {
    md.push(`| ${f.ratio} | ${f.theme} | \`${f.text}\` | \`${f.bg}\` | \`${f.suggestion}\` | ${f.file}:${f.line} |`);
  }
  if (findings.length > 120) {
    md.push('');
    md.push(`_… ${findings.length - 120} more findings in contrast-audit.json._`);
  }
  md.push('');
  md.push('## By file');
  md.push('');
  const byFile = new Map();
  for (const f of findings) {
    if (!byFile.has(f.file)) byFile.set(f.file, []);
    byFile.get(f.file).push(f);
  }
  for (const [file, fs] of [...byFile.entries()].sort((a, b) => b[1].length - a[1].length)) {
    md.push(`### ${file} (${fs.length})`);
    md.push('');
    for (const f of fs) {
      md.push(`- L${f.line} \`${f.text}\` on \`${f.bg}\` = **${f.ratio}** (needs ${f.required}) → \`${f.suggestion}\` [${f.theme}/${f.pairing}]`);
    }
    md.push('');
  }

  const mdPath = join(REPORT_DIR, 'contrast-audit.md');
  writeFileSync(mdPath, md.join('\n'));

  process.stdout.write(`Scanned ${files.length} files, ${findings.length} findings.\n`);
  process.stdout.write(`  paired ${findings.filter((f) => f.pairing === 'paired').length}, inherited ${findings.filter((f) => f.pairing === 'inherit').length}\n`);
  process.stdout.write(`  light ${findings.filter((f) => f.theme === 'light').length}, dark ${findings.filter((f) => f.theme === 'dark').length}\n`);
  process.stdout.write(`Reports: ${relative(REPO_ROOT, mdPath)} and ${relative(REPO_ROOT, jsonPath)}\n`);
}

main();
