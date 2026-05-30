import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Calendar, Download, ExternalLink } from 'lucide-react';
import { PaginationControls } from '@/components/ui/pagination-controls';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';
import { ledgerService } from '@/services/api/ledger.service';
import { assetService } from '@/services/api/asset.service';
import { useDownloadAsset } from '@/hooks/api/useAssets';
import type { LedgerEntry, AccountMetadata } from '@/types/api/financial.types';
import type { Project } from '@/types/api/project.types';

interface LedgerMobileListProps {
  entries: LedgerEntry[];
  isLoading?: boolean;
  onRowClick?: (entry: LedgerEntry) => void;
  projects?: Project[];
  accountMetadata?: AccountMetadata[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}

const ACCOUNT_COLORS: Record<string, string> = {
  cash: 'bg-emerald-100 text-emerald-800',
  receivable: 'bg-blue-100 text-blue-800',
  payable: 'bg-red-100 text-red-800',
  revenue: 'bg-purple-100 text-purple-800',
  expense: 'bg-orange-100 text-orange-800',
  equity: 'bg-slate-100 text-slate-800',
  loan: 'bg-amber-100 text-amber-800',
};

function getAmountColor(account: string, amount: number) {
  if (['cash', 'receivable', 'revenue'].includes(account)) return amount >= 0 ? 'text-emerald-700' : 'text-red-600';
  if (account === 'expense') return amount > 0 ? 'text-orange-600' : 'text-muted-foreground';
  if (account === 'payable') return amount > 0 ? 'text-red-600' : 'text-emerald-600';
  return 'text-foreground';
}

export function LedgerMobileList({
  entries,
  isLoading = false,
  onRowClick,
  projects = [],
  accountMetadata = [],
  pagination,
  onPageChange,
  onPageSizeChange,
}: LedgerMobileListProps) {
  const downloadAsset = useDownloadAsset();

  const fmtDate = (d: string) => {
    try { return format(new Date(d), 'dd/MM/yyyy', { locale: vi }); } catch { return d; }
  };

  const fmtCurrency = (n: number) => n === 0 ? '0 đ' : ledgerService.formatCurrency(n);

  const getAccountLabel = (account: string) => {
    const meta = accountMetadata.find(m => m.value === account);
    return meta?.label ?? account;
  };

  const getProjectName = (id?: number) => projects.find(p => p.id === id)?.name;

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="bg-card rounded-xl border border-border p-4 animate-pulse">
            <div className="flex justify-between mb-2">
              <div className="h-4 bg-gray-200 rounded w-2/3" />
              <div className="h-4 bg-gray-200 rounded w-1/5" />
            </div>
            <div className="h-3 bg-muted rounded w-1/2" />
          </div>
        ))}
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="text-center py-12">
        <Calendar className="mx-auto h-10 w-10 text-muted-foreground/40" />
        <p className="mt-3 text-sm font-medium text-muted-foreground">Không tìm thấy bút toán nào</p>
        <p className="mt-1 text-xs text-muted-foreground">Thử điều chỉnh bộ lọc</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {entries.map((entry) => {
          const displayAmount = Math.max(entry.debit, entry.credit);
          const amountColor = getAmountColor(entry.account, entry.net_amount);
          const projectName = getProjectName(entry.project_id);
          const evidence = ledgerService.getEvidenceDisplay(entry);

          return (
            <div
              key={entry.id}
              onClick={() => onRowClick?.(entry)}
              className="bg-card border border-border rounded-xl p-3.5 shadow-sm card-lift transition-all cursor-pointer touch-manipulation"
              role="button"
              tabIndex={0}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') onRowClick?.(entry); }}
            >
              {/* Top: party + amount */}
              <div className="flex items-start justify-between gap-3 mb-1.5">
                <p className="text-sm font-semibold text-foreground line-clamp-1 flex-1">{entry.party}</p>
                <p className={cn('text-sm font-bold tabular-nums shrink-0', amountColor)}>
                  {fmtCurrency(displayAmount)}
                </p>
              </div>

              {/* Middle: description */}
              {entry.description && (
                <p className="text-xs text-muted-foreground line-clamp-1 mb-1.5">{entry.description}</p>
              )}

              {/* Bottom: date + account badge + project */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs text-muted-foreground shrink-0">{fmtDate(entry.date)}</span>
                <span className={cn(
                  'text-[10px] font-medium px-1.5 py-0.5 rounded-full shrink-0',
                  ACCOUNT_COLORS[entry.account] || 'bg-muted text-foreground'
                )}>
                  {getAccountLabel(entry.account)}
                </span>
                {projectName && (
                  <span className="text-xs text-muted-foreground truncate">{projectName}</span>
                )}
                {evidence.hasEvidence && (
                  <span className="shrink-0" onClick={(e) => {
                    e.stopPropagation();
                    if (evidence.type === 'asset' && evidence.asset_id) {
                      downloadAsset.mutate({
                        id: evidence.asset_id,
                        filename: evidence.asset ? assetService.getDisplayName(evidence.asset) : `asset_${evidence.asset_id}`
                      });
                    }
                  }}>
                    {evidence.type === 'url' ? (
                      <ExternalLink className="h-3 w-3 text-blue-500" />
                    ) : (
                      <Download className="h-3 w-3 text-emerald-600" />
                    )}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      {pagination && onPageChange && onPageSizeChange && (
        <PaginationControls
          pagination={pagination}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
          className="border-t pt-3"
        />
      )}
    </div>
  );
}
