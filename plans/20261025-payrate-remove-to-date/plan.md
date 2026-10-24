# Plan: Remove effective_to/toDate from payrate — full stack + deploy + prod DDL

Status: DONE (deployed prod 25/10, ticket PAYR-15, commit ad388ead, merge b602b087)
Ticket: PAYR-15 (board PAYR) · Implementation on branch fix-payr-int-2410

## Owner decisions (grilling session 25/10 — 7 câu, MCQ từng câu)

| # | Câu hỏi | Quyết định |
|---|---------|-----------|
| Q1 | "Project effective from/to date" là gì? | Projects table không có effective dates → **không làm** phần project |
| Q2 | Độ sâu removal | **Full stack** (backend + API contract + FE), giữ nguyên fromDate |
| Q3 | Caller còn gửi effective_to | **400 reject** + message VN (hết parse-rồi-ignore) |
| Q4 | Active detection + UX | **Successor-based**: latest fromDate ≤ today; strip derive end từ successor |
| Q5 | Thời điểm | **Land ngay** |
| Q6 | Drop cột ở đâu | **Deploy prod rồi drop cột trên prod** — override gate 01/11 của PAYR-8 (rủi ro đã trình bày, owner chấp nhận) |
| Q7 | Ticket + branch | **Ticket mới PAYR-15**, code trên branch fix-payr-int-2410 |

## Phases (đã thực hiện)

1. Ticket PAYR-15 (Todo → In Progress → Done).
2. Backend: tripwire 400 trên Create/Update/Validate; xóa AttachDisplayEndDates + display ToDate + validate effective_to FieldResult + ProjectCurrentPayrate.ToDate + dead filter. Thử nghiệm mới: "no end date on the wire" + "effective_to → 400".
3. Frontend: types/payloads/dry-run contract; successor-based active (PayrateConfigTab, PayrateEditPage desktop+mobile, period strip, ShiftNamesSection, useTimesheetEntry); 'Đã thay thế'.
4. Gates: go test -race 97 ok · suite 428/0 FAIL · FE lint/type-check/vitest 929/929 · UI DRIVEN 1280+390 (qa/payr15/).
5. Merge main b602b087 → push → make deploy → backup prod → apply 138 prod → verify SQL + idempotency + route smoke.

## Acceptance — kết quả

- grep effective_to/toDate payrate paths: chỉ còn tripwire + comment giải thích ✓
- effective_to → 400 (integration test + handler code) ✓
- API responses không có key toDate (raw-JSON test) ✓
- FE successor-based, không render end date (UI DRIVEN, screenshots) ✓
- Prod: 138 applied, SHOW COLUMNS rỗng, indexes đúng, 80 rows ✓ (chi tiết HANDOFF.md)

## Out of scope

- project_employees start_date/last_date; schedule/check_in_effective_from; from_date concept.
