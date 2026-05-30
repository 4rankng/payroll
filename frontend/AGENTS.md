# File Responsibility

- **.tsx files** → Focus on UI/UX design only. No business logic.
- **.ts files** → Handle business logic, data manipulation, API calls, types.

# ✅ DOs

## Code & Architecture

1. Follow Shadcn philosophy → small, composable, accessible-first components.
2. Enforce TypeScript best practices → strict typing, predictable behavior.
3. Keep separation of concerns → UI in .tsx, logic in .ts.
4. Use path aliases for clean imports.
5. Group imports logically → React, UI libs, icons, hooks, local components, types.
6. Extract helpers (title mapping, filtering, etc.) into utility functions.
7. Split large JSX/TSX blocks into smaller render functions or subcomponents.
8. Improve naming consistency (handleXxx, renderXxx).
9. Refer to API spec if needed → avoid guessing API behavior. (/Users/dev/Documents/clients/payroll-backend/docs/api/)

## React & Hooks

9. Always include dependency arrays in useEffect.
10. Memoize functions/objects/elements with useMemo/useCallback to prevent infinite loops.
11. Add cleanup functions in useEffect when updating parent state.
12. Split effects by concern → keep dependencies minimal and explicit.
13. Prevent recursive renders with termination conditions.
14. Use React.memo / selective updates to skip unnecessary renders.
15. Avoid inline arrow functions or objects in JSX (memoize them instead).
16. **Memoize React elements passed to parents via callbacks.**
17. Follow hook rules strictly → only call at top level, not inside loops/conditions.

## UI/UX

18. Use Vietnamese for all user-facing text.
19. Check color contrast (use [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)).
20. Ensure accessibility → ARIA roles, focus states, keyboard navigation.
21. Ensure minimum tap target (44px) for interactive elements.
22. Do optimistic updates

## General

22. Use long-term solutions → readable, maintainable, scalable code.
23. Leverage existing knowledge → research tech company solutions before coding.
24. Solve with minimal code changes → don't introduce unnecessary complexity.

# ❌ DON'Ts

1. **DON'T** auto-run prettier or lint:fix without user control.
2. **DON'T** depend on and update the same state in a single useEffect without functional updates.
3. **DON'T** use fallback values that hide real issues.
4. **DON'T** compromise quality with "quick fixes".
5. **DON'T** prefix components with "Modern", "Improved", "Enhanced", etc. → we only keep the best.
6. **DON'T** maintain multiple versions of the same component.
7. **DON'T** build for legacy support.
8. **DON'T** pass new React elements in useEffect without memoization.
9. **DON'T** depend on non-memoized functions in useEffect.
10. **DON'T** forget cleanup functions when useEffect updates parent state.
11. **DON'T** run frontend.
12. **DON'T** implement nice-to-have features.

# 🌐 Project Philosophy

- **Composition over configuration** → build with primitives, not heavy abstractions.
- **Consistency first** → typography, spacing, theme tokens must align with system.
- **Type-safety at all costs** → strict TypeScript everywhere.
- **Accessibility as a feature** → build accessible-first.
- **Avoid over-abstraction** → keep utilities transparent and minimal.
