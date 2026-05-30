import React from 'react';
import { AuthProvider, useAuth } from './AuthContext';
import { UserPreferencesProvider } from './UserPreferencesContext';
import { CommandPaletteProvider } from './CommandPaletteContext';
import { AppStateProvider } from './AppStateContext';
import { MetadataProvider } from './MetadataContext';
import { NotificationProvider } from '@/components/notifications/NotificationProvider';
import { BottomNavProvider } from './BottomNavContext';

interface AppProvidersProps {
  children: React.ReactNode;
}

/**
 * Conditionally wraps children with MetadataProvider only for admin role
 * Partners and employees don't have permission to access metadata endpoints
 */
const ConditionalMetadataProvider = ({ children }: { children: React.ReactNode }) => {
  const { user } = useAuth();

  // Only load metadata for admin role
  if (user?.role === 'admin') {
    return <MetadataProvider>{children}</MetadataProvider>;
  }

  return <>{children}</>;
};

/**
 * Combined provider that wraps the entire app with all necessary contexts
 * Order is important: Auth -> UserPreferences -> CommandPalette -> AppState -> Metadata (conditional) -> Notifications
 */
export const AppProviders = ({ children }: AppProvidersProps) => {
  return (
    <AuthProvider>
      <UserPreferencesProvider>
        <CommandPaletteProvider>
          <AppStateProvider>
            <ConditionalMetadataProvider>
              <NotificationProvider>
                <BottomNavProvider>
                  {children}
                </BottomNavProvider>
              </NotificationProvider>
            </ConditionalMetadataProvider>
          </AppStateProvider>
        </CommandPaletteProvider>
      </UserPreferencesProvider>
    </AuthProvider>
  );
};

// Re-export all context hooks for convenience
export { useAuth } from './AuthContext';
export { useUserPreferences } from './UserPreferencesContext';
export { useCommandPalette } from './CommandPaletteContext';
export { useAppState } from './AppStateContext';
export { useMetadata } from './MetadataContext';

export { useBottomNav } from './BottomNavContext';
export type { UserAction, UserPreferences, ActionFrequency } from '@/lib/storage/types';
