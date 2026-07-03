import { Button } from '@/components/ui/button';
import { Save } from 'lucide-react';

interface MobileSubmitButtonProps {
  isFormValid: boolean;
  isSubmitting: boolean;
  onSubmit: () => void;
}

export function MobileSubmitButton({
  isFormValid,
  isSubmitting,
  onSubmit
}: MobileSubmitButtonProps) {
  return (
    <div className="fixed inset-x-3 bottom-[calc(0.75rem+env(safe-area-inset-bottom))] z-40 mx-auto max-w-md sm:inset-x-4 sm:bottom-[calc(1rem+env(safe-area-inset-bottom))]">
      <Button
        onClick={onSubmit}
        disabled={!isFormValid || isSubmitting}
        className="w-full h-14 sm:h-12 typography-body-large rounded-xl touch-manipulation active:scale-95 transition-transform shadow-sm"
        size="lg"
      >
        <Save className="w-5 h-5 mr-2" />
        {isSubmitting ? 'Đang lưu...' : 'Lưu Bảng Công'}
      </Button>
    </div>
  );
}
