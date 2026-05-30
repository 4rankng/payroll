import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useCallback, useEffect, useMemo } from 'react';
import { useQueryState, parseAsString } from 'nuqs';
import { 
  encryptModalData, 
  decryptModalData, 
  sanitizeParam, 
  isValidParam,
  getSecurityContext,
  type SecurityContext
} from '@/lib/modal-security';
import { 
  validateModalPermissionSync, 
  throwPermissionError
} from '@/lib/modal-permissions';
import { toast } from '@/components/ui/sonner';

/**
 * Secure Modal Hook
 * Provides type-safe, permission-checked modal state management with URL persistence
 */

export interface ModalParams {
  [key: string]: string | number | boolean | undefined;
}

export interface ModalOptions {
  requiresAuth?: boolean;
  encryptData?: boolean;
  validateParams?: (params: ModalParams) => boolean;
  onUnauthorized?: () => void;
  onError?: (error: Error) => void;
}

export interface SecureModalState<T extends ModalParams = ModalParams> {
  isOpen: boolean;
  params: T;
  securityContext: SecurityContext | null;
  open: (params?: Partial<T>) => void;
  close: () => void;
  updateParams: (newParams: Partial<T>) => void;
  isLoading: boolean;
  error: Error | null;
}

export function useSecureModal<T extends ModalParams = ModalParams>(
  modalId: string,
  options: ModalOptions = {}
): SecureModalState<T> {
  const routeParams = useParams();
  const [searchParams] = useSearchParams();
  
  // Use nuqs for URL state management - don't use default values for proper deeplink handling
  const [modalState, setModalState] = useQueryState('modal', parseAsString);
  
  const [encryptedData, setEncryptedData] = useQueryState('data', parseAsString);

  // Individual parameter handling with nuqs
  const [idParam, setIdParam] = useQueryState('id', parseAsString);
  const [tabParam, setTabParam] = useQueryState('tab', parseAsString);

  const {
    requiresAuth = true,
    encryptData = false,
    validateParams,
    onUnauthorized,
    onError
  } = options;

  // Check if this modal is currently open
  const isOpen = modalState === modalId;

  // Security context
  const securityContext = useMemo(() => {
    if (!requiresAuth || !isOpen) return null;
    
    try {
      return getSecurityContext();
    } catch (error) {
      onError?.(error as Error);
      return null;
    }
  }, [isOpen, requiresAuth, onError]);

  // Parse modal parameters using nuqs state
  const params = useMemo((): T => {
    const defaultParams = {} as T;
    
    if (!isOpen) return defaultParams;

    try {
      // Get params from route (e.g., /employees/:id)
      const routeData = { ...routeParams };
      
      // Get params from nuqs state
      const nuqsData: Record<string, string> = {};
      if (idParam) nuqsData.id = idParam;
      if (tabParam) nuqsData.tab = tabParam;

      // Decrypt encrypted data if present
      let encryptedParams = {};
      if (encryptData && encryptedData) {
        if (!isValidParam(encryptedData, 'encrypted')) {
          throw new Error('Invalid encrypted data format');
        }
        encryptedParams = decryptModalData(encryptedData);
      }

      const combined = { ...routeData, ...nuqsData, ...encryptedParams } as T;

      // Validate parameters if validator provided
      if (validateParams && !validateParams(combined)) {
        throw new Error('Parameter validation failed');
      }

      return combined;
    } catch (error) {
      console.error('Failed to parse modal parameters:', error);
      onError?.(error as Error);
      return defaultParams;
    }
  }, [isOpen, routeParams, idParam, tabParam, encryptedData, encryptData, validateParams, onError]);


  // Open modal with parameters
  const open = useCallback((newParams?: Partial<T>) => {
    try {
      // Validate permission before opening
      if (requiresAuth && !validateModalPermissionSync(modalId)) {
        throwPermissionError(modalId);
      }

      const paramsToSet = { ...params, ...newParams } as T;

      // Build URL with parameters
      if (encryptData && Object.keys(paramsToSet).length > 0) {
        // Encrypt sensitive data
        const encrypted = encryptModalData(paramsToSet);
        setEncryptedData(encrypted);
      } else {
        // Set individual parameters using nuqs
        if (paramsToSet.id) {
          setIdParam(String(paramsToSet.id));
        }
        if (paramsToSet.tab) {
          setTabParam(String(paramsToSet.tab));
        }
      }

      setModalState(modalId);
      
    } catch (error) {
      console.error('Failed to open modal:', error);
      onError?.(error as Error);
      
      if (error instanceof Error && error.message.includes('Access denied')) {
        toast({
          title: "Không có quyền truy cập",
          description: error.message,
          variant: "destructive"
        });
      }
    }
  }, [modalId, params, requiresAuth, encryptData, setModalState, setEncryptedData, setIdParam, setTabParam, onError]);

  // Close modal
  const close = useCallback(() => {
    // Only use nuqs methods - they handle URL updates automatically
    setModalState(null);
    setEncryptedData(null);
    setIdParam(null);
    setTabParam(null);
  }, [setModalState, setEncryptedData, setIdParam, setTabParam]);

  // Note: Auto-open removed to prevent race conditions with nuqs
  // nuqs will handle URL synchronization automatically

  // Permission validation effect
  useEffect(() => {
    if (!isOpen || !requiresAuth) return;

    try {
      const hasPermission = validateModalPermissionSync(modalId, params);
      
      if (!hasPermission) {
        onUnauthorized?.();
        toast({
          title: "Không có quyền truy cập",
          description: "Bạn không có quyền truy cập chức năng này.",
          variant: "destructive"
        });
        
        // Close modal and redirect
        close();
        return;
      }
    } catch (error) {
      onError?.(error as Error);
      close();
    }
  }, [isOpen, modalId, params, requiresAuth, onUnauthorized, onError, close]);

  // Update parameters
  const updateParams = useCallback((newParams: Partial<T>) => {
    if (!isOpen) return;
    
    open({ ...params, ...newParams });
  }, [isOpen, params, open]);

  return {
    isOpen,
    params,
    securityContext,
    open,
    close,
    updateParams,
    isLoading: false, // Can be extended for async operations
    error: null // Can be extended for error handling
  };
}

