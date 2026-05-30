/**
 * TanStack Query Persistence Layer
 *
 * Persists query cache to sessionStorage (survives page refreshes within same tab).
 * NOT localStorage — query data is tab-scoped, avoids stale cross-tab issues.
 *
 * Strategy:
 * - Auth/user data: NOT persisted (always fresh from server)
 * - Static/semi-static data: persisted (employees list, salary config, etc.)
 * - Max age: 24 hours (stale data auto-discarded)
 *
 * Why sessionStorage, not localStorage:
 * - localStorage persists forever across tabs → stale data risk
 * - sessionStorage clears on tab close → clean state for new session
 * - Perfect for "survive refresh" use case without stale data buildup
 */

import { QueryClient } from '@tanstack/react-query';
import {
  persistQueryClient,
} from '@tanstack/react-query-persist-client';
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister';

// Keys that should NOT be persisted (always fetch fresh)
const DO_NOT_PERSIST_KEYS = [
  'auth',        // auth state
  'notifications', // real-time notifications
  'me',          // current user profile
  'audit-logs',  // audit trail
];

/**
 * Check if a query key should be persisted
 */
function shouldPersist(queryKey: readonly unknown[]): boolean {
  if (!Array.isArray(queryKey) || queryKey.length === 0) return false;
  const rootKey = String(queryKey[0]);
  return !DO_NOT_PERSIST_KEYS.includes(rootKey);
}

/**
 * Create sessionStorage persister
 */
function createPersister() {
  return createSyncStoragePersister({
    storage: window.sessionStorage,
    key: 'payroll-query-cache',
    // Serialize/deserialize with error handling
    serialize: (data) => JSON.stringify(data),
    deserialize: (str) => {
      try {
        return JSON.parse(str);
      } catch {
        // Corrupted cache — clear and start fresh
        sessionStorage.removeItem('payroll-query-cache');
        return undefined;
      }
    },
  });
}

/**
 * Setup query persistence on the QueryClient
 * Call once after QueryClient creation
 */
export function setupQueryPersistence(queryClient: QueryClient): void {
  // Only run in browser
  if (typeof window === 'undefined') return;

  const persister = createPersister();

  persistQueryClient({
    queryClient,
    persister,
    maxAge: 24 * 60 * 60 * 1000, // 24 hours
    // Only persist queries that are not in the exclusion list
    filter: (query) => shouldPersist(query.queryKey),
  });
}
