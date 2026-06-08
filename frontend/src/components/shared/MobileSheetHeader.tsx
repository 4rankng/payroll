import { type LucideIcon, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';
import { cn } from '@/lib/utils';

interface MobileSheetHeaderProps {
  /** Icon displayed in a soft primary container */
  icon?: LucideIcon;
  /** Sheet title */
  title: string;
  /** Optional description below the title */
  description?: string;
  /** Close handler — typically `() => setOpen(false)` */
  onClose: () => void;
  /** Optional class override */
  className?: string;
}

/**
 * Consistent sheet header with icon, title, description, and close button.
 * Matches the MobileSubPageHeader visual pattern.
 *
 * Used in bottom sheets across the app for uniform look & feel.
 */
export const MobileSheetHeader = ({
  icon: Icon,
  title,
  description,
  onClose,
  className,
}: MobileSheetHeaderProps) => {
  return (
    <div
      className={cn(
        'flex items-center justify-between px-4 pt-5 pb-3 border-b shrink-0',
        className,
      )}
    >
      <div className="flex items-center gap-2.5 min-w-0">
        {Icon && (
          <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/8 shrink-0">
            <Icon className="h-[16px] w-[16px] text-primary/70" strokeWidth={2} />
          </div>
        )}
        <div className="min-w-0">
          <SheetTitle className="text-base font-display font-bold leading-tight truncate">
            {title}
          </SheetTitle>
          {description && (
            <SheetDescription className="text-xs mt-0.5">
              {description}
            </SheetDescription>
          )}
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 shrink-0"
        onClick={onClose}
        aria-label="Đóng"
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
};
