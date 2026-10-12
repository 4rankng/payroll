// One-off E2E driver (not part of the committed test suite): spawns
// payroll-ops-mcp/server.mjs over stdio and drives the full loop against the
// live backend — tool listing, scope gate, mutation parity, stale scope.
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { writeFileSync, chmodSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";

const BASE_URL = "http://localhost:8080/api/v1";

async function adminLogin() {
  const res = await fetch(`${BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: process.env.PAYROLL_ADMIN_USER ?? "frankng", password: process.env.PAYROLL_ADMIN_PASS ?? "Admin123" }),
  });
  const body = await res.json();
  return body.data?.access_token ?? body.access_token;
}

const token = await adminLogin();
if (!token) throw new Error("admin login failed");

async function api(method, p, body) {
  const res = await fetch(`${BASE_URL}${p}`, {
    method,
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  const json = await res.json().catch(() => null);
  return { status: res.status, json };
}

// 1. mint a read_only ops key
const created = await api("POST", "/admin/ops-api-keys", { name: `e2e-${Date.now()}` });
const plaintext = created.json?.data?.key;
const keyID = created.json?.data?.id;
if (!plaintext?.startsWith("pk_ops_")) throw new Error(`key mint failed: ${JSON.stringify(created.json)}`);
console.log(`✓ minted ops key id=${keyID} (read_only default: ${created.json.data.scope})`);

// 2. secrets file mode 600
const dir = mkdtempSync(path.join(tmpdir(), "e2e-ops-"));
const keyFile = path.join(dir, "key");
writeFileSync(keyFile, plaintext + "\n", { mode: 0o600 });
chmodSync(keyFile, 0o600);

// 3. spawn the MCP server over stdio
const transport = new StdioClientTransport({
  command: process.execPath,
  args: ["server.mjs"],
  cwd: path.dirname(new URL(import.meta.url).pathname),
  env: { ...process.env, PAYROLL_OPS_KEY_FILE: keyFile, PAYROLL_BASE_URL: BASE_URL },
});
const client = new Client({ name: "e2e-driver", version: "0.0.0" });
await client.connect(transport);
console.log("✓ MCP server connected over stdio");

// 4. tool listing
const { tools } = await client.listTools();
const names = tools.map((t) => t.name).sort();
console.log(`✓ tools listed: ${names.join(", ")}`);
if (!names.includes("payroll_api_request") || !names.includes("payroll_get") || !names.includes("payroll_key_info")) {
  throw new Error("expected proxy tools missing");
}

const call = (name, args) => client.callTool({ name, arguments: args });

// 5. payroll_get works with read_only
let r = await call("payroll_get", { path: "/api/v1/ops/key-info" });
console.log(`✓ payroll_get (read_only): ${r.content[0].text.slice(0, 80)}`);

// 6. read_only: MCP-layer refuses POST before forwarding
r = await call("payroll_api_request", {
  method: "POST",
  path: "/api/v1/approvals",
  body: { type: "general_request", params: { summary: `e2e mcp gate ${Date.now()}` } },
});
if (!r.isError || !/scope gate/i.test(r.content[0].text)) {
  throw new Error(`MCP-layer gate did not refuse: ${JSON.stringify(r.content[0].text)}`);
}
console.log(`✓ read_only POST refused at MCP layer: ${r.content[0].text.slice(0, 110)}`);

// 7. flip to read_write → same call succeeds end-to-end
await api("PUT", `/admin/ops-api-keys/${keyID}/scope`, { scope: "read_write" });
r = await call("payroll_api_request", {
  method: "POST",
  path: "/api/v1/approvals",
  body: { type: "general_request", params: { summary: `e2e mutation ${Date.now()}` } },
});
if (r.isError) throw new Error(`read_write mutation failed: ${r.content[0].text}`);
const filed = JSON.parse(r.content[0].text);
console.log(`✓ read_write POST filed approval id=${filed.id} status=${filed.status}`);

// 8. flip back to read_only → mutation blocked (MCP gate here; the backend's
//    per-request 403 is proven by the Go integration flow)
await api("PUT", `/admin/ops-api-keys/${keyID}/scope`, { scope: "read_only" });
r = await call("payroll_api_request", {
  method: "POST",
  path: "/api/v1/approvals",
  body: { type: "general_request", params: { summary: `e2e stale ${Date.now()}` } },
});
if (!r.isError || !/ops_read_only|READ ONLY|chỉ đọc|effective scope is read_only/.test(r.content[0].text)) {
  throw new Error(`stale-scope mutation not blocked: ${r.content[0].text}`);
}
console.log(`✓ stale-scope POST blocked: ${r.content[0].text.slice(0, 90)}`);

// 9. kill switch flips every call to a structured error
process.env.PAYROLL_OPS_MCP_DISABLED = "1"; // in-process env of the driver is NOT the server's — restart instead
await client.close();
const transport2 = new StdioClientTransport({
  command: process.execPath,
  args: ["server.mjs"],
  cwd: path.dirname(new URL(import.meta.url).pathname),
  env: { ...process.env, PAYROLL_OPS_KEY_FILE: keyFile, PAYROLL_BASE_URL: BASE_URL, PAYROLL_OPS_MCP_DISABLED: "1" },
});
const client2 = new Client({ name: "e2e-driver", version: "0.0.0" });
await client2.connect(transport2);
r = await client2.callTool({ name: "payroll_get", arguments: { path: "/api/v1/ops/key-info" } });
if (!r.isError || !/disabled/i.test(r.content[0].text)) throw new Error("kill switch did not fail closed");
console.log(`✓ kill switch: ${r.content[0].text.slice(0, 90)}`);
await client2.close();

// 10. revoke + cleanup
await api("DELETE", `/admin/ops-api-keys/${keyID}`);
rmSync(dir, { recursive: true, force: true });
console.log("✓ revoked key, removed secrets file — E2E PASS");
