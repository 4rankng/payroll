import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import { Plus, Download, PhoneOff, PhoneIncoming } from 'lucide-react';
import { useAuth } from '@/contexts';
import { cn } from '@/lib/utils';
import type { ComponentType, SVGProps } from 'react';

interface EmployeePageHeaderProps {
  totalEmployees: number;
  onAddEmployeeClick: () => void;
  onExportClick?: () => void;
  onExportNoMobileClick?: () => void;
  onImportMobilesClick?: () => void;
  onSearchFocus?: () => void;
  isExporting?: boolean;
}

interface ToolbarSegment {
  key: string;
  label: string;
  onClick: () => void;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  primary?: boolean;
  inert?: boolean;
}

// PAYR-33 toolbar (shared with TimesheetPage): actions render as one bordered
// segmented strip so they read — and wrap — as a single unit instead of loose
// buttons breaking into ragged rows at laptop widths.
const SEGMENT_BASE =
  'inline-flex items-center gap-1.5 h-9 px-3 text-sm font-medium whitespace-nowrap transition-colors';

export const EmployeePageHeader = ({
  totalEmployees,
  onAddEmployeeClick,
  onExportClick,
  onExportNoMobileClick,
  onImportMobilesClick,
  isExporting = false,
}: EmployeePageHeaderProps) => {
  const { user } = useAuth();
  const { pathname } = useLocation();
  const isAdmin = user?.role === 'admin';

  const segments: ToolbarSegment[] = [];
  if (onExportClick && (isAdmin || user?.role === 'partner')) {
    segments.push({
      key: 'export',
      label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
      onClick: onExportClick,
      icon: Download,
      inert: isExporting,
    });
  }
  if (onExportNoMobileClick && isAdmin) {
    segments.push({
      key: 'export-no-mobile',
      label: isExporting ? 'Đang xuất...' : 'NV chưa có SĐT',
      onClick: onExportNoMobileClick,
      icon: PhoneOff,
      inert: isExporting,
    });
  }
  if (onImportMobilesClick && isAdmin) {
    segments.push({
      key: 'import-mobiles',
      label: 'Nhập SĐT',
      onClick: onImportMobilesClick,
      icon: PhoneIncoming,
    });
  }
  segments.push({
    key: 'add-employee',
    label: 'Thêm nhân viên',
    onClick: onAddEmployeeClick,
    icon: Plus,
    primary: true,
  });

  return (
    <PageHeader
      kicker={getConsoleKicker(pathname)}
      title="Nhân viên"
      meta={`${totalEmployees.toLocaleString('vi-VN')} nhân viên`}
    >
      <div
        data-testid="employee-header-toolbar"
        className="flex w-full max-w-full flex-wrap items-center gap-px overflow-hidden rounded-xl border border-border bg-card sm:w-auto"
      >
        {segments.map((segment, index) => (
          <button
            key={segment.key}
            onClick={segment.onClick}
            aria-busy={segment.inert || undefined}
            className={cn(
              SEGMENT_BASE,
              index < segments.length - 1 && 'border-r border-border',
              segment.primary
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'bg-card text-foreground hover:bg-muted',
              segment.inert && 'pointer-events-none opacity-50',
            )}
          >
            <segment.icon className="h-4 w-4 shrink-0" aria-hidden="true" />
            {segment.label}
          </button>
        ))}
      </div>
    </PageHeader>
  );
};
