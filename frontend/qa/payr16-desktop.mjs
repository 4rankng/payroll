// Desktop-width (1280) widget evidence for parity (mobile evidence already captured).
import { chromium } from "@playwright/test";
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
await page.goto("http://localhost:3000/login");
await page.locator("#emailOrUsername").fill("frankng");
await page.locator("#emailOrUsername").blur();
await page.locator("altcha-widget").waitFor({ timeout: 15000 });
await page.locator('altcha-widget .altcha[data-state="verified"]').waitFor({ timeout: 30000 });
const text = ((await page.locator("altcha-widget").evaluate((el) => el.textContent ?? "")) ?? "").replace(/\s+/g, " ");
console.log("desktop widget:", text.slice(0, 80));
await page.screenshot({ path: "qa/payr16-widget-verified-desktop-1280.png" });
await browser.close();
console.log("desktop capture done");
