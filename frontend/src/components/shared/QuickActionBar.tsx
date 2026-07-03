import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import {
  type LucideIcon,
  ChevronUp,
  ChevronDown,
  HelpCircle,
  UserPlus,
  Download,
  FolderPlus,
  Users,
  Zap,
  Send,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/contexts';

interface QuickAction {
  id: string;
  label: string;
  icon: LucideIcon;
  action: () => void;
  variant?: 'default' | 'secondary' | 'outline';
  roles?: ('admin' | 'partner' | 'employee' | 'adv_partner')[];
}

interface QuickActionBarProps {
  actions: QuickAction[];
  position?: 'bottom-right' | 'bottom-left' | 'bottom-center';
  collapsible?: boolean;
  className?: string;
}

export const QuickActionBar = ({
  actions,
  position = 'bottom-right',
  collapsible = true,
  className
}: QuickActionBarProps) => {
  const { user } = useAuth();
  const [isExpanded, setIsExpanded] = useState(false);
  const [isVisible, setIsVisible] = useState(true);

  // Filter actions based on user role
  const filteredActions = actions.filter(action => {
    if (!action.roles || action.roles.length === 0) return true;
    return user && action.roles.includes(user.role);
  });

  // Hide the bar if no actions are available
  useEffect(() => {
    setIsVisible(filteredActions.length > 0);
  }, [filteredActions.length]);

  // Auto-collapse after 5 seconds of expansion
  useEffect(() => {
    if (!isExpanded || !collapsible) return;

    const timer = setTimeout(() => {
      setIsExpanded(false);
    }, 5000);

    return () => clearTimeout(timer);
  }, [isExpanded, collapsible]);

  if (!isVisible || !user) return null;

  const positionClasses = {
    'bottom-right': 'bottom-[calc(5rem+env(safe-area-inset-bottom))] right-4 sm:bottom-4',
    'bottom-left': 'bottom-[calc(5rem+env(safe-area-inset-bottom))] left-4 sm:bottom-4',
    'bottom-center': 'bottom-[calc(5rem+env(safe-area-inset-bottom))] left-1/2 transform -translate-x-1/2 sm:bottom-4'
  };

  const maxVisibleActions = 3;
  const visibleActions = isExpanded ? filteredActions : filteredActions.slice(0, maxVisibleActions);
  const hasMoreActions = filteredActions.length > maxVisibleActions;

  return (
    <TooltipProvider>
      <Card
        className={cn(
          'fixed z-50 p-2 shadow-sm border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60',
          positionClasses[position],
          'transition-all duration-300 ease-in-out',
          isExpanded ? 'scale-100' : 'scale-95',
          className
        )}
      >
        <div className="flex flex-col gap-2">
          {/* Action buttons */}
          <div className={cn(
            'flex gap-2',
            position === 'bottom-center' ? 'flex-row' : 'flex-col'
          )}>
            {visibleActions.map((action) => (
              <Tooltip key={action.id}>
                <TooltipTrigger asChild>
                  <Button
                    variant={action.variant || 'default'}
                    size="sm"
                    onClick={action.action}
                    className={cn(
                      'min-h-11 justify-start gap-2',
                      position === 'bottom-center' ? 'flex-col min-w-[80px]' : 'min-w-[140px]'
                    )}
                  >
                    <action.icon className="w-4 h-4" />
                    <span className={cn(
                      'text-xs',
                      position === 'bottom-center' ? 'block' : 'inline'
                    )}>
                      {action.label}
                    </span>
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  <div className="text-center">
                    <p className="font-medium">{action.label}</p>
                  </div>
                </TooltipContent>
              </Tooltip>
            ))}
          </div>

        {/* Expand/Collapse button */}
        {collapsible && hasMoreActions && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsExpanded(!isExpanded)}
            className="min-h-11 px-2 text-xs text-muted-foreground hover:text-foreground"
          >
            {isExpanded ? (
              <>
                <ChevronDown className="w-3 h-3 mr-1" />
                Ẩn
              </>
            ) : (
              <>
                <ChevronUp className="w-3 h-3 mr-1" />
                Thêm ({filteredActions.length - maxVisibleActions})
              </>
            )}
          </Button>
        )}

        </div>
      </Card>
    </TooltipProvider>
  );
};

// Hook to create quick actions for specific pages
export function useQuickActions(pageType: 'employees' | 'projects' | 'timesheet' | 'dashboard') {
  const baseActions: QuickAction[] = [
    {
      id: 'help',
      label: 'Trợ giúp',
      icon: HelpCircle,
      action: () => window.open('/help', '_blank'),
      variant: 'outline'
    }
  ];

  switch (pageType) {
    case 'employees':
      return [
        {
          id: 'add-employee',
          label: 'Thêm NV',
          icon: UserPlus,
          action: () => () => {},
          roles: ['admin', 'partner'] as const
        },
        {
          id: 'export-employees',
          label: 'Xuất Excel',
          icon: Download,
          action: () => () => {},
          variant: 'secondary' as const,
          roles: ['admin', 'partner'] as const
        },
        ...baseActions
      ];

    case 'projects':
      return [
        {
          id: 'create-project',
          label: 'Tạo DA',
          icon: FolderPlus,
          action: () => () => {},
          roles: ['admin', 'partner'] as const
        },
        {
          id: 'assign-employees',
          label: 'Phân công',
          icon: Users,
          action: () => () => {},
          variant: 'secondary' as const,
          roles: ['admin', 'partner'] as const
        },
        ...baseActions
      ];

    case 'timesheet':
      return [
        {
          id: 'quick-entry',
          label: 'Nhập nhanh',
          icon: Zap,
          action: () => () => {}
        },
        {
          id: 'submit-timesheet',
          label: 'Nộp BC',
          icon: Send,
          action: () => () => {},
          variant: 'secondary' as const,
          roles: ['partner'] as const
        },
        ...baseActions
      ];

    default:
      return baseActions;
  }
}
