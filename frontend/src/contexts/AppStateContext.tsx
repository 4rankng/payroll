import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import { useAuth } from './AuthContext';

interface Project {
  id: string;
  name: string;
  status: string;
}

interface FilterState {
  dateRange?: {
    start: Date;
    end: Date;
  };
  status?: string;
  department?: string;
  project?: string;
}

interface AppStateContextValue {
  // Global state
  currentProject: Project | null;
  pendingApprovalsCount: number;
  activeFilters: FilterState;
  isGlobalLoading: boolean;

  // Project context
  setCurrentProject: (project: Project | null) => void;
  clearCurrentProject: () => void;

  // Approvals
  setPendingApprovalsCount: (count: number) => void;

  // Filters
  setActiveFilters: (filters: FilterState) => void;
  clearFilters: () => void;

  // Global loading
  setGlobalLoading: (loading: boolean) => void;

  // Page context
  pageTitle: string;
  pageSubtitle?: string;
  setPageTitle: (title: string, subtitle?: string) => void;
}

const AppStateContext = createContext<AppStateContextValue | undefined>(undefined);

interface AppStateProviderProps {
  children: React.ReactNode;
}

export const AppStateProvider = ({ children }: AppStateProviderProps) => {
  const { user } = useAuth();

  // Global state
  const [currentProject, setCurrentProjectState] = useState<Project | null>(null);
  const [pendingApprovalsCount, setPendingApprovalsCountState] = useState(0);
  const [activeFilters, setActiveFiltersState] = useState<FilterState>({});
  const [isGlobalLoading, setIsGlobalLoading] = useState(false);
  const [pageTitle, setPageTitleState] = useState('');
  const [pageSubtitle, setPageSubtitleState] = useState<string | undefined>();

  // Project context actions
  const setCurrentProject = useCallback((project: Project | null) => {
    setCurrentProjectState(project);
  }, []);

  const clearCurrentProject = useCallback(() => {
    setCurrentProjectState(null);
  }, []);

  // Approvals actions
  const setPendingApprovalsCount = useCallback((count: number) => {
    setPendingApprovalsCountState(count);
  }, []);

  // Filter actions
  const setActiveFilters = useCallback((filters: FilterState) => {
    setActiveFiltersState(prev => ({ ...prev, ...filters }));
  }, []);

  const clearFilters = useCallback(() => {
    setActiveFiltersState({});
  }, []);

  // Loading actions
  const setGlobalLoading = useCallback((loading: boolean) => {
    setIsGlobalLoading(loading);
  }, []);

  // Page context actions
  const setPageTitle = useCallback((title: string, subtitle?: string) => {
    setPageTitleState(title);
    setPageSubtitleState(subtitle);
  }, []);

  const value: AppStateContextValue = useMemo(() => ({
    // State
    currentProject,
    pendingApprovalsCount,
    activeFilters,
    isGlobalLoading,
    pageTitle,
    pageSubtitle,

    // Actions
    setCurrentProject,
    clearCurrentProject,
    setPendingApprovalsCount,
    setActiveFilters,
    clearFilters,
    setGlobalLoading,
    setPageTitle
  }), [
    currentProject,
    pendingApprovalsCount,
    activeFilters,
    isGlobalLoading,
    pageTitle,
    pageSubtitle,
    setCurrentProject,
    clearCurrentProject,
    setPendingApprovalsCount,
    setActiveFilters,
    clearFilters,
    setGlobalLoading,
    setPageTitle
  ]);

  return (
    <AppStateContext.Provider value={value}>
      {children}
    </AppStateContext.Provider>
  );
};

export const useAppState = (): AppStateContextValue => {
  const context = useContext(AppStateContext);
  if (context === undefined) {
    throw new Error('useAppState must be used within an AppStateProvider');
  }
  return context;
};
