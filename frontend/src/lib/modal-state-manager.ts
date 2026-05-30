import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import { getModalConfig, validateModalParams } from './modal-registry-auto';
import { authManager } from '@/lib/auth';
import { toast } from '@/components/ui/sonner';

/**
 * Centralized Modal State Management with URL Synchronization
 * Single source of truth for modal state with no race conditions
 */

export interface ModalParams {
  [key: string]: string | number | boolean | undefined;
}

export interface ModalState {
  modalId: string | null;
  params: ModalParams;
  returnTo: string | null;
  isLoading: boolean;
  error: string | null;
}

export interface ModalActions {
  openModal: (modalId: string, params?: ModalParams, options?: OpenModalOptions) => Promise<void>;
  closeModal: (options?: CloseModalOptions) => void;
  updateParams: (params: Partial<ModalParams>) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  syncFromURL: () => Promise<void>;
  syncToURL: () => void;
  clearState: () => void;
}

export interface OpenModalOptions {
  replace?: boolean;
  preserveSearch?: boolean;
  skipReturnTo?: boolean;
  skipURLSync?: boolean;
}

export interface CloseModalOptions {
  replace?: boolean;
  skipURLSync?: boolean;
}

type ModalStore = ModalState & ModalActions;

// URL parameter constants
const URL_PARAM_MODAL = 'modal';
const URL_PARAM_RETURN_TO = 'returnTo';
const MODAL_PARAMS = ['id', 'tab', 'userId', 'projectId', 'employeeId', 'date', 'entryId'];

/**
 * Parse modal state from URL search params
 */
function parseModalStateFromURL(): ModalState {
  const searchParams = new URLSearchParams(window.location.search);
  const modalId = searchParams.get(URL_PARAM_MODAL);
  const returnTo = searchParams.get(URL_PARAM_RETURN_TO);

  if (!modalId) {
    return {
      modalId: null,
      params: {},
      returnTo: null,
      isLoading: false,
      error: null
    };
  }

  // Extract modal parameters
  const params: ModalParams = {};
  MODAL_PARAMS.forEach(param => {
    const value = searchParams.get(param);
    if (value !== null) {
      // Try to parse as number if it's numeric
      if (/^\d+$/.test(value)) {
        params[param] = parseInt(value, 10);
      } else if (value === 'true' || value === 'false') {
        params[param] = value === 'true';
      } else {
        params[param] = value;
      }
    }
  });

  return {
    modalId,
    params,
    returnTo,
    isLoading: false,
    error: null
  };
}

/**
 * Build URL search params from modal state
 */
function buildSearchParams(modalId: string | null, params: ModalParams, returnTo: string | null): URLSearchParams {
  const searchParams = new URLSearchParams();

  // Preserve existing non-modal params
  const currentParams = new URLSearchParams(window.location.search);
  currentParams.forEach((value, key) => {
    if (key !== URL_PARAM_MODAL && key !== URL_PARAM_RETURN_TO && !MODAL_PARAMS.includes(key)) {
      searchParams.set(key, value);
    }
  });

  // Add modal params
  if (modalId) {
    searchParams.set(URL_PARAM_MODAL, modalId);

    // Add modal-specific parameters
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        searchParams.set(key, String(value));
      }
    });

    // Add return chain
    if (returnTo) {
      searchParams.set(URL_PARAM_RETURN_TO, returnTo);
    }
  }

  return searchParams;
}

/**
 * Encode return path for modal chaining
 */
function encodeReturnPath(modalId: string, params: ModalParams, existingReturnTo?: string): string {
  const paramString = Object.entries(params)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => `${key}=${encodeURIComponent(String(value))}`)
    .join('&');

  const currentPath = paramString ? `${modalId}?${paramString}` : modalId;

  return existingReturnTo ? `${currentPath}>${existingReturnTo}` : currentPath;
}

/**
 * Parse return path to get parent modal info
 */
function parseReturnPath(returnTo: string): { modalId: string; params: ModalParams } | null {
  if (!returnTo) return null;

  const parts = returnTo.split('>');
  const parentPath = parts[0];

  if (!parentPath) return null;

  const [modalId, paramString] = parentPath.split('?');
  const params: ModalParams = {};

  if (paramString) {
    const searchParams = new URLSearchParams(paramString);
    searchParams.forEach((value, key) => {
      if (/^\d+$/.test(value)) {
        params[key] = parseInt(value, 10);
      } else if (value === 'true' || value === 'false') {
        params[key] = value === 'true';
      } else {
        params[key] = value;
      }
    });
  }

  return { modalId, params };
}

/**
 * Create the modal store
 */
