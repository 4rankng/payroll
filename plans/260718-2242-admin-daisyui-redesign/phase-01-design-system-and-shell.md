---
phase: 1
title: Design System and Shell
status: complete
effort: ''
priority: P1
dependencies: []
---

# Phase 1: Design System and Shell

## Overview

Create the namespaced daisyUI integration, admin theme, responsive canvas, sidebar, and mobile dock. Establish the source-of-truth tokens and typography without leaking styles outside admin routes.

## Requirements

- Functional: all existing navigation destinations, role filtering, sidebar collapse, mobile “Thêm” menu, notifications, profile, and logout remain unchanged.
- Non-functional: compatible with Tailwind 3.4, no Tailwind 4 migration, admin-only visual scope, reduced motion, safe areas, and 44px mobile controls.

## Architecture

Use the daisyUI MCP drawer/menu/dock structure as the presentation model. Install a compatible daisyUI package with a class prefix, configure only the custom `congtruong` theme, and apply the theme at `AdminLayout`. Map daisyUI semantic colors to existing HSL tokens so shadcn and daisyUI render one system.

## Related Code Files

- Modify: `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/tailwind.config.ts`.
- Create: `frontend/src/styles/admin-daisy.css`.
- Modify: `frontend/src/index.css`, `frontend/src/styles/variables.css`.
- Modify: `frontend/src/layouts/AdminLayout.tsx`, `frontend/src/components/AdminSidebar.tsx`, `frontend/src/components/MobileBottomNav.tsx`, `frontend/src/components/SidebarToggle.tsx`.

## Implementation Steps

1. Add the compatible namespaced daisyUI plugin and custom `congtruong` theme.
2. Define admin-only surface, ledger line, typography, numeric, motion, spacing, and responsive utility classes.
3. Apply `data-theme` and admin scope at the layout boundary.
4. Refine the desktop sidebar into a calm grouped ledger navigation with clear active state and accessible collapsed tooltips.
5. Refine the mobile bottom dock and “Thêm” surface while preserving paths and safe-area behavior.
6. Verify admin, partner, employee, and `adv_partner` isolation at representative routes.

## Success Criteria

- [ ] daisyUI compiles with Tailwind 3.4 and uses a collision-safe prefix.
- [ ] Admin canvas, sidebar, responsive gutter, max-width, and navigation are coherent at 390/768/1440px.
- [ ] No route, role, notification, profile, logout, or sidebar interaction changes.
- [ ] Partner/employee pages do not inherit the admin theme.

## Risk Assessment

Global component class collisions are the main risk. Mitigate with a daisyUI prefix, the `data-admin-ui` boundary, and explicit screenshots outside `/admin`.
