import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

interface MobileSectionHeaderProps {
  /** Section icon — rendered in a soft primary/5 container */
  icon: LucideIcon;
  /** Section title */
  title: string;
  /** Optional right-side content (selectors, badges, etc.) */
  children?: React.ReactNode;
  /** Optional class override for the outer wrapper */
  className?: string;
}

/**
 * Consistent section header for mobile pages.
 * Small icon in a rounded container + bold title + optional trailing content.
 *
 * Pattern extracted from the Dashboard's local SectionHeader.
 * Typography: font-display text-sm font-bold tracking-tight.
 */
export const MobileSectionHeader = ({
  icon: Icon,
  title,
  children,
  className,
}: MobileSectionHeaderProps) => {
  return (
    <div className={cn('flex items-center gap-2 mb-3', className)}>
      <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/5">
        <Icon className="h-3.5 w-3.5 text-primary/70" strokeWidth={2} />
      </div>
      <h2 className="font-display text-sm font-bold text-foreground tracking-tight leading-tight">
        {title}
      </h2>
      {children}
    </div>
  );
};
