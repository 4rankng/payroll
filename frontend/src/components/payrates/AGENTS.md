<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# payrates — Payrate Editor Components

## Purpose

Payrate configuration components supporting two editing modes: **flexible payrate editor** (for flexible/attendance-based projects) and **matrix payrate editor** (for fixed-rate projects with position/hour-type grids). Includes rate type selection, position management, quick templates, and JSON editing for advanced users.

## Key Files

| File | Description |
|------|-------------|
| `index.ts` | Barrel exports |
| `types.ts` | Payrate type definitions (rate entries, positions, hour types, templates) |
| `PayrateConfigTab.tsx` | Main tab component that switches between flexible and matrix editors |
| `FlexiblePayrateEditor.tsx` | Flexible project payrate editor with dynamic positions and hour types |
| `PayrateMatrixEditor.tsx` | Matrix editor for position x hour-type rate grid |
| `PayrateRateGrid.tsx` | Rate grid table component for the matrix editor |
| `PayrateJsonEditor.tsx` | Raw JSON editor for advanced payrate configuration |
| `CurrencyInput.tsx` | Currency-formatted input component for rate values |
| `DatePicker.tsx` | Date picker for effective date selection |
| `CategoryNameModal.tsx` | Modal for editing hour type category names |
| `RateTypeSelector.tsx` | Rate type selection (hourly, daily, monthly) |
| `PositionInput.tsx` | Position name input with auto-suggest |
| `PositionRateForm.tsx` | Form for adding/editing a position with its rates |
| `QuickTemplateSelector.tsx` | Quick template picker for common payrate configurations |
| `usePayrateValidation.ts` | Validation logic for payrate form data |
| `usePayrateEditor.ts` | Hook managing payrate editor state and operations |
| `useHourTypeManager.ts` | Hook for managing hour type categories |
| `usePositionManager.ts` | Hook for managing positions |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `components/` | Shared payrate sub-components |
| `components/matrix-editor/` | Matrix editor specific components |
| `components/rate-matrix/` | Rate matrix grid components |
| `hooks/` | Payrate-specific hooks (editor, position, hour type) |
| `utils/` | Payrate utility functions |

### utils/

| File | Description |
|------|-------------|
| `rateOperations.ts` | Rate CRUD operations for the editor |
| `statusHelpers.ts` | Payrate status badge and label helpers |
| `suggestions.ts` | Auto-suggest data for position names and hour types |
| `templateGenerator.ts` | Quick template generation for common payrate configs |

## For AI Agents

### Working In This Directory

- Flexible projects use `FlexiblePayrateEditor` — dynamic positions, no fixed grid.
- Matrix projects use `PayrateMatrixEditor` — fixed position x hour-type grid.
- All rate values use `CurrencyInput` which formats VND amounts.
- `types.ts` defines the payrate data structure that maps to the backend `PayRateConfig`.
- Use `QuickTemplateSelector` for common setups rather than building from scratch.

### Testing Requirements

- Run `pnpm type-check` after changes.
- Payrate changes affect project cost calculations — verify with backend integration tests.

### Common Patterns

- **Editor state**: `usePayrateEditor.ts` manages the entire editor state including dirty tracking.
- **Rate operations**: `rateOperations.ts` provides immutable update functions for rate entries.
- **Validation**: `usePayrateValidation.ts` validates before save, shows inline errors.

## Dependencies

### Internal
- `../../hooks/api/usePayRates.ts` for API mutations
- `../../types/payrates.ts` for type definitions
- `../../utils/payrateHelpers.ts` for calculation helpers
- `../ui/` for base components

### External
- react-hook-form, Zod

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
