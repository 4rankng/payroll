import React, { createContext, useContext, useCallback, useEffect } from 'react';
import { useSearchParams, useLocation } from 'react-router-dom';
import { NuqsAdapter } from 'nuqs/adapters/react-router';
import { validateModalPermissionSync } from '@/lib/modal-permissions';
import { getSecurityContext, type SecurityContext } from '@/lib/modal-security';
import { toast } from '@/components/ui/sonner';

/**
 * Secure Modal Provider
 * Provides centralized modal management with security and permissions
 */

export interface ModalContextType {
  isAuthenticated: boolean;
  userRole: string | null;
  canAccessModal: (modalId: string) => boolean;
  getModalComponent: (modalId: string) => React.ComponentType<Record<string, unknown>> | null;
}

const ModalContext = createContext<ModalContextType | null>(null);

export const useModalContext = () => {
  const context = useContext(ModalContext);
  if (!context) {
    throw new Error('useModalContext must be used within SecureModalProvider');
  }
  return context;
};

interface SecureModalProviderProps {
  children: React.ReactNode;
}

export function SecureModalProvider({ children }: SecureModalProviderProps) {
  const [searchParams] = useSearchParams();
  const location = useLocation();
  
  // Get current modal from URL
  const currentModal = searchParams.get('modal');

  // Security context
  const [securityContext, setSecurityContext] = React.useState<SecurityContext | null>(null);
  
  useEffect(() => {
    try {
      const context = getSecurityContext();
      setSecurityContext(context);
    } catch (error) {
      setSecurityContext(null);
    }
  }, [location]);

  // Check authentication
  const isAuthenticated = Boolean(securityContext);
  const userRole = securityContext?.userRole || null;

  // Check if user can access specific modal
  const canAccessModal = useCallback((modalId: string): boolean => {
    if (!isAuthenticated) return false;

    try {
      return validateModalPermissionSync(modalId);
    } catch {
      return false;
    }
  }, [isAuthenticated]);

  // Modal component registry - lazy loaded for performance
  const getModalComponent = useCallback((modalId: string): React.ComponentType<Record<string, unknown>> | null => {
    // Return null for now - will be implemented with dynamic imports
    return null;
  }, []);

  // Validate current modal access
  useEffect(() => {
    if (!currentModal || !isAuthenticated) return;

    const hasAccess = canAccessModal(currentModal);

    if (!hasAccess) {
      toast({
        title: "Không có quyền truy cập",
        description: `Bạn không có quyền truy cập modal: ${currentModal}`,
        variant: "destructive"
      });

      // Note: Don't manipulate URL directly - let nuqs handle it
      // The modal will be closed by the individual modal component
    }
  }, [currentModal, isAuthenticated, canAccessModal, searchParams]);

  const contextValue: ModalContextType = {
    isAuthenticated,
    userRole,
    canAccessModal,
    getModalComponent
  };

  return (
    <NuqsAdapter>
      <ModalContext.Provider value={contextValue}>
        {children}
      </ModalContext.Provider>
    </NuqsAdapter>
  );
}

/**
 * Higher-order component for securing modal components
 */
export function withModalSecurity<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  requiredModalId: string
) {
  return function SecuredModal(props: P) {
    const { canAccessModal } = useModalContext();
    
    if (!canAccessModal(requiredModalId)) {
      return (
        <div className="flex items-center justify-center p-6">
          <div className="text-center">
            <h3 className="typography-title-large text-destructive">
              Không có quyền truy cập
            </h3>
            <p className="text-muted-foreground mt-2">
              Bạn không có quyền truy cập chức năng này.
            </p>
          </div>
        </div>
      );
    }

    return <WrappedComponent {...props} />;
  };
}