/**
 * Hook for modal routes (when modal is rendered as a route)
 */
export function useModalRoute<T extends ModalParams = ModalParams>(
  modalId: string,
  options: ModalOptions = {}
) {
  const navigate = useNavigate();
  
  const modal = useSecureModal<T>(modalId, {
    ...options,
    onUnauthorized: () => {
      // Redirect to appropriate page on unauthorized access
      const userRole = options.onUnauthorized || (() => {
        navigate(window.location.pathname.includes('/admin') ? '/admin' : '/partner');
      });
      userRole();
    }
  });

  // Auto-open modal when component mounts (for route-based modals)
  useEffect(() => {
    if (!modal.isOpen) {
      modal.open();
    }
  }, [modal]);

  // Enhanced close that navigates back
  const closeAndNavigateBack = useCallback(() => {
    modal.close();
    navigate(-1); // Go back in history
  }, [modal, navigate]);

  return {
    ...modal,
    close: closeAndNavigateBack
  };
}

/**
 * Utility hook for creating modal links
 */
export function useModalNavigation() {
  const navigate = useNavigate();

  const navigateToModal = useCallback((
    modalId: string, 
    params?: ModalParams,
    options?: { replace?: boolean }
  ) => {
    // Build URL based on modal type
    const searchParams = new URLSearchParams();
    searchParams.set('modal', modalId);
    
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.set(key, String(value));
        }
      });
    }
    
    const url = `${window.location.pathname}?${searchParams.toString()}`;
    
    if (options?.replace) {
      navigate(url, { replace: true });
    } else {
      navigate(url);
    }
  }, [navigate]);

  return { navigateToModal };
}