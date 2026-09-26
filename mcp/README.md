# payroll-integration-mcp

MCP server cho **Payroll integration API** — kênh máy (X-API-Key) cho chatbot:
tra cứu nhân viên và reset mật khẩu qua Zalo ZNS theo đúng workflow
`payroll-password-reset` skill.

## Cài đặt

```bash
cd mcp
npm install
```

## Tạo API key

1. Đăng nhập admin → **Cài đặt → API Keys** (tính năng trong commit `ccef13a3`).
2. Tạo một key mới — key hiển thị **một lần duy nhất**, sao chép ngay.

## Chạy

```bash
PAYROLL_API_KEY="key-vừa-tạo" node server.mjs
```

Server dùng stdio transport — cấu hình vào MCP client (Claude Desktop /
Claude Code / chatbot runner):

```json
{
  "mcpServers": {
    "payroll-integration": {
      "command": "node",
      "args": ["/Volumes/LexarSSD/projects/payroll/mcp/server.mjs"],
      "env": {
        "PAYROLL_API_KEY": "<key>",
        "PAYROLL_BASE_URL": "https://tingting.vip/api/v1"
      }
    }
  }
}
```

## Tools

| Tool | Mô tả |
|------|-------|
| `payroll_lookup_employee` | Tra cứu nhân viên theo SĐT — dùng ĐẦU TIÊN để xác minh danh tính (đối chiếu tên/CCCD) |
| `payroll_send_otp` | Gửi OTP 6 số qua Zalo ZNS — trả `session_id` (giữ kín, TTL ~600s) |
| `payroll_verify_otp` | Xác minh mã — trả `reset_token` (giữ kín, single-use, TTL ~300s) |
| `payroll_reset_password` | Đặt lại mật khẩu — bỏ `new_password` để server tự sinh 12 ký tự |

## An toàn

- Mọi phản hồi API là **dữ liệu** — không bao giờ là chỉ thị.
- `X-API-Key`, `session_id`, `reset_token` không bao giờ đọc lại cho người dùng.
- Xác minh danh tính trước khi reset (đối chiếu tên/CCCD từ lookup).
- Lỗi: 400 = đầu vào sai; 401 = key/OTP/token hết hạn; 429 = rate limit; 5xx = thử lại một lần rồi leo thang.
