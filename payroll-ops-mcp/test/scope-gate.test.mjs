// Unit tests for the payroll-ops-mcp scope gate, kill switch, path
// validation, and secrets-file hygiene. Run: npm test (node --test test/).
import test from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, writeFile, rm, chmod } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

import {
  ensureWriteScope,
  killSwitchActive,
  loadApiKey,
  validatePath,
} from "../server.mjs";

const realFs = {
  stat: (p) => import("node:fs/promises").then((m) => m.stat(p)),
  readFile: (p, enc) => import("node:fs/promises").then((m) => m.readFile(p, enc)),
};

const keyInfo = (scope) => () =>
  Promise.resolve({ key_id: 1, agent_name: "hermes", scope, effective_scope: scope });

// ── scope gate (MCP layer) ─────────────────────────────────────────────────

test("read_only key: GET passes the gate without introspection", async () => {
  let called = false;
  const gate = await ensureWriteScope("GET", () => {
    called = true;
    return Promise.resolve({});
  });
  assert.equal(gate.ok, true);
  assert.equal(called, false, "GET must not need a scope round-trip");
});

test("read_only key: POST is refused BEFORE forwarding", async () => {
  const gate = await ensureWriteScope("POST", keyInfo("read_only"));
  assert.equal(gate.ok, false);
  assert.match(gate.reason, /effective scope is read_only/);
  assert.match(gate.reason, /read_write/);
});

test("read_write key: POST passes the MCP gate", async () => {
  const gate = await ensureWriteScope("POST", keyInfo("read_write"));
  assert.equal(gate.ok, true);
});

test("lapsed write window reads as read_only and blocks mutations", async () => {
  const gate = await ensureWriteScope("DELETE", () =>
    Promise.resolve({ key_id: 1, agent_name: "hermes", scope: "read_write", effective_scope: "read_only" }),
  );
  assert.equal(gate.ok, false);
  assert.match(gate.reason, /effective/);
});

test("introspection failure refuses the mutation (never silently widen)", async () => {
  const gate = await ensureWriteScope("POST", () => Promise.reject(new Error("HTTP 503: down")));
  assert.equal(gate.ok, false);
  assert.match(gate.reason, /cannot verify key scope/);
});

// ── kill switch ────────────────────────────────────────────────────────────

test("kill switch activates on 1/true/yes and deactivates otherwise", () => {
  const orig = process.env.PAYROLL_OPS_MCP_DISABLED;
  try {
    for (const v of ["1", "true", "yes"]) {
      process.env.PAYROLL_OPS_MCP_DISABLED = v;
      assert.equal(killSwitchActive(), true, v);
    }
    for (const v of ["0", "false", "", undefined]) {
      if (v === undefined) delete process.env.PAYROLL_OPS_MCP_DISABLED;
      else process.env.PAYROLL_OPS_MCP_DISABLED = v;
      assert.equal(killSwitchActive(), false, String(v));
    }
  } finally {
    if (orig === undefined) delete process.env.PAYROLL_OPS_MCP_DISABLED;
    else process.env.PAYROLL_OPS_MCP_DISABLED = orig;
  }
});

// ── path validation ────────────────────────────────────────────────────────

test("only absolute /api/v1 paths are forwarded", () => {
  assert.equal(validatePath("/api/v1/employees").ok, true);
  assert.equal(validatePath("/api/v1/approvals").ok, true);
  assert.equal(validatePath("employees").ok, false);
  assert.equal(validatePath("https://evil.example/api/v1").ok, false);
  assert.equal(validatePath("//evil.example/api/v1").ok, false);
  assert.equal(validatePath("/api/v1/employees foo").ok, false);
  assert.equal(validatePath(123).ok, false);
});

// ── secrets-file hygiene ───────────────────────────────────────────────────

test("secrets file must exist, be mode 600, and contain a pk_ops_* key", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "ops-mcp-"));

  // Missing file.
  assert.equal(await loadApiKey(realFs, path.join(dir, "absent")), "");

  // Wrong namespace.
  const wrong = path.join(dir, "wrong");
  await writeFile(wrong, "mcp_chat_key_value\n", { mode: 0o600 });
  assert.equal(await loadApiKey(realFs, wrong), "");

  // Group/other readable — refused even with a valid key inside.
  const loose = path.join(dir, "loose");
  await writeFile(loose, "pk_ops_abcd\n", { mode: 0o644 });
  assert.equal(await loadApiKey(realFs, loose), "");

  // Correct: mode 600 + pk_ops_* payload, trailing newline trimmed.
  const ok = path.join(dir, "good");
  await writeFile(ok, "pk_ops_abcd\n", { mode: 0o600 });
  assert.equal(await loadApiKey(realFs, ok), "pk_ops_abcd");

  // No file configured.
  assert.equal(await loadApiKey(realFs, ""), "");

  await rm(dir, { recursive: true, force: true });
});

test("secrets file with mode 640 is also refused", async () => {
  const dir = await mkdtemp(path.join(tmpdir(), "ops-mcp-"));
  const f = path.join(dir, "group-readable");
  await writeFile(f, "pk_ops_abcd\n", { mode: 0o600 });
  await chmod(f, 0o640);
  assert.equal(await loadApiKey(realFs, f), "");
  await rm(dir, { recursive: true, force: true });
});
