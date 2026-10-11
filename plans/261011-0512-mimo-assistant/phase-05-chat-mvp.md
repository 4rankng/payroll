---
title: "Phase 5: Chat MVP"
status: todo
---

# Phase 5: Chat MVP (2a)

## Overview

Chat surface for Admin + Partner: text Q&A over scoped data, tool-calling that can only read or file approval requests, graceful dead ends. Model: `mimo-v2.6-flash`, non-streaming (no SSE infra exists — revisit only on latency complaints).

## Requirements

- [ ] Conversations + messages persisted (MySQL, retention per plan.md open question); **ownership enforced at handler/service level on every conversation/message/attachment access** — Casbin is role×path only and `authorization.go` scopes per-record for `/projects/:id`/`/employees/:id` shapes only; sequential IDs + GET-by-id without owner checks = cross-partner chat leakage
- [ ] **Per-turn deadline + latency instrumentation from day one** (per-turn timing recorded via the existing API-metrics path): p95 ≤2s direct / ≤5s tool turns measured in production, not by a 3-sample spike; the SSE-streaming fast-follow triggers on measured p95 breach, not user complaints; tool-calling turns are ≥2 serialized model round trips — a hard turn deadline (model + tools end-to-end) prevents 30s axios-abort ghosts that leave half-executed tool loops
- [ ] **Role scope explicit**: chat available to `admin` + `partner` only — `adv_partner` reuses `AdminLayout` and would otherwise see the FAB; either gate the FAB by JWT role and test the adv_partner exclusion, or explicitly decide to include it
- [ ] Tool layer in Go, RBAC-checked against the **caller** identity at call time: read tools (my timesheets, import status, approval status, scoped wallet/advance info, error-code/FAQ lookup) + `file_approval_request` (phase-2 service). **No mutation tools.** Tool manifest per role.
- [ ] Prompt-injection resistance: conversation/attachment content marked untrusted in prompts; structural whitelist is the actual defense; injection test in CI ("send it now" / "approve it yourself" only files a request or returns fallback)
- [ ] Graceful fallback copy for unhandled intents ("tính năng này chưa hỗ trợ qua chat — thao tác tại …") — tool manifest is the contract, no hallucinated capability
- [ ] **Fast contract:** p95 ≤2s direct answers / ≤5s tool turns; flash only on the interactive path; system prompt + tool manifest byte-stable (MiMo prompt-cache hits); independent tool calls dispatched in parallel; context = last N turns + running summary
- [ ] **Useful contract:** quick-action chips on empty state ("Sao kê tháng này?", "Import lỗi?", "Trạng thái duyệt"); answers ≤4 sentences + deep link to the exact page; per-message 👍/👎 feedback feeding the tool-gap backlog
- [ ] FAB + chat panel: `MessageChatCircle` (`@untitledui/icons`) trigger; Sheet-based panel with message list + auto-resize composer; desktop + mobile × Admin + Partner. **Mobile entry is a decision, not a coordination note**: the bell FAB returns `null` on mobile (`NotificationFAB.tsx:34`) — choose bottom-nav item vs QuickActionBar-stacked FAB and pin it; panel is **lazy-loaded** (`lib/lazy.tsx`) so the layout chunk doesn't grow; visible progress states (typing indicator, per-tool-call status) so the composer is never a dead spinner on 4G
- [ ] All chat copy Vietnamese

## Implementation Steps

1. Backend: `internal/app/services/llmchat/` — conversation service, tool registry/executor (per-role manifest, audit every call), `chat_completions` orchestration via phase-1 `AIClient` (flash).
2. Tables: `chat_conversations`, `chat_messages` (+ `chat_tool_calls` or JSON column for audit trail) via existing migration mechanism.
3. API: `routes_chat.go` — `POST /chat/conversations`, `POST /chat/conversations/:id/messages`, `GET /chat/conversations/:id` (polling); Casbin rows (admin + partner).
4. Frontend: `services/api/chat.service.ts` (class+singleton per `services/api/client.ts`), `hooks/api/useChat.ts` (query keys pattern), `components/chat/AssistantPanel.tsx` (Sheet), `AssistantFAB.tsx` (icon decision above), mobile variant; mount in both layouts.
5. System prompt: role, scope, tool contract, Vietnamese responses, "never promise unlisted actions", data-as-data rule.
6. Tests: `backend/tests/integration/flow_chat.go` (message→tool call→request filed; RBAC scoping; injection case), frontend vitest for panel/composer/fallback states + parity tests.

## Related Code Files

- Create: `internal/app/services/llmchat/`, `internal/transport/http/handlers/chat/`, `internal/app/bootstrap/routes_chat.go`, `frontend/src/components/chat/`, `frontend/src/services/api/chat.service.ts`, `frontend/src/hooks/api/useChat.ts`, `backend/tests/integration/flow_chat.go`
- Modify: `configs/casbin_policy.csv`, `App.tsx` (none — panel mounts in layouts), `layouts/AdminLayout.tsx`, `layouts/PartnerLayout.tsx`, `components/MobileBottomNav.tsx` (if mobile entry is nav-based)

## Success Criteria

- [ ] Partner asks → gets grounded answer or filed request; never an executed admin action
- [ ] Injection tests green
- [ ] Fallback copy verified for an unlisted request
- [ ] Desktop + mobile × Admin + Partner verified (1280/390/320), 44px targets
- [ ] Target (first month): ≥50% of partner inquiries handled without admin UI intervention; median turns-to-resolution ≤ 2
- [ ] Latency budget met in spike: p95 ≤2s direct / ≤5s tool turns; if missed → SSE streaming fast-follow is mandatory (decision gate recorded)

## Risks

- Latency UX: non-streaming turns on flash should be sub-second-to-few-seconds; if the spike showed pro-level thinking needed for tool choice, cap `max_completion_tokens` and disable thinking for trivial turns.
- Conversation PII: chat history contains payroll-adjacent data — external API accepted by business; still send only the scoped slice needed for the turn.
