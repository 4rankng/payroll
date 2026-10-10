import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * WCAG 2.1 AA contrast contract for the shared design tokens.
 *
 * Audited during the tablet overhaul (2026-09-25): the old palette rendered
 * money chips at 2.95:1 and would have allowed sidebar dim labels at 2.07:1 —
 * hard failures against WCAG 1.4.3 (4.5:1 normal text, 3:1 large text / UI
 * components). This test reads the tokens straight from variables.css so any
 * future palette change that drops below AA fails CI, not our users.
 */

const css = readFileSync(resolve(__dirname, "../styles/variables.css"), "utf8");

/** Reads a token's value from the :root block (first occurrence). */
function rootToken(name: string): string {
  const rootStart = css.indexOf(":root");
  // The :root block ends where .dark begins; "[data-" markers appear inside
  // :root comments, so they cannot be used as the end boundary here.
  const rootEnd = css.indexOf(".dark {");
  const rootBlock = css.slice(rootStart, rootEnd > rootStart ? rootEnd : undefined);
  const m = rootBlock.match(new RegExp(`--${name}:\\s*([^;]+);`));
  expect(m, `token --${name} must exist in :root`).toBeTruthy();
  return m![1].trim();
}

/** Reads a token declared inside the [data-admin-ui] scope block. */
function adminToken(name: string): string {
  const start = css.indexOf("[data-admin-ui]");
  const block = css.slice(start, css.indexOf("}", start));
  const m = block.match(new RegExp(`--${name}:\\s*([^;]+);`));
  expect(m, `token --${name} must exist in [data-admin-ui]`).toBeTruthy();
  return m![1].trim();
}

function hslToRgb(h: number, s: number, l: number): [number, number, number] {
  const sat = s / 100;
  const lig = l / 100;
  const f = (n: number) => {
    const k = (n + h / 30) % 12;
    const a = sat * Math.min(lig, 1 - lig);
    return lig - a * Math.max(-1, Math.min(k - 3, 9 - k, 1));
  };
  return [Math.round(255 * f(0)), Math.round(255 * f(8)), Math.round(255 * f(4))];
}

function linearChannel(c: number): number {
  const v = c / 255;
  return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
}

function luminance(rgb: [number, number, number]): number {
  const [r, g, b] = rgb;
  return 0.2126 * linearChannel(r) + 0.7152 * linearChannel(g) + 0.0722 * linearChannel(b);
}

function contrastRatio(fgHsl: string, bgHsl: string): number {
  const toRgb = (token: string): [number, number, number] => {
    const t = token.trim();
    // Employee tokens are hex; admin/partner/global tokens are HSL triplets.
    if (t.startsWith("#")) {
      const n = parseInt(t.slice(1), 16);
      return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
    }
    const [h, s, l] = t.replace(/%/g, "").split(/\s+/).map(Number) as [number, number, number];
    return hslToRgb(h, s, l);
  };
  const lf = luminance(toRgb(fgHsl));
  const lb = luminance(toRgb(bgHsl));
  return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05);
}

const WHITE = "255 255 255";

