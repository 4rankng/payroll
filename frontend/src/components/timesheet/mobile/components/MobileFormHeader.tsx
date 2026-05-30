import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Clock } from 'lucide-react';

export function MobileFormHeader() {
  return (
    <Card className="border-none shadow-sm bg-card">
      <CardHeader className="pb-2 sm:pb-3 pt-3 sm:pt-6">
        <CardTitle className="typography-title-large sm:typography-headline-medium text-center flex items-center justify-center gap-2">
          <Clock className="h-4 w-4 sm:h-5 sm:w-5 text-primary" />
          Nhập Bảng Công
        </CardTitle>
      </CardHeader>
    </Card>
  );
}