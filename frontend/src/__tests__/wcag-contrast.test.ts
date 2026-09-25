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
  const scoped = css.indexOf("[data-");
  const rootBlock = css.slice(rootStart, scoped > rootStart ? scoped : undefined);
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
  const parse = (token: string) =>
    token
      .trim()
      .replace(/%/g, "")
      .split(/\s+/)
      .map(Number) as [number, number, number];
  const [fh, fs, fl] = parse(fgHsl);
  const [bh, bs, bl] = parse(bgHsl);
  const lf = luminance(hslToRgb(fh, fs, fl));
  const lb = luminance(hslToRgb(bh, bs, bl));
  return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05);
}

const WHITE = "255 255 255";

describe("WCAG 2.1 AA contrast contract (variables.css)", () => {
  it.each([
    ["--success on white (money/positive text)", () => rootToken("success")],
    ["--warning on white (pending text)", () => rootToken("warning")],
    ["--destructive on white (failed text)", () => rootToken("destructive")],
    ["--admin-success on white", () => adminToken("admin-success")],
    ["--admin-warning on white", () => adminToken("admin-warning")],
    ["--admin-destructive on white", () => adminToken("admin-destructive")],
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
});
