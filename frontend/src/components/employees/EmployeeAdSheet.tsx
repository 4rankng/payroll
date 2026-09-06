import { Phone, ExternalLink } from 'lucide-react';

import { Sheet, SheetContent } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import type { AdBanner, AdBannerCTA } from '@/types/api/ad-banner.types';

interface AdContentProps {
  banner: AdBanner;
  /** Fired on CTA taps; the caller owns click recording + navigation. */
  onCTAClick?: (cta: AdBannerCTA, index: number) => void;
  /** Rendered after the CTA block — the sheet passes its dismiss button. */
  footerSlot?: React.ReactNode;
}

/**
 * The campaign markup — single source shared by the employee bottom sheet
 * and the admin composer's live preview, so the two cannot drift.
 */
export const EmployeeAdContent = ({ banner, onCTAClick, footerSlot }: AdContentProps) => (
  <div className="space-y-4 px-4 pb-6 pt-4">
    <p className="employee-type-label-caps text-[var(--employee-accent)]">Thông báo</p>

    <h2 className="employee-type-section-title text-[var(--employee-text)]">
      {banner.title}
    </h2>

    {banner.body && (
      <p className="employee-type-body-sm text-[var(--employee-text-secondary)]">
        {banner.body}
      </p>
    )}

    {banner.bullets.length > 0 && (
      <ul className="space-y-2">
        {banner.bullets.map((bullet) => (
          <li
            key={bullet}
            className="employee-type-body-sm flex gap-2 text-[var(--employee-text)]"
          >
            <span
              aria-hidden="true"
              className="mt-[0.4rem] h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--employee-accent)]"
            />
            <span>{bullet}</span>
          </li>
        ))}
      </ul>
    )}

    {banner.ctas.length > 0 && (
      <div className="flex flex-col gap-2 pt-1">
        {banner.ctas.map((cta, index) => (
          <button
            key={`${cta.label}-${index}`}
            type="button"
            onClick={() => onCTAClick?.(cta, index)}
            className={cn(
              'flex h-12 w-full items-center justify-center gap-2 rounded-xl border text-[0.9375rem] font-semibold transition-colors',
              'border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]',
              'hover:bg-[var(--employee-accent)] hover:text-white active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]',
            )}
          >
            {cta.type === 'phone' ? (
              <Phone className="h-4 w-4" aria-hidden="true" />
            ) : (
              <ExternalLink className="h-4 w-4" aria-hidden="true" />
            )}
            {cta.label}
          </button>
        ))}
      </div>
    )}

    {banner.footer && (
      <p className="employee-type-label text-[var(--employee-text-secondary)]">
        {banner.footer}
      </p>
    )}

    {footerSlot}
  </div>
);

interface EmployeeAdSheetProps {
  open: boolean;
  onClose: () => void;
  banner: AdBanner;
  onCTAClick?: (cta: AdBannerCTA, index: number) => void;
}

/**
 * Full-campaign bottom sheet for the employee portal.
 */
export const EmployeeAdSheet = ({ open, onClose, banner, onCTAClick }: EmployeeAdSheetProps) => {
  const isMobile = useIsMobile();

  return (
    <Sheet open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <SheetContent
        side={isMobile ? 'bottom' : 'right'}
        title="Thông báo"
        description="Quảng cáo"
        data-theme="employee"
        className="!w-full sm:!w-[420px] flex flex-col overflow-hidden bg-[var(--employee-surface)] p-0 text-[var(--employee-text)] shadow-none"
      >
        <div className="flex-1 overflow-y-auto">
          <EmployeeAdContent
            banner={banner}
            onCTAClick={onCTAClick}
            footerSlot={
              <button
                type="button"
                onClick={onClose}
                className="employee-type-body-sm w-full py-2 text-center font-semibold text-[var(--employee-text-secondary)] underline-offset-4 hover:underline focus-visible:outline-none"
              >
                Đóng
              </button>
            }
          />
        </div>
      </SheetContent>
    </Sheet>
  );
};
