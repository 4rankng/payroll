import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/utils';
import { MODAL_IDS, getModalMetadata, type ModalId } from '@/constants/modalRegistry';
import { 
  buildSearchParams,
  validateModalParams,
  type ModalSchemaKey, 
  type ModalParams,
  type GenericModalParams 
} from '@/schemas/modalSchemas';
import { authManager } from '@/lib/auth';

/**
 * Type-Safe Modal Link Component
 * Generates deep links to modals with parameter validation
 */

interface BaseModalLinkProps {
  modalId: ModalId;
  children: React.ReactNode;
  className?: string;
  preserveSearch?: boolean;
  replace?: boolean;
  onClick?: (event: React.MouseEvent) => void;
}

// Typed modal link with parameter validation
interface TypedModalLinkProps<T extends ModalSchemaKey> extends BaseModalLinkProps {
  params?: ModalParams<T>;
}

// Simple modal link without parameter validation
interface SimpleModalLinkProps extends BaseModalLinkProps {
  params?: GenericModalParams;
}

// Union type for all modal link props
type ModalLinkProps<T extends ModalSchemaKey> = TypedModalLinkProps<T> | SimpleModalLinkProps;

function ModalLinkComponent<T extends ModalSchemaKey>({
  modalId,
  params,
  children,
  className,
  preserveSearch = false,
  replace = false,
  onClick,
  ...props
}: ModalLinkProps<T>) {
  // Generate the modal URL
  const { href, isAccessible, error } = useMemo(() => {
    // Get modal metadata
    const metadata = getModalMetadata(modalId);
    if (!metadata) {
      return {
        href: '#',
        isAccessible: false,
        error: 'Modal not found'
      };
    }

    // Check permissions
    if (metadata.requiresAuth) {
      const userRole = authManager.getUserRole();
      
      if (!userRole) {
        return {
          href: '#',
          isAccessible: false,
          error: 'Authentication required'
        };
      }

      if (!metadata.roles.includes(userRole)) {
        return {
          href: '#',
          isAccessible: false,
          error: 'Insufficient permissions'
        };
      }
    }

    // Build URL
    let searchParams: URLSearchParams;
    
    try {
      if (metadata.hasParams && metadata.paramSchema && params) {
        // Validate parameters
        const validation = validateModalParams(metadata.paramSchema as T, params);
        
        if (!validation.success) {
          return {
            href: '#',
            isAccessible: false,
            error: 'Invalid parameters'
          };
        }

        searchParams = buildSearchParams(modalId, metadata.paramSchema as T, validation.data);
      } else {
        // Simple modal without parameters
        searchParams = new URLSearchParams();
        searchParams.set('modal', modalId);
        
        // Add unvalidated parameters if provided
        if (params) {
          Object.entries(params).forEach(([key, value]) => {
            if (value !== undefined && value !== null && value !== '') {
              searchParams.set(key, String(value));
            }
          });
        }
      }

      // Preserve existing search params if requested
      if (preserveSearch) {
        const currentParams = new URLSearchParams(window.location.search);
        currentParams.forEach((value, key) => {
          if (key !== 'modal' && !searchParams.has(key)) {
            searchParams.set(key, value);
          }
        });
      }

      const href = `${window.location.pathname}?${searchParams.toString()}`;

      return {
        href,
        isAccessible: true,
        error: null
      };
    } catch (error) {
      return {
        href: '#',
        isAccessible: false,
        error: 'URL generation failed'
      };
    }
  }, [modalId, params, preserveSearch]);

  // Handle click events
  const handleClick = (event: React.MouseEvent<HTMLAnchorElement>) => {
    // Call custom onClick handler if provided
    onClick?.(event);

    // Prevent navigation if not accessible
    if (!isAccessible) {
      event.preventDefault();
      console.warn(`ModalLink: Cannot navigate to ${modalId} - ${error}`);
    }
  };

  // Render disabled state
  if (!isAccessible) {
    return (
      <span 
        className={cn(
          'cursor-not-allowed opacity-50',
          className
        )}
        title={`Không thể truy cập: ${error}`}
        {...props}
      >
        {children}
      </span>
    );
  }

  // Render active link
  return (
    <Link
      to={href}
      replace={replace}
      className={className}
      onClick={handleClick}
      {...props}
    >
      {children}
    </Link>
  );
}

