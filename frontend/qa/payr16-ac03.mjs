// AC-03: single-use replay rejection (gate kept active; wrong-password burn).
import { altchaSolve } from "./solve-challenge.mjs";
const API = "http://localhost:8091";
const USER = "frankng";
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
for (let i = 0; i < 3; i++) {
  const r = await rawLogin("WrongPassword");
  console.log(`re-arm ${i + 1}: HTTP ${r.status} "${r.message.slice(0, 45)}"`);
}
const ch = await (await fetch(`${API}/api/v1/auth/captcha`)).json();
const payload = await altchaSolve(ch.data.challenge);
const burn = await rawLogin("WrongPassword", payload);
console.log(`burn use: HTTP ${burn.status} "${burn.message.slice(0, 45)}"`);
if (burn.status !== 401 || /captcha/i.test(burn.message)) throw new Error(`expected 401 invalid-credentials after passing the gate, got ${burn.status} "${burn.message}"`);
const replay = await rawLogin("WrongPassword", payload);
console.log(`replay:   HTTP ${replay.status} "${replay.message.slice(0, 45)}"`);
if (!/captcha/i.test(replay.message)) throw new Error("REPLAY WAS NOT REJECTED");
console.log("AC-03 PASS: replay rejected with captcha error, counter kept climbing");
