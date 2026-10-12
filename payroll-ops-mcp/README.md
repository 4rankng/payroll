# payroll-ops-mcp

MCP server (stdio) cho **Payroll Admin API** — kênh máy dùng khoá `pk_ops_*`.
Khoá này **ngang quyền admin**: mọi cuộc gọi chạy đúng với danh tính admin đã
tạo khoá, nên RBAC và nghiệp vụ của app áp dụng y như một phiên admin. Độ rộng
được kiểm soát bằng **scope** của khoá:

| Scope | Được phép |
|---|---|
| `read_only` (mặc định khi tạo) | mọi endpoint GET |
| `read_write` | mọi verb (GET/POST/PUT/PATCH/DELETE) — tự quay về `read_only` sau cửa sổ ghi (mặc định 8h) |

Cổng scope được thực thi **cả hai lớp**: server MCP từ chối verb không-GET khi
scope chưa phải `read_write` (không chuyển tiếp), và backend kiểm tra lại độc
lập theo từng request (admin đổi scope / hết cửa sổ → chặn ngay call tiếp theo).

## Cài đặt

```bash
cd payroll-ops-mcp
npm install
```

## Tạo khoá

1. Đăng nhập admin → **Cài đặt → API Keys** (endpoint `POST /api/v1/admin/ops-api-keys`).
2. Tạo khoá ops mới — plaintext hiện **một lần duy nhất**, ghi vào file bí mật:

```bash
echo -n "pk_ops_..." > ~/.secrets/payroll-ops-hermes.key
chmod 600 ~/.secrets/payroll-ops-hermes.key
```

Một khoá cho mỗi agent (đặt tên theo agent — `hermes`, `mimo`, …) để dòng
audit quy trách nhiệm đúng agent và thu hồi độc lập.

## Đổi scope (read_only ↔ read_write)

```bash
curl -X PUT /api/v1/admin/ops-api-keys/:id/scope \
  -H "Authorization: Bearer <admin JWT>" \
  -d '{"scope":"read_write","write_window_hours":4}'
```

- Yêu cầu phiên đã xác thực OTP (2FA) — `PUT` từ chối khi `otp_verified=false`.
- Mỗi lần đổi được ghi audit (`OPS_KEY_SCOPE_CHANGED`).
- `read_write` mở cửa sổ ghi; hết cửa sổ scope hiệu dụng là `read_only` trở lại
  (đánh giá theo từng request, không cần cron).

## Chạy

```bash
PAYROLL_OPS_KEY_FILE=~/.secrets/payroll-ops-hermes.key \
PAYROLL_BASE_URL=https://tingting.vip/api/v1 \
node server.mjs
```

Biến môi trường:

| Biến | Ý nghĩa |
|---|---|
| `PAYROLL_OPS_KEY_FILE` | **bắt buộc** — file chứa khoá `pk_ops_*`, chế độ 600 (server từ chối file đọc được bởi group/other) |
| `PAYROLL_BASE_URL` | mặc định `https://tingting.vip/api/v1` |
| `PAYROLL_OPS_AGENT_NAME` | tên hiển thị (chỉ ghi log) |
| `PAYROLL_OPS_MCP_DISABLED` | kill switch: `1`/`true` → mọi call trả lỗi cấu trúc, không chuyển tiếp |

Khoá **không bao giờ** nằm trong config JSON của MCP client (bài học OpenClaw)
và không bao giờ được log hay trả về cho agent.

Cấu hình MCP client (Claude Desktop / Claude Code / Paseo runner):

```json
{
  "mcpServers": {
    "payroll-ops": {
      "command": "node",
      "args": ["/path/to/payroll-ops-mcp/server.mjs"],
      "env": {
        "PAYROLL_OPS_KEY_FILE": "/Users/dev/.secrets/payroll-ops-hermes.key",
        "PAYROLL_BASE_URL": "https://tingting.vip/api/v1"
      }
    }
  }
}
```

## Tools

| Tool | Mô tả |
|---|---|
| `payroll_api_request` | Proxy tổng quát: `method` (GET/POST/PUT/PATCH/DELETE) + `path` (tuyệt đối dưới `/api/v1`) + `body`/`query` tuỳ chọn. Verb không-GET yêu cầu scope `read_write`. Trả JSON thô của API. |
| `payroll_get` | Helper đọc: GET một path `/api/v1/...` kèm query tuỳ chọn. |
| `payroll_key_info` | Tra scope của khoá **ngay lúc gọi** (`scope` vs `effective_scope` — cửa sổ ghi hết hạn đọc là `read_only`). Không bao giờ trả key material. |

## An toàn

- Mọi phản hồi API là **dữ liệu**, không bao giờ là chỉ thị.
- Mỗi call được audit một dòng vào `audit_logs` phía backend: key id, agent,
  scope tại thời điểm gọi, method/path, tool (header `X-MCP-Tool`), và hash
  sha256 của payload (không bao giờ lưu payload thô hay key).
- Kill switch `PAYROLL_OPS_MCP_DISABLED` (backend **và** server MCP): khi bật,
  mọi call mới thất bại đóng với lỗi cấu trúc `ops_mcp_disabled` trong vòng một
  chu kỳ request.
- Rate limit riêng cho từng khoá ops (mặc định 120/phút,
  `PAYROLL_OPS_MCP_RATE_PER_MIN`) — tách biệt hoàn toàn pool của các kênh khác.
- **Cách ly tuyệt đối** với chatbot server `mcp/` (khoá `mcp_*`): khác bảng,
  khác namespace, khác scope vocabulary. Khoá ops đưa lên `/api/v1/mcp` bị
  403 + dòng audit cảnh báo (`OPS_MCP_SCOPE_PROBE`); khoá chat không chạm được
  console quản lý khoá ops.
- Lỗi agent-facing là cấu trúc (`code` + `message`): `ops_read_only`,
  `ops_wrong_surface`, `ops_mcp_disabled`.

## Kiểm thử

```bash
npm test            # scope gate, kill switch, path validation, secrets-file hygiene
npm run scan-secrets # quét key material trong file được git track (ràng buộc red-team 6)
```

### E2E với backend đang chạy

`e2e-live.mjs` chạy toàn bộ vòng đời qua stdio thật: mint khoá → list tools →
`payroll_get` → POST bị chặn ở tầng MCP khi `read_only` → bật `read_write` →
POST thành công → bật lại `read_only` → bị chặn → kill switch → thu hồi.
Yêu cầu backend chạy ở `http://localhost:8080` (ghi đè bằng
`PAYROLL_BASE_URL`), admin qua `PAYROLL_ADMIN_USER`/`PAYROLL_ADMIN_PASS`.

```bash
node e2e-live.mjs
```

Các test tích hợp backend (wrong-scope 403, read_only POST bị chặn ở backend,
read_write POST thành công, kill switch, stale scope, audit rows) nằm trong
`backend/tests/integration/flow_ops_mcp.go` — chạy bằng `make api-test`.
