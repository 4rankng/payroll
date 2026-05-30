import React, { createContext, useContext, useEffect, useState, useCallback, useMemo } from 'react';
import { useAuth } from './AuthContext';
import { indexedDBManager } from '@/lib/storage/indexedDB';
import { UserAction, UserPreferences, ActionFrequency } from '@/lib/storage/types';

interface UserPreferencesContextValue {
  // User preferences
  preferences: UserPreferences | null;
  isLoading: boolean;
  
  // Quick actions
  recentActions: UserAction[];
  frequentActions: ActionFrequency[];
  pinnedActions: string[];
  
  // Actions
  addRecentAction: (action: Omit<UserAction, 'id' | 'timestamp' | 'userId'>) => Promise<void>;
  updatePreferences: (updates: Partial<UserPreferences>) => Promise<void>;
  togglePinnedAction: (actionId: string) => Promise<void>;
  addRecentSearch: (search: string) => Promise<void>;
  
  // Getters
  isActionPinned: (actionId: string) => boolean;
  getTableSettings: (tableId: string) => UserPreferences['tableSettings'][string] | null;
  updateTableSettings: (tableId: string, settings: Partial<UserPreferences['tableSettings'][string]>) => Promise<void>;
}

const UserPreferencesContext = createContext<UserPreferencesContextValue | undefined>(undefined);

interface UserPreferencesProviderProps {
  children: React.ReactNode;
}

export const UserPreferencesProvider = ({ children }: UserPreferencesProviderProps) => {
  const { user } = useAuth();
  const [preferences, setPreferences] = useState<UserPreferences | null>(null);
  const [recentActions, setRecentActions] = useState<UserAction[]>([]);
  const [frequentActions, setFrequentActions] = useState<ActionFrequency[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const loadUserData = useCallback(async (userId: string) => {
    try {
      // Load preferences
      const userPrefs = await indexedDBManager.getUserPreferences(userId);
      if (userPrefs) {
        setPreferences(userPrefs);
      } else {
        // Create default preferences
        const defaultPrefs: UserPreferences = {
          userId,
          pinnedActions: [],
          tableSettings: {},
          recentSearches: [],
          language: 'vi',
          sidebarCollapsed: false,
          updatedAt: new Date()
        };
        await indexedDBManager.setUserPreferences(defaultPrefs);
        setPreferences(defaultPrefs);
      }

      // Load recent actions
      const recent = await indexedDBManager.getUserActions(userId, 10);
      setRecentActions(recent);

      // Load frequent actions
      const frequent = await indexedDBManager.getTopActions(userId, 5);
      setFrequentActions(frequent);

      // Cleanup old actions (keep last 30 days)
      await indexedDBManager.clearOldUserActions(userId, 30);
    } catch (error) {
      console.error('Failed to load user data:', error);
    }
  }, []);

  // Initialize IndexedDB and load user data
  useEffect(() => {
    const initialize = async () => {
      // Early return for null user - don't initialize IndexedDB on login page
      if (!user) {
        setPreferences(null);
        setRecentActions([]);
        setFrequentActions([]);
        setIsLoading(false);
        return;
      }

      // Set loading state
      setIsLoading(true);

      try {
        await indexedDBManager.initialize();
        await loadUserData(user.id);
      } catch (error) {
        console.error('Failed to initialize user preferences:', error);
        // Reset states on error
        setPreferences(null);
        setRecentActions([]);
        setFrequentActions([]);
      } finally {
        setIsLoading(false);
      }
    };

    initialize();
  }, [user, loadUserData]);

  const addRecentAction = useCallback(async (actionData: Omit<UserAction, 'id' | 'timestamp' | 'userId'>) => {
    if (!user) return;

    const action: UserAction = {
      ...actionData,
      id: `${user.id}-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
      timestamp: new Date(),
      userId: user.id
    };

    try {
      await indexedDBManager.addUserAction(action);
      
      // Update local state
      setRecentActions(prev => [action, ...prev.slice(0, 9)]);
      
      // Refresh frequent actions
      const frequent = await indexedDBManager.getTopActions(user.id, 5);
      setFrequentActions(frequent);
    } catch (error) {
      console.error('Failed to save recent action:', error);
    }
  }, [user]);

  const updatePreferences = useCallback(async (updates: Partial<UserPreferences> | ((prev: UserPreferences) => Partial<UserPreferences>)) => {
    if (!user) return;

    setPreferences(prevPreferences => {
      if (!prevPreferences) return null;
      
      const resolvedUpdates = typeof updates === 'function' ? updates(prevPreferences) : updates;
      
      const updatedPrefs: UserPreferences = {
        ...prevPreferences,
        ...resolvedUpdates,
        updatedAt: new Date()
      };

      // Async operation outside of state update
      indexedDBManager.setUserPreferences(updatedPrefs).catch(error => {
        console.error('Failed to update preferences:', error);
      });

      return updatedPrefs;
    });
  }, [user]);

  const togglePinnedAction = useCallback(async (actionId: string) => {
    await updatePreferences(prevPrefs => {
      if (!prevPrefs) return {};
      
      const pinnedActions = prevPrefs.pinnedActions.includes(actionId)
        ? prevPrefs.pinnedActions.filter(id => id !== actionId)
        : [...prevPrefs.pinnedActions, actionId];

      return { pinnedActions };
    });
  }, [updatePreferences]);

  const addRecentSearch = useCallback(async (search: string) => {
    if (!search.trim()) return;

    await updatePreferences(prevPrefs => {
      if (!prevPrefs) return {};

      const recentSearches = [
        search,
        ...prevPrefs.recentSearches.filter(s => s !== search)
      ].slice(0, 10);

      return { recentSearches };
    });
  }, [updatePreferences]);

  const isActionPinned = useCallback((actionId: string) => {
    return preferences?.pinnedActions.includes(actionId) || false;
  }, [preferences]);

  const getTableSettings = useCallback((tableId: string) => {
    return preferences?.tableSettings[tableId] || null;
  }, [preferences]);

  const updateTableSettings = useCallback(async (
    tableId: string, 
    settings: Partial<UserPreferences['tableSettings'][string]>
  ) => {
    await updatePreferences(prevPrefs => {
      if (!prevPrefs) return {};

      const tableSettings = {
        ...prevPrefs.tableSettings,
        [tableId]: {
          ...prevPrefs.tableSettings[tableId],
          ...settings
        }
      };

      return { tableSettings };
    });
  }, [updatePreferences]);

  const value: UserPreferencesContextValue = useMemo(() => ({
    preferences,
    isLoading,
    recentActions,
    frequentActions,
    pinnedActions: preferences?.pinnedActions || [],
    addRecentAction,
    updatePreferences,
    togglePinnedAction,
    addRecentSearch,
    isActionPinned,
    getTableSettings,
    updateTableSettings
  }), [
    preferences,
    isLoading,
    recentActions,
    frequentActions,
    addRecentAction,
    updatePreferences,
    togglePinnedAction,
    addRecentSearch,
    isActionPinned,
    getTableSettings,
    updateTableSettings
  ]);

  return (
    <UserPreferencesContext.Provider value={value}>
      {children}
    </UserPreferencesContext.Provider>
  );
};

export const useUserPreferences = (): UserPreferencesContextValue => {
  const context = useContext(UserPreferencesContext);
  if (context === undefined) {
    throw new Error('useUserPreferences must be used within a UserPreferencesProvider');
  }
  return context;
};