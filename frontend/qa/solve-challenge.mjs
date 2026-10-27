// ALTCHA v2 solver matching the Go lib's DeriveKeySHA + CounterModeUint32.
import crypto from "node:crypto";

const sha256 = (buf) => crypto.createHash("sha256").update(buf).digest();

function deriveKeySHA(params, saltBytes, password) {
  let data = saltBytes;
  const iterations = params.cost || 1;
  for (let i = 0; i < iterations; i++) {
    data = i === 0 ? sha256(Buffer.concat([saltBytes, password])) : sha256(data);
  }
  return data.subarray(0, params.keyLength || 32);
}

export async function altchaSolve(challenge) {
  const p = challenge.parameters;
  const salt = Buffer.from(p.salt, "hex");
  const nonce = Buffer.from(p.nonce, "hex");
  for (let n = 0; ; n++) {
    const password = Buffer.concat([nonce, (() => { const b = Buffer.alloc(4); b.writeUInt32BE(n); return b; })()]);
    const key = deriveKeySHA(p, salt, password);
    if (key.toString("hex").startsWith(p.keyPrefix || "00")) {
      const payload = { challenge, solution: { counter: n, derivedKey: key.toString("hex") } };
      return Buffer.from(JSON.stringify(payload)).toString("base64");
    }
  }
}
