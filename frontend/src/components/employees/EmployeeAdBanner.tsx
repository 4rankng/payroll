import { useEffect, useState } from 'react';
import { X } from 'lucide-react';

import { useEmployeeAdBanner, useRecordAdBannerClick } from '@/hooks/api/useEmployeeAdBanner';
import type { AdBanner, AdBannerCTA } from '@/types/api/ad-banner.types';

import { EmployeeAdSheet } from './EmployeeAdSheet';

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
    window.open(value, '_blank', 'noopener,noreferrer');
  };

  return (
    <>
      {stage === 'card' && (
        <section aria-label="Quảng cáo" className="employee-surface-card px-4 py-3">
          <div className="flex items-start justify-between gap-2">
            <p className="employee-type-label-caps text-[var(--employee-accent)]">Quảng cáo</p>
            <button
              type="button"
              aria-label="Đóng quảng cáo"
              onClick={() => persistStage('hidden')}
              className="-mr-1 -mt-1 rounded-lg p-1.5 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>

          <h3 className="employee-type-label mt-1 font-semibold text-[var(--employee-text)]">
            {banner.title}
          </h3>

          <p className="employee-type-body-sm mt-1 line-clamp-2 text-[var(--employee-text-secondary)]">
            {banner.body || banner.bullets[0] || ''}
          </p>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            {banner.ctas.map((cta, index) => (
              <button
                key={`${cta.label}-${index}`}
                type="button"
                onClick={() => handleCTAClick(cta, index)}
                className="employee-type-body-sm h-9 rounded-full border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-3.5 font-semibold text-[var(--employee-accent)] transition-colors hover:bg-[var(--employee-accent)] hover:text-white active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
              >
                {cta.label}
              </button>
            ))}
            <button
              type="button"
              onClick={() => setSheetOpen(true)}
              className="employee-type-body-sm px-1 font-semibold text-[var(--employee-accent)] underline-offset-4 hover:underline focus-visible:outline-none"
            >
              Xem chi tiết
            </button>
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
