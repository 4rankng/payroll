import { altchaSolve } from "./solve-challenge.mjs";
const API = "http://localhost:8091";
const ch = await (await fetch(`${API}/api/v1/auth/captcha`)).json();
const payload = await altchaSolve(ch.data.challenge);
const r = await fetch(`${API}/api/v1/auth/login`, {
  method: "POST", headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ username: "frankng", password: "Admin123", captcha: payload }),
});
const b = await r.json().catch(() => ({}));
console.log("reset login:", r.status, b.message ?? "");
const req = await (await fetch(`${API}/api/v1/auth/captcha/required?username=frankng`)).json();
console.log("required:", JSON.stringify(req.data));