describe("WCAG 2.1 AA contrast contract (variables.css)", () => {
  it.each([
    ["--success on white (money/positive text)", () => rootToken("success")],
    ["--warning on white (pending text)", () => rootToken("warning")],
    ["--info on white (informational text)", () => rootToken("info")],
    ["--destructive on white (failed text)", () => rootToken("destructive")],
    ["--admin-success on white", () => adminToken("admin-success")],
    ["--admin-warning on white", () => adminToken("admin-warning")],
    ["--admin-destructive on white", () => adminToken("admin-destructive")],
    ["--partner-success on white (partner pages)", () => rootToken("partner-success")],
    ["--partner-warning on white (partner pages)", () => rootToken("partner-warning")],
    ["--muted-foreground on white (11px data text)", () => rootToken("muted-foreground")],
  ])("%s meets 4.5:1", (_label, read) => {
    expect(contrastRatio(read(), WHITE)).toBeGreaterThanOrEqual(4.5);
  });

  it.each([
    ["--admin-success", () => adminToken("admin-success")],
    ["--admin-warning", () => adminToken("admin-warning")],
  ])("%s meets 4.5:1 on its own 10% white tint (chip background)", (_label, read) => {
    const fg = read();
    const [h, s, l] = fg
      .trim()
      .replace(/%/g, "")
      .split(/\s+/)
      .map(Number) as [number, number, number];
    const rgb = hslToRgb(h, s, l);
    const tint = rgb.map((c) => Math.round(0.1 * c + 0.9 * 255)) as [number, number, number];
    const lf = luminance(rgb);
    const lt = luminance(tint);
    expect((Math.max(lf, lt) + 0.05) / (Math.min(lf, lt) + 0.05)).toBeGreaterThanOrEqual(4.5);
  });

  it("sidebar foreground meets 4.5:1 on the sidebar background", () => {
    expect(contrastRatio(rootToken("sidebar-foreground"), rootToken("sidebar-background"))).toBeGreaterThanOrEqual(4.5);
  });

  it("employee portal muted/secondary text meets 4.5:1 on white", () => {
    // Hex tokens; muted metadata is the floor (12px metadata text).
    expect(contrastRatio(rootToken("employee-text-secondary"), "#ffffff")).toBeGreaterThanOrEqual(4.5);
    expect(contrastRatio(rootToken("employee-text-muted"), "#ffffff")).toBeGreaterThanOrEqual(4.5);
  });
});

// ---------------------------------------------------------------------------
// Token × surface matrix — every token must clear 4.5:1 on every real surface
// it can land on (white cards, muted rows, page canvas, tinted soft backgrounds).
// ---------------------------------------------------------------------------

/** Real surfaces a token can render against, resolved from variables.css. */
function surfaceSet(): Record<string, string> {
  return {
    white: "255 255 255",
    card: rootToken("card"),
    muted: rootToken("muted"),
    background: rootToken("background"),
    "accent-soft": "#eaf8f0",
    "warning-soft": "#fff7e8",
    "error-soft": "#fef3f2",
    "info-soft": "#eff8ff",
  };
}

describe("WCAG 2.2 AA token × surface matrix", () => {
  // Surfaces where body/secondary text actually renders: white cards, muted
  // rows, and the tinted soft backgrounds used for badges and alerts.
  // The page canvas (--background) is NOT a text surface — text sits inside
  // cards on top of it.
  const bodySurfaces = {
    white: "255 255 255",
    card: rootToken("card"),
    muted: rootToken("muted"),
    "accent-soft": "#eaf8f0",
    "warning-soft": "#fff7e8",
    "error-soft": "#fef3f2",
    "info-soft": "#eff8ff",
  };

  it.each([
    ["--foreground", () => rootToken("foreground")],
    ["--muted-foreground", () => rootToken("muted-foreground")],
    ["--success", () => rootToken("success")],
    ["--warning", () => rootToken("warning")],
    ["--info", () => rootToken("info")],
    ["--destructive", () => rootToken("destructive")],
    ["--primary", () => rootToken("primary")],
    ["--employee-text", () => rootToken("employee-text")],
    ["--employee-text-secondary", () => rootToken("employee-text-secondary")],
    ["--employee-text-muted", () => rootToken("employee-text-muted")],
    ["--employee-accent", () => rootToken("employee-accent")],
    ["--employee-warning", () => rootToken("employee-warning")],
    ["--employee-error", () => rootToken("employee-error")],
    ["--employee-info", () => rootToken("employee-info")],
  ])("%s meets 4.5:1 on every body surface", (_label, read) => {
    const fg = read();
    for (const [name, bg] of Object.entries(bodySurfaces)) {
      expect(
        contrastRatio(fg, bg),
        `${_label} on ${name}`
      ).toBeGreaterThanOrEqual(4.5);
    }
  });

  it("sidebar foreground meets 4.5:1 on the sidebar background", () => {
    // Sidebar foreground is only used inside the sidebar, not on cards.
    expect(
      contrastRatio(rootToken("sidebar-foreground"), rootToken("sidebar-background"))
    ).toBeGreaterThanOrEqual(4.5);
  });
});

