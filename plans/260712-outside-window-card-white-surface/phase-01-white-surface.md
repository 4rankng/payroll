# Phase 01: White surface

## Context

The outside-window branch in `frontend/src/components/employees/EmployeeCheckInCard.tsx` presents the next-shift timing message plus the attendance reference. Its outer card should be visually neutral and pure white; amber communicates only the timing state inside the card.

## Files

- Modify: `frontend/src/components/employees/EmployeeCheckInCard.tsx`
- Verify: `frontend/src/components/employees/EmployeeCheckInCard.test.tsx`

## Steps

- [ ] Replace the outer outside-window card background utility with `bg-white`.
- [ ] Keep its neutral border and shadow, plus all amber inner timing elements, unchanged.
- [ ] Run the focused employee check-in component test.
- [ ] Run frontend lint, including its TypeScript check.

## Risks and rollback

The only intended change is visual. No hooks, props, API calls, or attendance actions may change. Revert the background utility to restore the ivory treatment.
