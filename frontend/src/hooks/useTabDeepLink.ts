import { useLocation, useNavigate } from "react-router-dom";
import { useCallback, useEffect, useMemo, useState, useRef } from "react";

/**
 * Hook for managing tab deep-linking in modal components
 * Uses URL as single source of truth to prevent race conditions
 */
export interface UseTabDeepLinkOptions {
  validTabs: string[];
  defaultTab: string;
  paramName?: string;
}

export function useTabDeepLink({
  validTabs,
  defaultTab,
  paramName = 'tab'
}: UseTabDeepLinkOptions) {
  const location = useLocation();
  const navigate = useNavigate();

  const search = location.search;
  const params = useMemo(() => new URLSearchParams(search), [search]);

  // Memoize validTabs array to prevent reference changes
  const stableValidTabs = useMemo(() => validTabs, [validTabs]);

  const readTab = useCallback((p: URLSearchParams) => {
    const v = p.get(paramName) || defaultTab;
    return stableValidTabs.includes(v) ? v : defaultTab;
  }, [paramName, defaultTab, stableValidTabs]);

  // Single source of truth comes from URL
  const [activeTab, setActiveTab] = useState(() => readTab(params));

  // Debounce timer for URL updates
  const debounceTimerRef = useRef<NodeJS.Timeout>();

  // Keep local state in sync with URL changes
  useEffect(() => {
    const next = readTab(params);
    setActiveTab(prev => (prev === next ? prev : next));
  }, [params, readTab]);

  // When user clicks a tab, update state immediately and debounce URL sync
  const handleTabChange = useCallback((next: string) => {
    if (!stableValidTabs.includes(next)) return;

    setActiveTab(next); // ✅ immediate update (optimistic)

    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Debounce URL update to prevent excessive re-renders
    debounceTimerRef.current = setTimeout(() => {
      const nextParams = new URLSearchParams(params);
      nextParams.set(paramName, next);

      navigate({
        pathname: location.pathname,
        search: nextParams.toString(),
      }, { replace: true });
    }, 100); // 100ms debounce
  }, [navigate, location.pathname, params, paramName, stableValidTabs]);

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  return {
    activeTab,
    handleTabChange,
    isTabActive: (tab: string) => activeTab === tab
  };
}

/**
 * Helper to generate tab URLs for navigation
 */
export function createTabUrl(currentUrl: string, tabName: string, defaultTab: string, paramName = 'tab'): string {
  const url = new URL(currentUrl);
  const searchParams = new URLSearchParams(url.search);
  
  if (tabName !== defaultTab) {
    searchParams.set(paramName, tabName);
  } else {
    searchParams.delete(paramName);
  }
  
  return `${url.pathname}?${searchParams.toString()}`;
}

/**
 * Common tab configurations for different modal types
 */
export const TAB_CONFIGS = {
  PROJECT_DETAILS: {
    validTabs: ['info', 'details', 'employees', 'payrates'] as string[],
    defaultTab: 'info'
  },
  USER_DETAILS: {
    validTabs: ['details', 'activity'] as string[],
    defaultTab: 'details'
  },
  EMPLOYEE_DETAILS: {
    validTabs: ['details', 'statistics', 'permissions'] as string[],
    defaultTab: 'details'
  },
  TIMESHEET_DETAILS: {
    validTabs: ['overview', 'breakdown', 'history'] as string[],
    defaultTab: 'overview'
  },
  LEDGER_ENTRY_DETAILS: {
    validTabs: ['details', 'metadata'] as string[],
    defaultTab: 'details'
  },
  APPROVAL_DETAILS: {
    validTabs: ['details', 'history', 'documents'] as string[],
    defaultTab: 'details'
  }
};