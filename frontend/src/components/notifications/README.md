# Notification System

A comprehensive notification system that provides in-app notifications for users with real-time updates, badge counts, and toast notifications.

## Features

- ✅ **Real-time notifications** - Automatic polling every 5 minutes
- ✅ **Badge counter** - Shows unread notification count
- ✅ **Dropdown interface** - Quick access to recent notifications
- ✅ **Toast notifications** - Real-time alerts for new notifications
- ✅ **Mark as read** - Individual and bulk actions
- ✅ **Type-safe** - Full TypeScript support
- ✅ **Mobile responsive** - Works on all screen sizes
- ✅ **Accessible** - ARIA compliant components

## Components

### `NotificationDropdown`
Main notification component with bell icon and dropdown menu.
- Shows unread count badge
- Displays recent unread notifications
- "Mark all as read" functionality
- Empty state handling

### `NotificationBadge`
Simple badge component for displaying notification counts.
- Shows count up to 99, then "99+"
- Auto-hides when count is 0
- Styled with destructive variant for visibility

### `NotificationItem`
Individual notification display component.
- Type-specific icons and colors
- Unread/read visual states  
- Relative time formatting
- Click to mark as read

### `NotificationProvider`
Global provider for managing notification behavior.
- Handles toast notifications for new items
- Manages polling lifecycle
- Listens for auth events

## Notification Types

The system supports these notification types from the backend:

- `timesheet_reminder` - Partners should check timesheets
- `approval_request` - Admins should approve timesheets  
- `approval_result` - Approval/rejection results
- `payroll_complete` - Payroll processing completed

## API Integration

### Endpoints Used
- `GET /api/v1/notifications` - Paginated notifications
- `GET /api/v1/notifications/unread` - Unread notifications with count
- `PUT /api/v1/notifications/{id}/read` - Mark as read
- `PUT /api/v1/notifications/read-all` - Mark all as read

### React Query Integration
- Automatic caching and background refetching
- Optimistic updates for better UX
- Error handling with toast feedback
- Window focus refetching

## Usage

The notification system is automatically integrated into both Admin and Partner layouts. No additional setup required.

### Manual Usage
```tsx
import { NotificationDropdown } from '@/components/notifications';

// In your component
<NotificationDropdown />
```

### Hooks
```tsx
import { useUnreadNotifications } from '@/hooks/api/useNotifications';

// Get unread notifications with count
const { data } = useUnreadNotifications();
const notifications = data?.notifications || [];
const count = data?.count || 0;
```

## Configuration

The system automatically:
- Polls for new notifications every 5 minutes
- Refreshes when browser tab gains focus
- Shows toast notifications for new items
- Stops polling when user is not authenticated

## Accessibility

- Proper ARIA labels on interactive elements
- Keyboard navigation support
- Screen reader friendly
- High contrast indicators for unread items

## Future Enhancements

- WebSocket support for real-time updates
- Sound notifications
- Notification preferences/settings
- Push notifications (PWA)
- Desktop notifications API
- Notification categories/filters