export const useModalStore = create<ModalStore>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    modalId: null,
    params: {},
    returnTo: null,
    isLoading: false,
    error: null,

    // Actions
    openModal: async (modalId: string, params: ModalParams = {}, options: OpenModalOptions = {}) => {
      try {
        set({ isLoading: true, error: null });

        // Get modal configuration
        const config = await getModalConfig(modalId);
        if (!config) {
          throw new Error(`Modal configuration not found: ${modalId}`);
        }

        // Check authentication
        if (config.requiresAuth) {
          const userRole = authManager.getUserRole();
          if (!userRole) {
            toast({
              title: "Cần đăng nhập",
              description: "Vui lòng đăng nhập để truy cập chức năng này.",
              variant: "destructive"
            });
            set({ isLoading: false });
            return;
          }

          // Check role permissions
          if (!config.permissions.roles.includes(userRole)) {
            toast({
              title: "Không có quyền truy cập",
              description: "Bạn không có quyền truy cập chức năng này.",
              variant: "destructive"
            });
            set({ isLoading: false });
            return;
          }
        }

        // Validate parameters
        if (config.deeplink.enabled) {
          const isValid = await validateModalParams(modalId, params);
          if (!isValid) {
            toast({
              title: "Tham số không hợp lệ",
              description: "Tham số được cung cấp không hợp lệ cho modal này.",
              variant: "destructive"
            });
            set({ isLoading: false });
            return;
          }
        }

        // Handle return chain
        let returnTo: string | null = null;
        if (!options.skipReturnTo) {
          const currentState = get();
          if (currentState.modalId) {
            returnTo = encodeReturnPath(
              currentState.modalId,
              currentState.params,
              currentState.returnTo || undefined
            );
          }
        }

        // Update state
        set({
          modalId,
          params,
          returnTo,
          isLoading: false,
          error: null
        });

        // Sync to URL
        if (!options.skipURLSync) {
          get().syncToURL();
        }

      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';
        set({ error: errorMessage, isLoading: false });
        toast({
          title: "Lỗi hệ thống",
          description: errorMessage,
          variant: "destructive"
        });
      }
    },

    closeModal: (options: CloseModalOptions = {}) => {
      const currentState = get();

      if (currentState.returnTo) {
        // Navigate back to parent modal
        const parentModal = parseReturnPath(currentState.returnTo);
        if (parentModal) {
          const remainingChain = currentState.returnTo.split('>').slice(1).join('>');
          set({
            modalId: parentModal.modalId,
            params: parentModal.params,
            returnTo: remainingChain || null,
            isLoading: false,
            error: null
          });

          if (!options.skipURLSync) {
            get().syncToURL();
          }
          return;
        }
      }

      // No return chain, close all modals
      set({
        modalId: null,
        params: {},
        returnTo: null,
        isLoading: false,
        error: null
      });

      if (!options.skipURLSync) {
        get().syncToURL();
      }
    },

    updateParams: (newParams: Partial<ModalParams>) => {
      const currentState = get();
      set({
        params: { ...currentState.params, ...newParams }
      });
      get().syncToURL();
    },

    setLoading: (loading: boolean) => {
      set({ isLoading: loading });
    },

    setError: (error: string | null) => {
      set({ error });
    },

    syncFromURL: async () => {
      const urlState = parseModalStateFromURL();

      // Validate modal exists if one is specified
      if (urlState.modalId) {
        const config = await getModalConfig(urlState.modalId);
        if (!config) {
          console.warn(`Modal not found in URL: ${urlState.modalId}`);
          set({
            modalId: null,
            params: {},
            returnTo: null,
            isLoading: false,
            error: null
          });
          return;
        }
      }

      set(urlState);
    },

    syncToURL: () => {
      const currentState = get();
      const searchParams = buildSearchParams(
        currentState.modalId,
        currentState.params,
        currentState.returnTo
      );

      const newURL = `${window.location.pathname}${searchParams.toString() ? `?${searchParams.toString()}` : ''}`;

      // Use replaceState to avoid creating history entries for every modal change
      window.history.replaceState(null, '', newURL);
    },

    clearState: () => {
      set({
        modalId: null,
        params: {},
        returnTo: null,
        isLoading: false,
        error: null
      });
    }
  }))
);

/**
 * Initialize modal store from URL on page load
 */
export async function initializeModalStore() {
  await useModalStore.getState().syncFromURL();
}

/**
 * Subscribe to browser navigation events
 */
export function setupModalStoreSync() {
  // Listen for browser back/forward navigation
  window.addEventListener('popstate', () => {
    useModalStore.getState().syncFromURL();
  });

  // Listen for URL changes from other parts of the app
  const originalPushState = window.history.pushState;
  const originalReplaceState = window.history.replaceState;

  window.history.pushState = function(...args) {
    originalPushState.apply(this, args);
    useModalStore.getState().syncFromURL();
  };

  window.history.replaceState = function(...args) {
    originalReplaceState.apply(this, args);
    // Don't sync on replace state to avoid infinite loops
    // as syncToURL uses replaceState
  };
}

/**
 * Get navigation info for current modal
 */
export function useModalNavigationInfo() {
  return useModalStore(state => ({
    currentModal: state.modalId,
    hasParent: !!state.returnTo,
    isLoading: state.isLoading,
    error: state.error,
    params: state.params
  }));
}