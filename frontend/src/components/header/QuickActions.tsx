import { useState } from 'react';
import { Briefcase, FileDown, BarChart3, Wallet } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { PayrollReportExportDialog, PayrollReportExportParams } from '@/components/timesheet/PayrollReportExportDialog';
import { AdvancePaymentExportDialog } from '@/components/advance-payment/AdvancePaymentExportDialog';
import { useAuth } from '@/contexts';
import { useNavigate } from 'react-router-dom';
import { useExportPayrollReport } from '@/hooks/api/usePayrolls';
import { cn } from '@/lib/utils';

// Custom icon component for PNG images
interface IconImageProps {
  src: string;
  alt: string;
  className?: string;
}

const IconImage = ({ src, alt, className }: IconImageProps) => (
  <img
    src={src}
    alt={alt}
    className={cn("h-4 w-4", className)}
  />
);

interface FixedAction {
  id: string;
  label: string;
  icon: typeof Briefcase | React.ComponentType<{ className?: string }>;
  href?: string;
  onClick?: () => void;
  permissions: ('admin' | 'partner' | 'employee' | 'adv_partner')[];
}

interface QuickActionsProps {
  className?: string;
  maxActions?: number;
}

export const QuickActions = ({ className }: QuickActionsProps) => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [payrollReportDialogOpen, setPayrollReportDialogOpen] = useState(false);
  const [advancePaymentExportDialogOpen, setAdvancePaymentExportDialogOpen] = useState(false);
  const exportPayrollReportMutation = useExportPayrollReport();

  const handlePayrollReportExport = () => {
    setPayrollReportDialogOpen(true);
  };

  const handlePayrollReportExportSubmit = async (params: PayrollReportExportParams) => {
    try {
      await exportPayrollReportMutation.mutateAsync(params);
      setPayrollReportDialogOpen(false);
    } catch (error) {
      // Error is handled by the mutation's onError callback
    }
  };

  // Fixed list of quick actions - ALL actions available for both roles
  const fixedActions: FixedAction[] = [
    {
      id: 'create-project',
      label: 'Tạo dự án',
      icon: () => <IconImage src="/icons/add-project.png" alt="Add project" />,
      href: '/admin/projects?modal=project_create',
      permissions: ['admin', 'partner'],
    },
    {
      id: 'create-employee',
      label: 'Thêm nhân viên',
      icon: () => <IconImage src="/icons/add-worker.png" alt="Add employee" />,
      href: '/admin/employees?modal=add_employee',
      permissions: ['admin', 'partner'],
    },
    {
      id: 'create-timesheet',
      label: 'Tạo chấm công',
      icon: () => <IconImage src="/icons/add-event.png" alt="Add timesheet" />,
      href: '/admin/timesheet?modal=timesheet_entry',
      permissions: ['admin', 'partner'],
    },
  ];

  const handleActionClick = (action: FixedAction) => {
    // Handle onClick actions first
    if (action.onClick) {
      action.onClick();
      return;
    }

    // Handle navigation actions
    if (action.href) {
      let targetHref = action.href;

      if (user.role === 'partner') {
        // Update URLs for partner role with same modals
        if (action.id === 'create-project') {
          targetHref = '/partner/projects?modal=project_create';
        } else if (action.id === 'create-employee') {
          targetHref = '/partner/employees?modal=add_employee';
        } else if (action.id === 'create-timesheet') {
          targetHref = '/partner/timesheet?modal=timesheet_entry';
        }
      }

      navigate(targetHref);
    }
  };

  if (!user) {
    return null;
  }

  // Filter actions based on user permissions
  const allowedActions = fixedActions.filter(action =>
    action.permissions.includes(user.role)
  );

  return (
    <TooltipProvider>
      <div className={cn("flex items-center gap-1", className)}>
        {allowedActions.map((action) => {
          const IconComponent = action.icon;

          return (
            <Tooltip key={action.id}>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="relative h-9 w-9 p-0 hover:bg-accent hover:text-accent-foreground touch-manipulation"
                  onClick={() => handleActionClick(action)}
                  aria-label={action.label}
                >
                  <IconComponent className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p className="font-medium">{action.label}</p>
              </TooltipContent>
            </Tooltip>
          );
        })}

        {/* Xuất sao kê — dropdown for Bảng công and Ứng lương */}
        {user && (user.role === 'admin' || user.role === 'partner') && (
          <DropdownMenu>
            <Tooltip>
              <TooltipTrigger asChild>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="relative h-9 w-9 p-0 hover:bg-accent hover:text-accent-foreground touch-manipulation"
                    aria-label="Xuất sao kê"
                    disabled={exportPayrollReportMutation.isPending}
                  >
                    <IconImage src="/icons/export-doc.png" alt="Export statement" />
                  </Button>
                </DropdownMenuTrigger>
              </TooltipTrigger>
              <TooltipContent>
                <p className="font-medium">Xuất sao kê</p>
              </TooltipContent>
            </Tooltip>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handlePayrollReportExport}>
                <BarChart3 className="w-4 h-4 mr-2" />
                Bảng công
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setAdvancePaymentExportDialogOpen(true)}>
                <Wallet className="w-4 h-4 mr-2" />
                Ứng lương
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <PayrollReportExportDialog
        open={payrollReportDialogOpen}
        onOpenChange={setPayrollReportDialogOpen}
        onExport={handlePayrollReportExportSubmit}
        isLoading={exportPayrollReportMutation.isPending}
      />
      <AdvancePaymentExportDialog
        open={advancePaymentExportDialogOpen}
        onOpenChange={setAdvancePaymentExportDialogOpen}
      />
    </TooltipProvider>
  );
};
