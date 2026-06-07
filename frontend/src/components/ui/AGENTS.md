<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# ui — shadcn/ui Base Components

## Purpose

Base UI primitive components built on shadcn/ui (which wraps Radix UI). These are the atomic building blocks used throughout the application — buttons, inputs, dialogs, tables, dropdowns, etc. Custom extensions beyond the standard shadcn/ui library include domain-specific selectors (bank, employee, project, user), data table, infinite list, mobile table, and typography components.

## Key Files

| File | Description |
|------|-------------|
| `button.tsx` | Button variants (default, destructive, outline, ghost, link) |
| `input.tsx` | Text input with consistent styling |
| `select.tsx` | Dropdown select with Radix UI primitives |
| `dialog.tsx` | Modal dialog component |
| `sheet.tsx` | Slide-over panel component |
| `drawer.tsx` | Drawer component (mobile sheet variant) |
| `table.tsx` | Base table primitives (Table, TableHeader, TableBody, TableRow, TableCell) |
| `data-table.tsx` | Full-featured data table with sorting, filtering, pagination, search |
| `card.tsx` | Card container with header, content, footer sections |
| `badge.tsx` | Status badge with color variants |
| `tabs.tsx` | Tab navigation component |
| `form.tsx` | Form components integrated with react-hook-form |
| `sidebar.tsx` | Collapsible sidebar navigation layout |
| `dropdown-menu.tsx` | Dropdown menu with items, separators, labels |
| `command.tsx` | Command palette component (cmdk) |
| `popover.tsx` | Popover overlay component |
| `tooltip.tsx` | Tooltip hover component |
| `calendar.tsx` | Calendar date picker component |
| `date-range-picker.tsx` | Date range selection with presets |
| `alert-dialog.tsx` | Confirmation dialog with cancel/confirm actions |
| `confirm-dialog.tsx` | Simplified confirm/cancel dialog |
| `searchable-dropdown.tsx` | Dropdown with search/filter functionality |
| `multi-searchable-dropdown.tsx` | Multi-select searchable dropdown |
| `async-searchable-dropdown.tsx` | Async-loaded searchable dropdown |
| `employee-selector.tsx` | Employee picker with search |
| `employee-multi-selector.tsx` | Multi-employee picker |
| `project-selector.tsx` | Project picker with search |
| `project-multi-selector.tsx` | Multi-project picker |
| `user-selector.tsx` | User picker with search |
| `user-multi-selector.tsx` | Multi-user picker |
| `bank-selector.tsx` | Vietnamese bank picker with logo |
| `mobile-table.tsx` | Card-based table layout for mobile |
| `infinite-list.tsx` | Infinite scrolling list component |
| `infinite-scroll-container.tsx` | Infinite scroll wrapper with load-more trigger |
| `modal-link.tsx` | Deep-link component that opens modals by URL slug |
| `typography.tsx` | Typography components (heading levels, body text) with design tokens |
| `loading-spinner.tsx` | Animated loading spinner |
| `loading-states.tsx` | Skeleton and loading state presets |
| `loading.tsx` | Loading overlay and inline spinner |
| `error-state.tsx` | Error state display component |
| `user-avatar.tsx` | User avatar with fallback initials |
| `user-card-row.tsx` | User info row (avatar + name + role) |
| `validated-input.tsx` | Input with inline validation feedback |
| `password-strength-indicator.tsx` | Password strength meter |
| `container.tsx` | Max-width content container |
| `responsive-container.tsx` | Responsive width container |
| `responsive-table.tsx` | Table that adapts to mobile layout |
| `pagination.tsx` | Page navigation component |
| `pagination-controls.tsx` | Pagination with page size selector |
| `filter-chip-bar.tsx` | Horizontal filter chip display |
| `long-text.tsx` | Truncated text with expand/collapse |
| `callout.tsx` | Callout/alert box component |
| `carousel.tsx` | Image/content carousel |
| `accordion.tsx` | Collapsible accordion sections |
| `collapsible.tsx` | Simple collapsible wrapper |
| `resizable.tsx` | Resizable panel layout |
| `context-menu.tsx` | Right-click context menu |
| `hover-card.tsx` | Hover card overlay |
| `navigation-menu.tsx` | Top-level navigation menu |
| `menubar.tsx` | Menu bar component |
| `scroll-area.tsx` | Custom scroll area |
| `separator.tsx` | Horizontal/vertical divider |
| `skeleton.tsx` | Loading skeleton placeholder |
| `slider.tsx` | Range slider component |
| `sonner.tsx` | Toast notification component (sonner) |
| `switch.tsx` | Toggle switch component |
| `checkbox.tsx` | Checkbox component |
| `radio-group.tsx` | Radio button group |
| `textarea.tsx` | Multi-line text input |
| `toggle.tsx` | Toggle button component |
| `toggle-group.tsx` | Toggle button group |
| `label.tsx` | Form label component |
| `progress.tsx` | Progress bar component |
| `aspect-ratio.tsx` | Aspect ratio container |
| `alert.tsx` | Alert box component |
| `breadcrumb.tsx` | Breadcrumb navigation |
| `button-group.tsx` | Button group layout |
| `input-otp.tsx` | OTP input component |
| `animated-hamburger.tsx` | Animated hamburger menu icon |
| `auth-loading.tsx` | Authentication loading state |
| `avatar.tsx` | Avatar image component |
| `auth-loading.tsx` | Auth loading spinner |
| `FileUpload.tsx` | File upload drop zone component |
| `FileList.tsx` | File list display component |
| `UrlInput.tsx` | URL input with validation |
| `typography.md` | Typography design system documentation |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `mobile/` | Mobile-specific UI component variants |

## For AI Agents

### Working In This Directory

- These are **base primitives** — do not add domain-specific logic here.
- Do not modify shadcn/ui component APIs without updating all usages.
- Custom selectors (employee, project, user, bank) are acceptable extensions.
- Use `cn()` from `utils.ts` for conditional class merging.
- `typography.tsx` components enforce the design system — use them instead of raw heading tags.

### Testing Requirements

- Visual testing via E2E tests.
- Run `pnpm type-check` after changes.

### Common Patterns

- **Import pattern**: `import { Button } from '@/components/ui/button'`.
- **Variant pattern**: Components use `cva` (class-variance-authority) for variant styling.
- **Composition**: Build complex UI by composing primitives (e.g., Dialog + Button + Form).

## Dependencies

### Internal
- `../../lib/utils.ts` for `cn()` utility

### External
- Radix UI primitives, class-variance-authority, clsx, tailwind-merge, cmdk, sonner, react-hook-form

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
