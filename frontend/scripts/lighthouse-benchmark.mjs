#!/usr/bin/env node
/**
 * Self-improve benchmark for the payroll frontend.
 *
 * Metric: Lighthouse PERFORMANCE score (desktop preset), 0-100, higher is better.
 * Pipeline: pnpm build -> vite preview on :4173 -> Lighthouse x3 (median) -> JSON.
 *
 * Output (last stdout line): {"primary": <0-100>, "sub_scores": {fcp,lcp,tbt,cls,si}}
 *
 * Sealed: this file MUST NOT be modified by the improvement loop.
 */
import { spawn, spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url)); // frontend/scripts
const FRONTEND = dirname(HERE); // frontend
const URL = process.env.LH_URL || "http://localhost:4173/";
const PORT = "4173";
const RUNS = parseInt(process.env.LH_RUNS || "3", 10);

const LH_BIN = [
  join(FRONTEND, "node_modules/.bin/lighthouse"),
  join(FRONTEND, "node_modules/lighthouse/lighthouse-cli/index.js"),
].find((p) => existsSync(p));

function run(cmd, args, opts = {}) {
  const res = spawnSync(cmd, args, { cwd: FRONTEND, encoding: "utf8", ...opts });
  if (res.error) throw res.error;
  return res;
}

function build() {
  const res = run("pnpm", ["build"], { stdio: ["ignore", "pipe", "pipe"] });
  if (res.status !== 0) {
    process.stderr.write(res.stdout || "");
    process.stderr.write(res.stderr || "");
    throw new Error(`pnpm build failed (exit ${res.status})`);
  }
}

async function waitForServer(url, timeoutMs = 30000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(url, { mode: "no-cors" });
      if (r || r.status !== undefined) return true;
    } catch {
      await new Promise((r) => setTimeout(r, 500));
    }
  }
  throw new Error(`server ${url} did not become ready in ${timeoutMs}ms`);
}

function runLighthouse(outPath) {
  const args = [
    URL,
    "--only-categories=performance",
    "--preset=desktop",
    "--output=json",
    `--output-path=${outPath}`,
    "--quiet",
    "--max-wait-for-load=60000",
    "--chrome-flags=--headless=new --no-sandbox --disable-gpu --no-first-run --no-default-browser-check",
  ];
  const res = spawnSync(process.execPath, [LH_BIN, ...args].filter(Boolean), {
    cwd: FRONTEND,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
    env: { ...process.env, CHROME_PATH: process.env.CHROME_PATH || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" },
  });
  if (res.status !== 0) {
    process.stderr.write(res.stderr || res.stdout || "");
    throw new Error(`lighthouse failed (exit ${res.status})`);
  }
}

function median(nums) {
  const s = [...nums].sort((a, b) => a - b);
  const m = Math.floor(s.length / 2);
  return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2;
}

function parseReport(path) {
  const json = JSON.parse(run("node", ["-e", `process.stdout.write(require('fs').readFileSync('${path}','utf8'))`]).stdout);
  const perf = json.categories?.performance?.score ?? 0;
  const aud = json.audits || {};
  const num = (a) => (a?.numericValue ?? null);
  return {
    perf: Math.round((perf || 0) * 100),
    sub: {
      fcp: num(aud["first-contentful-paint"]),
      lcp: num(aud["largest-contentful-paint"]),
      tbt: num(aud["total-blocking-time"]),
      cls: num(aud["cumulative-layout-shift"]),
      si: num(aud["speed-index"]),
    },
  };
}

function startPreview() {
  const proc = spawn("pnpm", ["exec", "vite", "preview", "--port", PORT, "--strictPort"], {
    cwd: FRONTEND,
    stdio: ["ignore", "pipe", "pipe"],
    detached: true,
  });
  proc.unref();
  return proc;
}

function killProc(proc) {
  if (!proc) return;
  try {
    process.kill(-proc.pid, "SIGTERM"); // process group
  } catch {
    try { proc.kill("SIGTERM"); } catch {}
  }
}

async function main() {
  if (!LH_BIN) throw new Error("lighthouse binary not found; run `pnpm install` in frontend/");
  const tmp = mkdtempSync(join(tmpdir(), "lh-bench-"));
  let preview;
  try {
    process.stderr.write("[bench] building...\n");
    build();
    process.stderr.write("[bench] starting vite preview...\n");
    preview = startPreview();
    await waitForServer(URL, 30000);
    process.stderr.write(`[bench] running lighthouse x${RUNS}...\n`);
    const perfs = [];
    let medianSub = {};
    const subs = [];
    for (let i = 0; i < RUNS; i++) {
      const out = join(tmp, `run-${i}.json`);
      runLighthouse(out);
      const { perf, sub } = parseReport(out);
      process.stderr.write(`[bench] run ${i + 1}: perf=${perf}\n`);
      perfs.push(perf);
      subs.push(sub);
    }
    const med = median(perfs);
    // sub-scores from the run closest to the median
    let best = subs[0],
      bestDist = Math.abs(perfs[0] - med);
    for (let i = 1; i < subs.length; i++) {
      const d = Math.abs(perfs[i] - med);
      if (d < bestDist) { bestDist = d; best = subs[i]; }
    }
    const result = { primary: med, sub_scores: best };
    process.stdout.write(JSON.stringify(result) + "\n");
    process.stderr.write(`[bench] DONE: ${JSON.stringify(result)}\n`);
  } finally {
    killProc(preview);
    rmSync(tmp, { recursive: true, force: true });
  }
}

main().catch((e) => {
  process.stderr.write(`[bench] ERROR: ${e?.stack || e}\n`);
  // Failure output: score 0 so the loop treats this as a failed candidate.
  process.stdout.write(JSON.stringify({ primary: 0, sub_scores: {}, error: String(e?.message || e) }) + "\n");
  process.exit(0);
});
