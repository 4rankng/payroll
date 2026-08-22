import { memo, type ReactNode } from 'react';
import { type LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  EmptyStateIllustration,
  type EmptyStateIllustrationVariant,
} from '@/components/shared/EmptyStateIllustration';
import { cn } from '@/lib/utils';

export interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  children?: ReactNode;
  variant?: EmptyStateIllustrationVariant;
  size?: 'sm' | 'default';
  className?: string;
}

function inferVariant(title: string, description?: string): EmptyStateIllustrationVariant {
  const context = `${title} ${description ?? ''}`.toLocaleLowerCase('vi-VN');

  if (/nhân viên|người dùng|tài khoản|chấm công/.test(context)) return 'employees';
  if (/dự án/.test(context)) return 'projects';
  if (/thanh toán|ngân hàng|giao dịch|bút toán|khoản vay|lương|sao kê|ví|tài chính/.test(context)) return 'finance';
  if (/hoạt động|lịch sử|email|tệp|file|thông báo|nhật ký|tải lên/.test(context)) return 'activity';
  if (/không tìm thấy|bộ lọc|tìm kiếm/.test(context)) return 'search';

  return 'records';
}

/**
 * Standard empty state — icon, primary message, optional description, optional action.
 * Implements Requirement 6 criterion 2.
 */
export const EmptyState = memo(function EmptyState({
  title,
  description,
  action,
  children,
  variant,
  size = 'default',
  className,
}: EmptyStateProps) {
  return (
    <div
      data-slot="empty-state"
      data-admin-surface="empty-state"
      className={cn(
        'admin-empty-state flex items-center justify-center gap-4 text-left',
        size === 'default' ? 'py-6 sm:py-8' : 'py-4',
        className,
      )}
    >
      <EmptyStateIllustration
        variant={variant ?? inferVariant(title, description)}
        className={size === 'default' ? 'h-24 w-24 sm:h-28 sm:w-28' : 'h-16 w-16 sm:h-20 sm:w-20'}
      />
      <div className="min-w-0 max-w-md">
        <p className="text-xs font-semibold text-foreground">{title}</p>
        {description && (
          <p className="mt-1 text-[11px] leading-relaxed text-muted-foreground">{description}</p>
        )}
        {action && (
          <Button variant="outline" size="sm" onClick={action.onClick} className="mt-2">
            {action.label}
          </Button>
        )}
        {children}
      </div>
    </div>
  );
});
