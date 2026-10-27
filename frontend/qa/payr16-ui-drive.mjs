// PAYR-16 UI drive-through (real login page, ALTCHA enforced).
// 1) Cross the threshold via API (3 wrong-password logins).
// 2) UI: username blur -> ALTCHA widget appears (AC-04), auto-solves (AC-02 UI).
// 3) UI: correct password + payload -> login succeeds (AC-02).
// 4) Server proof: counter reset (required=false). AC-03 replay via API.
import { chromium } from "@playwright/test";
import fs from "node:fs";
import { altchaSolve } from "./solve-challenge.mjs";

const BASE = "http://localhost:3000";
const API = "http://localhost:8091";
const USER = "frankng";
const PASS = "Admin123";
fs.mkdirSync("qa", { recursive: true });
const log = (m) => console.log(`[${new Date().toISOString()}] ${m}`);
const errors = [];

const login = async (password) => {
  const r = await fetch(`${API}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: USER, password }),
  });
  return { status: r.status, body: await r.json().catch(() => ({})) };
};

// 1) Ensure the threshold is crossed through the real login endpoint. A wrong
// password yields 401; once the counter is at/over the threshold, the captcha
// gate yields 400 with the captcha error (and still increments the counter —
// PAYR-16 AC-03 side effect).
for (let i = 1; i <= 3; i++) {
  const { status, body } = await login("WrongPassword");
  log(`wrong login ${i}: HTTP ${status} "${(body.message ?? "").slice(0, 60)}"`);
  if (status !== 401 && status !== 400) throw new Error(`expected 401/400, got ${status}`);
}
const reqBefore = await (await fetch(`${API}/api/v1/auth/captcha/required?username=${USER}`)).json();
if (reqBefore.data?.required !== true) throw new Error("threshold not reached server-side");
log("server now requires captcha for frankng");

// 2) UI: the real form.
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 390, height: 844 } }); // mobile-first like the workers
page.on("pageerror", (e) => errors.push(`pageerror: ${e.message}`));
page.on("console", (m) => { if (m.type() === "error") errors.push(`console: ${m.text()}`); });

await page.goto(`${BASE}/login`);
await page.locator("#emailOrUsername").fill(USER);
await page.locator("#emailOrUsername").blur();
const widget = page.locator("altcha-widget");
await widget.waitFor({ state: "visible", timeout: 20000 });
log("AC-04: altcha-widget visible; #captchaCode inputs = " + (await page.locator("#captchaCode").count()));

// 3) The widget solves the proof-of-work by itself (auto="onload").
await page.locator('altcha-widget .altcha[data-state="verified"]').waitFor({ timeout: 30000 });
const widgetText = ((await widget.evaluate((el) => el.textContent ?? "")) ?? "").replace(/\s+/g, " ");
const checkboxChecked = await widget.evaluate((el) => el.querySelector("input[type=checkbox]")?.checked === true);
if (!widgetText.includes("Đã xác minh") || !checkboxChecked) {
  throw new Error(`widget state: text="${widgetText.slice(0, 100)}" checkbox=${checkboxChecked}`);
}
log(`AC-02 (UI): self-verified, checkbox ticked — "${widgetText.slice(0, 90)}"`);
await page.screenshot({ path: "qa/payr16-widget-verified-mobile-390.png" });

// 4) Correct password + auto payload -> login succeeds.
await page.locator("#password").fill(PASS);
await page.locator('button[type="submit"]').click();
await page.waitForURL(/\/(admin|partner|accountant|employee)/, { timeout: 30000 });
log(`AC-02: login succeeded after captcha -> ${page.url()}`);
await page.screenshot({ path: "qa/payr16-after-login.png" });

// 5) Side effect: counter reset by the successful login.
const reqAfter = await (await fetch(`${API}/api/v1/auth/captcha/required?username=${USER}`)).json();
log(`side effect: captcha/required after success = ${JSON.stringify(reqAfter.data)} (expect false)`);
if (reqAfter.data?.required !== false) throw new Error("counter not reset after successful login");

await browser.close();

// 6) AC-03 at API level (gate stays active): solve once, consume it with a
// WRONG password (captcha passes -> 401 invalid credentials, payload burned),
// then replay the SAME payload -> rejected as captcha error.
const solveOnce = async () => {
  const ch = await (await fetch(`${API}/api/v1/auth/captcha`)).json();
  return altchaSolve(ch.data.challenge);
};
const rawLogin = async (password, captcha) => {
  const body = { username: USER, password };
  if (captcha) body.captcha = captcha;
  const r = await fetch(`${API}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const b = await r.json().catch(() => ({}));
  return { status: r.status, message: b.message ?? "" };
};
for (let i = 0; i < 3; i++) await rawLogin("WrongPassword"); // re-arm the gate
const payload = await solveOnce();
const burn = await rawLogin("WrongPassword", payload);
log(`AC-03 burn use: HTTP ${burn.status} "${burn.message.slice(0, 50)}"`);
if (burn.status !== 401 || /captcha/i.test(burn.message)) {
  throw new Error(`expected the solved payload to pass the gate and fail on password (401), got ${burn.status} "${burn.message}"`);
}
const replay = await rawLogin("WrongPassword", payload);
log(`AC-03 replay: HTTP ${replay.status} "${replay.message.slice(0, 50)}"`);
if (!/captcha/i.test(replay.message)) throw new Error("replayed payload was not rejected as captcha error");
log(`console/page errors during run: ${errors.length ? errors.join(" | ") : "none"}`);
log("DONE");
