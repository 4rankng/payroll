import { memo, useMemo } from 'react';
import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';

export interface TransactionSummaryStripProps {
  total: number;
  completed: number;
  failed: number;
  className?: string;
}

/**
 * Transaction result summary strip — total / completed / failed counts.
 * Used in TransferSheet detail dialog and ResultUploadDialog.
 */
export const TransactionSummaryStrip = memo(function TransactionSummaryStrip({
  total,
  completed,
  failed,
  className,
}: TransactionSummaryStripProps) {
  const items = useMemo<InlineStatItem[]>(
    () => [
      { label: 'Tổng', value: total },
      { label: 'Thành công', value: completed },
      { label: 'Thất bại', value: failed, highlight: failed > 0 },
    ],
    [total, completed, failed],
  );

  return <InlineStatStrip items={items} className={className} />;
});