// Export the main component
export const ModalLink = ModalLinkComponent;

// Convenience components for common modal types
export function UserModalLink({
  userId,
  tab,
  children,
  className
}: {
  userId: string;
  tab?: 'details' | 'activity' | 'permissions';
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <ModalLink
      modalId={MODAL_IDS.USER_DETAILS}
      params={{ id: userId, tab }}
      className={className}
    >
      {children}
    </ModalLink>
  );
}

export function EmployeeModalLink({
  employeeId,
  tab,
  variant = 'admin',
  children,
  className
}: {
  employeeId: string;
  tab?: 'details' | 'projects' | 'timesheet' | 'payroll';
  variant?: 'admin' | 'partner';
  children: React.ReactNode;
  className?: string;
}) {
  const modalId = MODAL_IDS.EMPLOYEE_DETAILS;
    
  return (
    <ModalLink
      modalId={modalId}
      params={{ id: employeeId, tab }}
      className={className}
    >
      {children}
    </ModalLink>
  );
}

export function ProjectModalLink({
  projectId,
  tab,
  variant = 'admin',
  children,
  className
}: {
  projectId: string;
  tab?: 'details' | 'employees' | 'timesheet' | 'settings';
  variant?: 'admin' | 'partner';
  children: React.ReactNode;
  className?: string;
}) {
  const modalId = MODAL_IDS.PROJECT_DETAILS;
    
  return (
    <ModalLink
      modalId={modalId}
      params={{ id: projectId, tab }}
      className={className}
    >
      {children}
    </ModalLink>
  );
}

export function TimesheetModalLink({
  timesheetId,
  tab,
  children,
  className
}: {
  timesheetId: string;
  tab?: 'details' | 'entries' | 'history';
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <ModalLink
      modalId={MODAL_IDS.TIMESHEET_DETAILS}
      params={{ id: timesheetId, tab }}
      className={className}
    >
      {children}
    </ModalLink>
  );
}

// Button-style modal triggers (opens modal via navigation)
export function ModalButton<T extends ModalSchemaKey>({
  modalId,
  params,
  children,
  className,
  variant = 'default',
  size = 'default',
  disabled = false,
  ...props
}: {
  modalId: ModalId;
  params?: ModalParams<T>;
  children: React.ReactNode;
  className?: string;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link';
  size?: 'default' | 'sm' | 'lg' | 'icon';
  disabled?: boolean;
} & Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onClick'>) {
  const metadata = getModalMetadata(modalId);
  
  // Check if accessible
  const isAccessible = useMemo(() => {
    if (!metadata) return false;
    
    if (metadata.requiresAuth) {
      const userRole = authManager.getUserRole();
      return userRole && metadata.roles.includes(userRole);
    }
    
    return true;
  }, [metadata]);

  const handleClick = () => {
    if (!isAccessible || disabled) return;
    
    // Use the navigation hook to open modal
    // This would need to be implemented with a custom hook or context
    const event = new CustomEvent('openModal', { 
      detail: { modalId, params } 
    });
    window.dispatchEvent(event);
  };

  // Button variant styles
  const variantStyles = {
    default: 'bg-primary text-primary-foreground hover:bg-primary/90',
    destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
    outline: 'border border-input bg-card hover:bg-accent hover:text-accent-foreground',
    secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
    ghost: 'hover:bg-accent hover:text-accent-foreground',
    link: 'text-primary underline-offset-4 hover:underline'
  };

  const sizeStyles = {
    default: 'h-10 px-4 py-2',
    sm: 'h-9 rounded-md px-3',
    lg: 'h-11 rounded-md px-8',
    icon: 'h-10 w-10'
  };

  return (
    <button
      className={cn(
        'inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
        variantStyles[variant],
        sizeStyles[size],
        (!isAccessible || disabled) && 'opacity-50 cursor-not-allowed',
        className
      )}
      onClick={handleClick}
      disabled={disabled || !isAccessible}
      {...props}
    >
      {children}
    </button>
  );
}

// Export all components and types
export type { ModalLinkProps, BaseModalLinkProps, TypedModalLinkProps, SimpleModalLinkProps };
export default ModalLink;