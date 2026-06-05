import { Card, CardContent } from '@/components/ui/card';
import { transactionService } from '@/services/api/transaction.service';
import { downloadAssetFile } from '@/utils/file-download';
import { getUserFullName } from '@/utils/userHelpers';
import type { Settlement } from '@/services/api/transaction.service';
import type { UserListResponse } from '@/services/api/user.service';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { FileText, Calendar, User } from 'lucide-react';

interface SettlementCardProps {
  settlement: Settlement;
  userMap?: Map<number, UserListResponse['data'][0]> | Record<number, UserListResponse['data'][0]>;
}

export function SettlementCard({ settlement, userMap }: SettlementCardProps) {
  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'dd/MM/yyyy', { locale: vi });
    } catch {
      return dateString;
    }
  };

  const handleDownload = async () => {
    if (!settlement.proof_asset_id && !settlement.proof_url) return;

    if (settlement.proof_asset_id) {
      const filename = settlement.proof_asset?.original_filename
        || settlement.proof_asset?.filename
        || `chung_tu_thanh_toan_${settlement.id}.pdf`;

      await downloadAssetFile(settlement.proof_asset_id, filename);
    } else if (settlement.proof_url) {
      window.open(settlement.proof_url, '_blank', 'noopener,noreferrer');
    }
  };

  const createdByName = getUserFullName(settlement.created_by, userMap);
  const hasProof = !!settlement.proof_asset_id || !!settlement.proof_url;

  return (
    <Card className="transition-shadow max-w-[250px]">
      <CardContent className="p-3">
        <div className="flex items-start justify-between gap-3">
          {/* Left: Amount, Date, Person */}
          <div className="flex flex-col gap-1.5 min-w-0 flex-1">
            {/* Amount */}
            <span className="text-lg font-semibold text-emerald-700">
              {transactionService.formatCurrency(settlement.amount)}
            </span>

            {/* Date */}
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Calendar className="h-3 w-3 flex-shrink-0" />
              <span>{formatDate(settlement.settlement_date)}</span>
            </div>

            {/* Created By */}
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <User className="h-3 w-3 flex-shrink-0" />
              <span className="truncate">{createdByName}</span>
            </div>
          </div>

          {/* Right: Document icon */}
          <button
            type="button"
            onClick={handleDownload}
            disabled={!hasProof}
            className={`flex-shrink-0 rounded-xl p-2 transition-all ${
              hasProof
                ? 'text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 hover:shadow-sm cursor-pointer'
                : 'text-slate-300 cursor-not-allowed'
            }`}
            title={hasProof ? 'Tải xuống chứng từ' : 'Không có chứng từ'}
          >
            <FileText className="h-12 w-12" />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}
