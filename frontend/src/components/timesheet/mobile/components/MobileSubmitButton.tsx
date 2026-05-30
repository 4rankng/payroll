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
    <div className="fixed bottom-3 left-3 right-3 sm:bottom-4 sm:left-4 sm:right-4 max-w-md mx-auto">
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