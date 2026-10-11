import { useState } from 'react';
import { ChevronDown, ChevronRight, Lightbulb } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { useDismissInsight, useInsightDigest } from '@/hooks/api/useInsights';
import { cn } from '@/lib/utils';

const SEVERITY_DOT: Record<string, string> = {
  critical: 'bg-red-500',
  warning: 'bg-amber-500',
  info: 'bg-slate-400',
};

const findingLine = (narrative: string, facts: Record<string, unknown> | undefined): string => {
  if (narrative) return narrative;
  const summary = facts?.summary;
  return typeof summary === 'string' && summary ? summary : 'Phát hiện vận hành mới';
};

/** Daily ops digest on Tổng Quan: deterministic signals, LLM-written
 * narratives, ≤5 items. Renders nothing when the digest is empty (non-zero
 * law) — silence IS the "nothing important" signal. */
export const InsightDigestCard = ({ compact = false }: { compact?: boolean }) => {
  const { data, isLoading } = useInsightDigest();
  const [open, setOpen] = useState(false);
  const dismiss = useDismissInsight();

  const findings = (data ?? []).filter((f) => !f.dismissed_at);
  if (isLoading || findings.length === 0) return null;

  const criticalCount = findings.filter((f) => f.severity === 'critical').length;

  return (
    <section
      aria-label="Phát hiện trong ngày"
      className={cn('rounded-xl border bg-card text-card-foreground', compact ? 'px-4 py-3' : 'p-4 sm:p-5')}
    >
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center gap-2 text-left"
      >
        {open ? (
          <ChevronDown aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRight aria-hidden="true" className="size-4 shrink-0 text-muted-foreground" />
        )}
        <Lightbulb aria-hidden="true" className="size-4 shrink-0 text-indigo-600" />
        <span className="text-sm font-semibold text-foreground">
          Phát hiện trong ngày ({findings.length})
        </span>
        {criticalCount > 0 ? (
          <span className="ml-auto rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800">
            {criticalCount} quan trọng
          </span>
        ) : null}
      </button>
      {open ? (
        <ul className="mt-3 space-y-2">
          {findings.slice(0, 5).map((finding) => (
            <li key={finding.id} className="flex items-start gap-2.5 rounded-lg border bg-background px-3 py-2.5">
              <span
                aria-hidden="true"
                className={cn('mt-1.5 size-2 shrink-0 rounded-full', SEVERITY_DOT[finding.severity] ?? 'bg-slate-400')}
              />
              <p className="min-w-0 flex-1 text-sm leading-5 text-foreground">
                {findingLine(finding.narrative, finding.facts)}
              </p>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8 shrink-0 px-2 text-xs text-muted-foreground"
                disabled={dismiss.isPending}
                onClick={() => dismiss.mutate(finding.id)}
                aria-label="Ẩn phát hiện này trong 7 ngày"
              >
                Ẩn 7 ngày
              </Button>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
