import { useEffect, useState } from 'react';
import { ChevronRight, Megaphone, X } from 'lucide-react';

import { useEmployeeAdBanner, useRecordAdBannerClick } from '@/hooks/api/useEmployeeAdBanner';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import type { AdBanner, AdBannerCTA } from '@/types/api/ad-banner.types';

import { AdCTAIcon, EmployeeAdSheet } from './EmployeeAdSheet';

/** localStorage state: which campaign version the employee is done with. */
const STORAGE_KEY = 'employee_ad_state';

type Stage = 'sheet' | 'card' | 'hidden';

interface StoredAdState {
  id: number;
  version: string;
  stage: Stage;
}

const readStoredState = (): StoredAdState | null => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as StoredAdState) : null;
  } catch {
    return null;
  }
};

const writeStoredState = (state: StoredAdState) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // localStorage unavailable (private mode) — the employee simply re-sees
    // the sheet next visit; nothing here is worth failing over.
  }
};

/**
 * Employee ad slot: one campaign at a time, three-stage dismissal.
 *
 * 1. Fresh campaign version → sheet auto-opens once.
 * 2. Dismiss sheet → compact card stays in the feed (hotline stays findable).
 * 3. Dismiss card → nothing for this version.
 *
 * An admin edit bumps `updatedAt` (the version), restarting the cycle — that
 * is a republish, not a leak.
 */
export const EmployeeAdBanner = () => {
  const { data: banner, isLoading } = useEmployeeAdBanner();
  const recordClick = useRecordAdBannerClick();
  const isMobile = useIsMobile();
  const [sheetOpen, setSheetOpen] = useState(false);
  const [stage, setStage] = useState<Stage | null>(null);

  useEffect(() => {
    if (!banner) return;
    const stored = readStoredState();
    const seenThisVersion =
      stored !== null && stored.id === banner.id && stored.version === banner.updatedAt;

    if (!seenThisVersion) {
      setStage('sheet');
      setSheetOpen(true);
    } else {
      setStage(stored.stage);
      setSheetOpen(false);
    }
    // Re-runs on refetch are idempotent: the decision depends only on the
    // stored state versus (id, updatedAt), so a new object identity with the
    // same version lands on the same stage.
  }, [banner]);

  if (isLoading || !banner || stage === null || stage === 'hidden') {
    return null;
  }

  const persistStage = (next: Stage) => {
    setStage(next);
    writeStoredState({ id: banner.id, version: banner.updatedAt, stage: next });
  };

  const dismissSheet = () => {
    setSheetOpen(false);
    persistStage('card');
  };

  const handleCTAClick = (cta: AdBannerCTA, index: number) => {
    // Fire-and-forget: tracking must never gate the hotline.
    recordClick.mutate({ bannerId: banner.id, ctaIndex: index });

    const value = cta.value.trim();
    if (cta.type === 'phone') {
      window.location.href = `tel:${value}`;
      return;
    }
    if (cta.type === 'zalo' && isMobile) {
      // zalo.me links are universal links: same-tab navigation is what lets
      // the OS hand the tap to the Zalo app — a new tab would keep it in the
      // browser instead of switching. Desktop keeps the tab-based open.
      window.location.href = value;
      return;
    }
    window.open(value, '_blank', 'noopener,noreferrer');
  };

  return (
    <>
      {stage === 'card' && (
        <section aria-label="Quảng cáo" className="employee-surface-card relative overflow-hidden">
          <button
            type="button"
            aria-label="Đóng quảng cáo"
            onClick={() => persistStage('hidden')}
            className="absolute right-1 top-1 flex h-9 w-9 items-center justify-center rounded-full text-[var(--employee-text-muted)] transition-colors hover:bg-[var(--employee-surface-muted)] hover:text-[var(--employee-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>

          <div className="px-4 py-3.5">
            <div className="flex items-center gap-1.5 pr-8">
              <Megaphone className="h-3.5 w-3.5 text-[var(--employee-accent)]" aria-hidden="true" />
              <span className="employee-type-label-caps text-[var(--employee-accent)]">
                Quảng cáo
              </span>
            </div>

            <h3 className="employee-type-strong mt-1 font-semibold leading-snug text-[var(--employee-text)]">
              {banner.title}
            </h3>

            <p className="employee-type-body-sm mt-1.5 line-clamp-2 leading-relaxed text-[var(--employee-text-secondary)]">
              {banner.body || banner.bullets[0] || ''}
            </p>

            {/* Sits directly under the clamped copy — it continues that
                sentence, so it belongs with the text, not with the CTA row. */}
            <button
              type="button"
              onClick={() => setSheetOpen(true)}
              className="employee-type-body-sm mt-0.5 inline-flex items-center gap-0.5 font-semibold text-[var(--employee-accent)] underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
            >
              Xem chi tiết
              <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
            </button>

            <div className="mt-3 flex flex-wrap items-center gap-1.5">
              {banner.ctas.map((cta, index) => (
                <button
                  key={`${cta.label}-${index}`}
                  type="button"
                  onClick={() => handleCTAClick(cta, index)}
                  className={cn(
                    'employee-type-body-sm inline-flex h-9 items-center gap-1.5 rounded-full px-3.5 font-semibold transition-colors active:scale-[0.98]',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2',
                    index === 0
                      ? 'bg-[var(--employee-accent)] text-white hover:bg-[var(--employee-accent-strong)]'
                      : 'border border-[var(--employee-border)] bg-[var(--employee-surface)] text-[var(--employee-text-secondary)] hover:border-[var(--employee-accent-border)] hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)]',
                  )}
                >
                  <AdCTAIcon type={cta.type} className="h-3.5 w-3.5" />
                  {cta.label}
                </button>
              ))}
            </div>
          </div>
        </section>
      )}

      <EmployeeAdSheet
        open={sheetOpen}
        onClose={dismissSheet}
        banner={banner}
        onCTAClick={handleCTAClick}
      />
    </>
  );
};
