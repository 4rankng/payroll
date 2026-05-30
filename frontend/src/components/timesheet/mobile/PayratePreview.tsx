import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { DollarSign, TrendingUp, Calculator } from 'lucide-react';

interface PayratePreviewProps {
  payrate: number;
  hours: number;
  amount: number;
}

export function PayratePreview({
  payrate,
  hours,
  amount
}: PayratePreviewProps) {
  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency: 'VND',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0
    }).format(value);
  };

  const isHighAmount = amount > 500000; // Above 500k VND

  return (
    <Card className="border-2 border-primary/20 bg-gradient-to-br from-primary/5 to-primary/10">
      <CardContent className="pt-6">
        <div className="space-y-4">
          {/* Header */}
          <div className="flex items-center gap-2 mb-4">
            <Calculator className="h-5 w-5 text-primary" />
            <h3 className="font-semibold text-primary">Tính lương</h3>
          </div>

          {/* Calculation breakdown */}
          <div className="space-y-3">
            {/* Hourly rate */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <DollarSign className="h-4 w-4 text-muted-foreground" />
                <span className="typography-body-medium text-muted-foreground">Đơn giá/giờ:</span>
              </div>
              <Badge variant="outline" className="font-mono">
                {formatCurrency(payrate)}
              </Badge>
            </div>

            {/* Hours worked */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
                <span className="typography-body-medium text-muted-foreground">Số giờ:</span>
              </div>
              <Badge variant="secondary" className="font-mono">
                {hours}h
              </Badge>
            </div>

            {/* Divider */}
            <hr className="border-primary/20" />

            {/* Total amount */}
            <div className="flex items-center justify-between">
              <span className="font-medium text-foreground">Tổng tiền:</span>
              <Badge
                variant={isHighAmount ? "default" : "secondary"}
                className="typography-body-large px-3 py-1"
              >
                {formatCurrency(amount)}
              </Badge>
            </div>
          </div>

          {/* Calculation formula */}
          <div className="bg-muted/30 rounded-xl p-3 mt-4">
            <p className="typography-body-small text-muted-foreground text-center font-mono">
              {formatCurrency(payrate)} × {hours}h = {formatCurrency(amount)}
            </p>
          </div>

          {/* Amount indicators */}
          {amount > 0 && (
            <div className="flex items-center justify-center gap-2 mt-3">
              {isHighAmount && (
                <Badge variant="warning" className="typography-body-small">
                  Mức lương cao
                </Badge>
              )}
              {hours > 8 && (
                <Badge variant="info" className="typography-body-small">
                  Có tăng ca
                </Badge>
              )}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
