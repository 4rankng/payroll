# Runbook — payroll-ops-mcp (kênh agent quản trị)

Server MCP `payroll-ops-mcp/` cho phép agent AI (Hermes, omp/Paseo, codex…)
gọi thẳng Payroll Admin API bằng khoá máy `pk_ops_*`. Khoá **ngang quyền
admin** — mọi call chạy với danh tính admin đã tạo khoá, RBAC và nghiệp vụ
của app áp dụng y như phiên admin. Độ rộng chỉ do **scope** của khoá quyết
định: `read_only` (mặc định) → GET; `read_write` → mọi verb, tự hồi về
`read_only` sau cửa sổ ghi (mặc định 8h, tuỳ chỉnh được).

Đây là runbook vận hành. Hướng dẫn agent và cấu hình MCP client nằm trong
`payroll-ops-mcp/README.md`.

## Tạo khoá cho một agent

```bash
curl -X POST http://localhost:8080/api/v1/admin/ops-api-keys \
  -H "Authorization: Bearer <admin JWT>" -H "Content-Type: application/json" \
  -d '{"name":"hermes"}'
```

- Plaintext hiện **một lần duy nhất** — ghi vào file mode 600 (vd
  `~/.secrets/payroll-ops-hermes.key`), không bao giờ đưa vào config JSON.
- Mỗi agent một khoá (tên khoá = tên agent) để dòng audit quy trách nhiệm.
- Scope mặc định `read_only`. Muốn mutation: xem "Đổi scope" bên dưới.

## Đổi scope (read_only ↔ read_write)

```bash
curl -X PUT http://localhost:8080/api/v1/admin/ops-api-keys/<id>/scope \
  -H "Authorization: Bearer <admin JWT>" -H "Content-Type: application/json" \
  -d '{"scope":"read_write","write_window_hours":4}'
```

- Chỉ admin, và yêu cầu phiên **OTP-verified** (`otp_verified=true`); khi
  bật 2FA email, admin phải vào qua `/auth/login/verify` trước.
- Mỗi lần đổi ghi audit `OPS_KEY_SCOPE_CHANGED`.
- Hiệu lực **ngay call tiếp theo** — enforcement theo từng request, agent
  đang chạy không giữ quyền cũ (ràng buộc stale-scope).
- `write_window_hours` mặc định 8. Hết cửa sổ, scope hiệu dụng về
  `read_only` (lazy — không cần cron).

## Giám sát

```sql
-- Mỗi call một dòng: key id, agent, scope tại thời điểm gọi, tool, payload hash
SELECT created_at, action, entity_id, LEFT(message,120)
FROM audit_logs WHERE action LIKE 'OPS_%' ORDER BY id DESC LIMIT 50;
```

| Action | Ý nghĩa |
|---|---|
| `OPS_MCP_CALL` | một call đã xác thực (method/path + tool + sha256 payload) |
| `OPS_MCP_SCOPE_PROBE` | **cảnh báo**: khoá ops đưa lên sai kênh (vd `/api/v1/mcp`) — tín hiệu dò tìm |
| `OPS_KEY_CREATED` / `OPS_KEY_REVOKED` / `OPS_KEY_SCOPE_CHANGED` | vòng đời khoá |

Rate limit riêng theo khoá: mặc định 120/phút (`PAYROLL_OPS_MCP_RATE_PER_MIN`),
tách biệt pool của mọi kênh khác.

## Kill switch

```bash
PAYROLL_OPS_MCP_DISABLED=1   # backend: mọi call ops 503 mã ops_mcp_disabled
                             # server MCP: mỗi tool call trả lỗi cấu trúc
```

Đặt trong env của backend (restart để áp dụng) và/hoặc env của server MCP.
Backend fail-closed trong một chu kỳ request; server MCP không chuyển tiếp
gì khi switch bật.

## Thu hồi

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/ops-api-keys/<id> \
  -H "Authorization: Bearer <admin JWT>"
```

Call tiếp theo bằng khoá đó nhận 401. Khoá chatbot (`mcp_*`) và khoá ops
hoàn toàn độc lập — thu hồi bên này không ảnh hưởng bên kia.

## Ràng buộc cách ly (không thỏa hiệp)

- Khoá ops đưa lên `/api/v1/mcp` (mặt integration cho chatbot) → 403
  `ops_wrong_surface` + dòng audit `OPS_MCP_SCOPE_PROBE`.
- Khoá `mcp_*` không chạm được console quản lý khoá ops (403).
- Khoá ops không với được `/api/v1/auth`, quản lý API key, webhooks, vendor.
- Không có khoá chung, không fallback, không tái dùng credential giữa hai
  server (`mcp/` và `payroll-ops-mcp/`).

## Quét rò rỉ key material

```bash
cd payroll-ops-mcp && npm run scan-secrets
```

Quét file được git track theo pattern `pk_ops_*`/`mcp_*` đủ dài là khoá thật.
Chạy trước khi release; gắn vào CI khi repo có pipeline.
