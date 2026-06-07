<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# settings — Settings Components

## Purpose

Components for the settings page including individual setting cards and the send notification dialog. Settings cards follow a consistent layout pattern with title, description, and action.

## Key Files

| File | Description |
|------|-------------|
| `SettingCard.tsx` | Reusable setting card with title, description, and action slot |
| `send-notification-dialog/SendNotificationDialog.tsx` | Dialog for sending push notifications to users |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `send-notification-dialog/` | Send notification dialog sub-components |

## For AI Agents

### Working In This Directory

- `SettingCard` is the standard layout for all settings items.
- Send notification dialog uses the push notification system (VAPID).

### Testing Requirements

- Run `pnpm type-check` after changes.

### Common Patterns

- **Setting card pattern**: `<SettingCard title="..." description="..." action={<Button />}>`.

## Dependencies

### Internal
- `../../hooks/api/useNotifications.tsx` for notification sending
- `../ui/` for base components

### External
- None beyond React

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
