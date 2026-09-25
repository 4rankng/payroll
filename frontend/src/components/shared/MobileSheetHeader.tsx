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
        'flex items-center justify-between px-4 pt-3 pb-3 border-b shrink-0',
        className,
      )}
    >
      <div className="flex items-center gap-3 min-w-0">
        {Icon && (
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10 shrink-0">
            <Icon className="h-[18px] w-[18px] text-primary" strokeWidth={2} />
          </div>
        )}
        <div className="min-w-0">
          <SheetTitle className="text-base font-semibold leading-tight truncate">
            {title}
          </SheetTitle>
          {description && (
            <SheetDescription className="text-xs text-muted-foreground mt-0.5">
              {description}
            </SheetDescription>
          )}
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="h-11 w-11 shrink-0 rounded-full text-muted-foreground hover:bg-muted hover:text-foreground"
        onClick={onClose}
        aria-label="Đóng"
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
};
