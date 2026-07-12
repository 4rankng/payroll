# Phase 1 — Frontend Error Guidance

## Context

- `frontend/src/components/employees/EmployeeCheckInCard.tsx` owns attendance action failures and currently shows timing guidance only for the confirmed no-salary checkout path.
- `backend/internal/app/services/attendance/attendance_service.go` already returns `Giờ vào làm không hợp lệ...` and `Chỉ có thể tan ca từ ... đến ...` messages.
- `frontend/src/components/employees/EmployeeLocationMap.tsx` already renders the employee marker, nearest gate, distance, and route line.
- Backend guidance fields are optional so existing API clients continue using the Vietnamese message fallback.

## Files

- Create: `frontend/src/utils/attendance-error-guidance.ts` and its test.
- Modify: `frontend/src/components/employees/EmployeeCheckInCard.tsx`.
- Modify: backend attendance error transport/domain path and focused backend tests.

## Implementation

1. Attach optional structured timing and nearest-checkpoint guidance to backend attendance errors while preserving the current message text.
2. Classify check-in and checkout timing messages into a display model using backend guidance and message text as a fallback.
3. Render one closable informational dialog for timing failures; retain the existing no-salary confirmation for the supported checkout override.
4. Open the existing map disclosure whenever a GPS accuracy, permission, or geofence error is classified, while retaining normal manual map control.

## Validation

- Unit tests for timing classification and fallback text.
- Existing location-hook and dock tests.
- Frontend lint/type-check and production build.

## Risks and Rollback

- Timing strings are backend-owned. Matching is limited to the existing classifier phrases and uses the original message if no structured range is available.
- Rollback is limited to the new frontend utility and card presentation state.
