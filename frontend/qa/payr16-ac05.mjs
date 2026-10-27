// AC-05: a captcha-required login error reloads the page (stale-bundle recovery).
// Driver: tamper the challenge signature -> widget solves -> server rejects
// with the captcha error -> the page must reload (fresh bundle render).
import { chromium } from "@playwright/test";
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
let tampered = 0;
await page.route("**/api/v1/auth/captcha*", async (route) => {
  if (!route.request().url().includes("/required")) {
    const res = await route.fetch();
    const body = await res.json();
    const sig = body.data.challenge.signature;
    body.data.challenge.signature = (sig[0] === "0" ? "1" : "0") + sig.slice(1); // forge
    tampered++;
    await route.fulfill({ res, json: body });
  } else return route.continue();
});
await page.goto("http://localhost:3000/login");
await page.locator("#emailOrUsername").fill("frankng");
await page.locator("#emailOrUsername").blur();
await page.locator('altcha-widget .altcha[data-state="verified"]').waitFor({ timeout: 30000 });
console.log("widget solved the TAMPERED challenge (forged signature)");
await page.locator("#password").fill("Admin123");
const navPromise = page.waitForEvent("framenavigated", { timeout: 20000 });
await page.locator('button[type="submit"]').click();
await navPromise;
await page.waitForTimeout(2500); // post-reload settle
const after = await page.evaluate(() => ({
  url: location.pathname,
  widget: document.querySelectorAll("altcha-widget").length,
}));
console.log(`reloaded after captcha error: path=${after.url} widgets-on-fresh-page=${after.widget} tampered-requests=${tampered}`);
if (!after.url.includes("login")) throw new Error("expected to land back on /login after reload");
console.log("AC-05 PASS: captcha-required error triggered page reload");
await browser.close();