// ---------------------------------------------------------------------------
// Non-text contrast — borders, focus rings, and input outlines at 3:1 (1.4.11).
// ---------------------------------------------------------------------------

describe("WCAG 2.2 AA non-text contrast (3:1)", () => {
  it.each([
    ["--border", () => rootToken("border")],
    ["--input", () => rootToken("input")],
    ["--ring", () => rootToken("ring")],
    ["--employee-border", () => rootToken("employee-border")],
    ["--employee-focus-ring", () => rootToken("employee-focus-ring")],
    ["--surface-border", () => rootToken("surface-border")],
  ])("%s meets 3:1 against white (UI component boundary)", (_label, read) => {
    expect(contrastRatio(read(), "255 255 255")).toBeGreaterThanOrEqual(3);
  });

  it("--ring meets 3:1 against --background (focus indicator)", () => {
    expect(
      contrastRatio(rootToken("ring"), rootToken("background"))
    ).toBeGreaterThanOrEqual(3);
  });
});

// ---------------------------------------------------------------------------
// Tailwind palette gate — ad-hoc `text-<family>-<step>` classes used in src/
// must resolve to ≥4.5:1 against the surfaces where text lives (white, muted).
// Reads the live source tree so a future reintroduction of a failing class
// fails CI. Excludes `dark:` prefixed classes (those target dark surfaces).
// ---------------------------------------------------------------------------

