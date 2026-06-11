import { RefreshCw, Wallet as WalletIcon, ArrowDownCircle, ArrowUpCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatCurrency, formatDateTime } from '@/utils/formatters';
import type { WalletBalance } from '@/types/api/wallet.types';

interface WalletBalanceCardProps {
  balance: WalletBalance;
  onRefresh: () => void;
}

export default function WalletBalanceCard({ balance, onRefresh }: WalletBalanceCardProps) {
  const currency = balance.currency || 'VND';

  return (
    <Card className="bg-gradient-to-r from-blue-600 to-blue-700 text-white border-0 shadow-lg">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-white text-lg font-semibold flex items-center gap-2">
            <WalletIcon className="h-5 w-5" />
            Số dư ví
          </CardTitle>
          <Button
            variant="ghost"
            size="icon"
            onClick={onRefresh}
            className="text-white hover:bg-blue-800"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-end justify-between">
          <div>
            <p className="text-blue-100 text-sm mb-1">Số dư khả dụng</p>
            <p className="text-4xl font-bold text-white">
              {formatCurrency(balance.available, currency)}
            </p>
          </div>
          <div className="text-right text-sm space-y-1">
            <div className="flex items-center gap-1 justify-end text-blue-100">
              <ArrowDownCircle className="h-3.5 w-3.5" />
              <span>Chờ nhận: {formatCurrency(balance.pending_in, currency)}</span>
            </div>
            <div className="flex items-center gap-1 justify-end text-blue-100">
              <ArrowUpCircle className="h-3.5 w-3.5" />
              <span>Chờ chuyển: {formatCurrency(balance.pending_out, currency)}</span>
            </div>
            <p className="text-blue-200/60 text-xs mt-1">
              Cập nhật: {formatDateTime(balance.as_of)}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
