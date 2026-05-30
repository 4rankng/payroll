import { memo } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Info } from 'lucide-react';

interface RatePreviewCardProps {
  ratePreview?: number;
  hoursWorked: number;
}

export const RatePreviewCard = memo(({ ratePreview, hoursWorked }: RatePreviewCardProps) => {
  if (!ratePreview) return null;

  return (
    <Alert>
      <Info className="h-4 w-4" />
      <AlertDescription>
        Mức lương dự kiến: <strong>{ratePreview.toLocaleString('vi-VN')} VND/giờ</strong>
        {hoursWorked > 0 && (
          <span> → Tổng: <strong>{(ratePreview * hoursWorked).toLocaleString('vi-VN')} VND</strong></span>
        )}
      </AlertDescription>
    </Alert>
  );
});

RatePreviewCard.displayName = 'RatePreviewCard';