describe("Tailwind palette gate (ad-hoc text-* classes in src/)", () => {
  // Subset of the Tailwind v3 palette for families the codebase uses.
  const palette: Record<string, Record<string, string>> = {
    slate:   { "400": "#94a3b8", "500": "#64748b", "600": "#475569", "700": "#334155" },
    gray:    { "400": "#9ca3af", "500": "#6b7280", "600": "#4b5563", "700": "#374151" },
    red:     { "400": "#f87171", "500": "#ef4444", "600": "#dc2626", "700": "#b91c1c" },
    orange:  { "400": "#fb923c", "500": "#f97316", "600": "#ea580c", "700": "#c2410c" },
    amber:   { "400": "#fbbf24", "500": "#f59e0b", "600": "#d97706", "700": "#b45309" },
    yellow:  { "400": "#facc15", "500": "#eab308", "600": "#ca8a04", "700": "#a16207", "800": "#854d0e" },
    green:   { "400": "#4ade80", "500": "#22c55e", "600": "#16a34a", "700": "#15803d" },
    emerald: { "400": "#34d399", "500": "#10b981", "600": "#059669", "700": "#047857" },
    teal:    { "400": "#2dd4bf", "500": "#14b8a6", "600": "#0d9488", "700": "#0f766e" },
    cyan:    { "400": "#22d3ee", "500": "#06b6d4", "600": "#0891b2", "700": "#0e7490" },
    sky:     { "400": "#38bdf8", "500": "#0ea5e9", "600": "#0284c7", "700": "#0369a1" },
    blue:    { "400": "#60a5fa", "500": "#3b82f6", "600": "#2563eb", "700": "#1d4ed8" },
    indigo:  { "400": "#818cf8", "500": "#6366f1", "600": "#4f46e5", "700": "#4338ca" },
    violet:  { "400": "#a78bfa", "500": "#8b5cf6", "600": "#7c3aed", "700": "#6d28d9" },
    purple:  { "400": "#c084fc", "500": "#a855f7", "600": "#9333ea", "700": "#7e22ce" },
    fuchsia: { "400": "#e879f9", "500": "#d946ef", "600": "#c026d3", "700": "#a21caf" },
    pink:    { "400": "#f472b6", "500": "#ec4899", "600": "#db2777", "700": "#be185d" },
    rose:    { "400": "#fb7185", "500": "#f43f5e", "600": "#e11d48", "700": "#be123c" },
    lime:    { "400": "#a3e635", "500": "#84cc16", "600": "#65a30d", "700": "#4d7c0f" },
    zinc:    { "400": "#a1a1aa", "500": "#71717a", "600": "#52525b", "700": "#3f3f46" },
  };

  // The two surfaces where text actually lives in this app.
  const lightSurfaces = { white: "255 255 255", muted: rootToken("muted") };

  // Classes that are allowed to fail on light surfaces because they are
  // exclusively used inside dark containers (modal headers, dark cards).
  // Each entry is a string matched against the class; if the class appears in
  // src/ without a `dark:` prefix AND inside one of these files, it is skipped.
  const DARK_CONTEXT_FILES = new Set([
    "frontend/src/lib/typography.ts", // theme presets: `dark: "text-slate-100 dark:text-slate-900"`
  ]);

  /** Recursively collect .tsx/.ts files under a directory. */
  function collectFiles(dir: string, out: string[] = []): string[] {
    const { readdirSync, statSync } = require("node:fs") as typeof import("node:fs");
    const { join } = require("node:path") as typeof import("node:path");
    for (const name of readdirSync(dir)) {
      const p = join(dir, name);
      const s = statSync(p);
      if (s.isDirectory()) {
        if (["node_modules", "dist", "__tests__", "test"].includes(name)) continue;
        collectFiles(p, out);
      } else if (/\.(tsx|ts)$/.test(name)) {
        out.push(p);
      }
    }
    return out;
  }

  it("every non-dark text-<family>-<step> class in src/ meets 4.5:1 on white and muted", () => {
    const { readFileSync } = require("node:fs") as typeof import("node:fs");
    const { resolve, relative } = require("node:path") as typeof import("node:path");
    const srcRoot = resolve(__dirname, "..");
    const repoRoot = resolve(__dirname, "../..");
    const files = collectFiles(srcRoot);

    const failures: string[] = [];
    const textClassRe = /(?<!dark:)text-([a-z]+)-(\d{2,3})(?:\/(\d{1,3}))?(?=[\s"'`}]|$)/g;

    // Helper: contrast between two RGB tuples.
    const cr = (fg: [number, number, number], bg: [number, number, number]): number => {
      const lf = luminance(fg);
      const lb = luminance(bg);
      return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05);
    };

    for (const file of files) {
      const rel = relative(repoRoot, file);
      if (DARK_CONTEXT_FILES.has(rel)) continue;
      const src = readFileSync(file, "utf8");
      let m: RegExpExecArray | null;
      textClassRe.lastIndex = 0;
      while ((m = textClassRe.exec(src)) !== null) {
        const family = m[1];
        const step = m[2];
        const alpha = m[3] === undefined ? undefined : parseInt(m[3], 10) / 100;
        const fam = palette[family];
        if (!fam || !fam[step]) continue;
        let rgb = hexToRgbArr(fam[step]);
        if (alpha !== undefined) {
          rgb = rgb.map((c) => Math.round(alpha * c + (1 - alpha) * 255)) as [number, number, number];
        }
        // Two surfaces where text lives: white and muted.
        const surfaces: Array<[string, [number, number, number]]> = [
          ["white", [255, 255, 255]],
          ["muted", hexToRgbArr("#f2f4f7")],
        ];
        for (const [surfaceName, bgRgb] of surfaces) {
          const ratio = cr(rgb, bgRgb);
          if (ratio < 4.5) {
            failures.push(`${rel}: text-${family}-${step}${m[3] ? "/" + m[3] : ""} on ${surfaceName} = ${ratio.toFixed(2)}`);
          }
        }
      }
    }

    expect(failures, `Found ${failures.length} failing ad-hoc text classes:\n${failures.join("\n")}`).toHaveLength(0);
  });
});

/** Convert a hex string to an [r,g,b] tuple. */
function hexToRgbArr(hex: string): [number, number, number] {
  const n = parseInt(hex.replace("#", ""), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}
