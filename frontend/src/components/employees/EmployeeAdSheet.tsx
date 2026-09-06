import { Check, ExternalLink, Megaphone, Phone, X } from 'lucide-react';

import { Sheet, SheetContent } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import type { AdBanner, AdBannerCTA, AdBannerCTAType } from '@/types/api/ad-banner.types';

/** Brand mark for Zalo CTAs — public asset, served same-origin. */
const ZALO_ICON_SRC = '/employee/zalo-icon.webp';

/**
 * Icon per CTA type — shared by the compact card and the sheet so both
 * surfaces render the same marks. Zalo uses its logo image; everything else
 * stays on lucide line icons.
 */
export const AdCTAIcon = ({ type, className }: { type: AdBannerCTAType; className: string }) => {
  if (type === 'phone') {
    return <Phone className={className} aria-hidden="true" />;
  }
  if (type === 'zalo') {
    return <img src={ZALO_ICON_SRC} alt="" aria-hidden="true" className={className} />;
  }
  return <ExternalLink className={className} aria-hidden="true" />;
};

interface AdContentProps {
  banner: AdBanner;
  /** Fired on CTA taps; the caller owns click recording + navigation. */
  onCTAClick?: (cta: AdBannerCTA, index: number) => void;
  /** When provided, the header renders a close control. Admin preview omits it. */
  onDismiss?: () => void;
  /** Rendered after the CTA block — the sheet passes its dismiss button. */
  footerSlot?: React.ReactNode;
}

/**
 * The campaign markup — single source shared by the employee bottom sheet
 * and the admin composer's live preview, so the two cannot drift.
 *
 * Hierarchy is deliberate: a branded header strip identifies the message,
 * the benefit list sits in one soft emerald panel, and the first CTA is the
 * solid primary action (admins control ordering, so index 0 is the ask).
 */
export const EmployeeAdContent = ({
  banner,
  onCTAClick,
  onDismiss,
  footerSlot,
}: AdContentProps) => (
  <div className="flex flex-col">
    {/* The header exists to carry the headline, not a label bar: the emerald
        wash and the icon are the only chrome, and the close control floats so
        it costs no vertical space. */}
    <header className="relative bg-[linear-gradient(135deg,var(--employee-accent)_0%,var(--employee-accent-strong)_100%)] px-4 py-3.5 pr-12">
      {onDismiss && (
        <button
          type="button"
          aria-label="Đóng thông báo"
          onClick={onDismiss}
          className="absolute right-1.5 top-1.5 flex h-9 w-9 items-center justify-center rounded-full text-white/70 transition-colors hover:bg-white/15 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/70"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      )}

      <div className="flex items-start gap-2.5">
        <Megaphone aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-white/80" />
        <h2 className="employee-type-strong leading-snug text-white">{banner.title}</h2>
      </div>
    </header>

    <div className="space-y-4 bg-[var(--employee-surface)] px-4 pb-6 pt-4">
      {banner.body && (
        <p className="employee-type-body-sm text-[var(--employee-text-secondary)]">{banner.body}</p>
      )}

      {banner.bullets.length > 0 && (
        <ul className="space-y-2.5 rounded-[var(--employee-radius-card)] border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-3.5 py-3.5">
          {banner.bullets.map((bullet) => (
            <li
              key={bullet}
              className="employee-type-body-sm flex items-start gap-2.5 text-[var(--employee-text)]"
            >
              <span
                aria-hidden="true"
                className="mt-px flex h-[1.125rem] w-[1.125rem] shrink-0 items-center justify-center rounded-full bg-[var(--employee-accent)] text-white"
              >
                <Check className="h-3 w-3" strokeWidth={3} />
              </span>
              <span className="leading-snug">{bullet}</span>
            </li>
          ))}
        </ul>
      )}

      {banner.ctas.length > 0 && (
        <div className="flex flex-col gap-2.5 pt-1">
          {banner.ctas.map((cta, index) => {
            const isPrimary = index === 0;

            return (
              <button
                key={`${cta.label}-${index}`}
                type="button"
                onClick={() => onCTAClick?.(cta, index)}
                className={cn(
                  'flex h-11 w-full items-center justify-center gap-2 rounded-[var(--employee-radius-control)] employee-type-body font-semibold transition-colors active:scale-[0.99]',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2',
                  isPrimary
                    ? 'bg-[var(--employee-accent)] text-white shadow-[var(--employee-cta-shadow)] hover:bg-[var(--employee-accent-strong)]'
                    : 'border border-[var(--employee-border-strong)] bg-[var(--employee-surface)] text-[var(--employee-text)] hover:border-[var(--employee-accent-border)] hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)]',
                )}
              >
                <AdCTAIcon type={cta.type} className="h-4 w-4" />
                {cta.label}
              </button>
            );
          })}
        </div>
      )}

      {banner.footer && (
        <p className="employee-type-label border-t border-[var(--employee-border)] pt-4 text-center text-[var(--employee-text-muted)]">
          {banner.footer}
        </p>
      )}

      {footerSlot}
    </div>
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
        className={cn(
          '!w-full sm:!w-[420px] flex flex-col overflow-hidden bg-[var(--employee-surface)] p-0 text-[var(--employee-text)]',
          isMobile
            ? 'max-h-[88vh] rounded-t-[1.5rem] shadow-[0_-8px_32px_rgba(16,24,40,0.16)]'
            : 'shadow-none',
        )}
      >
        {isMobile && (
          <div className="flex shrink-0 justify-center pb-1 pt-2.5">
            <span
              aria-hidden="true"
              className="h-1 w-9 rounded-full bg-[var(--employee-border-strong)]"
            />
          </div>
        )}

        <div className="flex-1 overflow-y-auto">
          <EmployeeAdContent
            banner={banner}
            onCTAClick={onCTAClick}
            onDismiss={onClose}
            footerSlot={
              <button
                type="button"
                onClick={onClose}
                className="employee-type-body h-11 w-full rounded-[var(--employee-radius-control)] font-semibold text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-surface-muted)] hover:text-[var(--employee-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
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
