import { ResponsiveTable } from '@/components/ui/responsive-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Plus, Undo2, Download, FileText, ExternalLink, Calendar, Building } from 'lucide-react';
import { ColumnDef } from '@tanstack/react-table';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { useIsMobile } from '@/hooks/use-mobile';
import { ledgerService } from '@/services/api/ledger.service';
import { assetService } from '@/services/api/asset.service';
import { useDownloadAsset } from '@/hooks/api/useAssets';
import type { LedgerEntry, AccountMetadata } from '@/types/api/financial.types';
import type { MobileField, RowAction } from '@/components/ui/mobile-table';
import type { Project } from '@/types/api/project.types';

interface LedgerEntriesTableProps {
  entries: LedgerEntry[];
  isLoading?: boolean;
  onReverse: (entry: LedgerEntry) => void;
  onAddEntry?: () => void;
  onRowClick?: (entry: LedgerEntry) => void;
  projects?: Project[];
  accountMetadata?: AccountMetadata[];
  currentPage?: number;
  pageSize?: number;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}

export function LedgerEntriesTable({
  entries,
  onReverse,
  onAddEntry,
  onRowClick,
  projects = [],
  accountMetadata = [],
  currentPage = 1,
  pageSize = 50,
  pagination,
  onPageChange,
  onPageSizeChange,
}: LedgerEntriesTableProps) {
  const isMobile = useIsMobile();
  const downloadAsset = useDownloadAsset();

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'dd/MM/yyyy', { locale: vi });
    } catch {
      return dateString;
    }
  };

  const formatCurrency = (amount: number) => {
    if (amount === 0) return '0 đ';
    return ledgerService.formatCurrency(amount);
  };

  const getProjectName = (projectId?: number): string | undefined => {
    if (!projectId) return undefined;
    return projects.find(p => p.id === projectId)?.name;
  };

  const getAccountBadge = (account: string) => {
    const variants = {
      cash: 'bg-emerald-100 text-emerald-800',
      receivable: 'bg-blue-100 text-blue-800',
      payable: 'bg-red-100 text-red-800',
      revenue: 'bg-purple-100 text-purple-800',
      expense: 'bg-orange-100 text-orange-800',
    };

    // Use backend label only - no fallback
    const meta = accountMetadata?.find(m => m.value === account);
    if (!meta) {
      console.error(`Missing metadata for account: ${account}`);
      return (
        <Badge className="bg-red-100 text-red-800">
          ERROR: {account}
        </Badge>
      );
    }

    return (
      <Badge className={variants[account as keyof typeof variants] || 'bg-slate-100 text-slate-800'}>
        {meta.label}
      </Badge>
    );
  };




  const handleAssetDownload = (asset: LedgerEntry['asset'], assetId?: number) => {
    if (!assetId) return;

    downloadAsset.mutate({
      id: assetId,
      filename: asset ? assetService.getDisplayName(asset) : `asset_${assetId}`
    });
  };

  const renderEvidenceFiles = (entry: LedgerEntry) => {
    const evidence = ledgerService.getEvidenceDisplay(entry);

    if (!evidence.hasEvidence) {
      return <span className="typography-caption text-slate-400">-</span>;
    }

    return (
      <div className="flex items-center gap-2">
        {evidence.type === 'url' && evidence.url && (
          <div className="flex items-center gap-1">
            <ExternalLink className="w-3 h-3 text-blue-600" />
            <span className="typography-caption text-blue-600">URL</span>
          </div>
        )}
        {evidence.type === 'asset' && evidence.asset_id && evidence.asset_id > 0 && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleAssetDownload(evidence.asset, evidence.asset_id);
            }}
            className="flex items-center gap-1 hover:bg-muted rounded px-1 py-0.5 transition-colors"
            title={`Tải xuống: ${evidence.asset ? assetService.getDisplayName(evidence.asset) : `File ${evidence.asset_id}`}`}
          >
            <Download className="w-3 h-3 text-green-600" />
            <span className="typography-caption text-green-600">
              {evidence.asset ? assetService.getFileIcon(evidence.asset) : '📎'} File
            </span>
          </button>
        )}
        {evidence.type === 'both' && (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1">
              <ExternalLink className="w-3 h-3 text-blue-600" />
              <span className="typography-caption text-blue-600">URL</span>
            </div>
            {evidence.asset_id && evidence.asset_id > 0 && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleAssetDownload(evidence.asset, evidence.asset_id);
                }}
                className="flex items-center gap-1 hover:bg-muted rounded px-1 py-0.5 transition-colors"
                title={`Tải xuống: ${evidence.asset ? assetService.getDisplayName(evidence.asset) : `File ${evidence.asset_id}`}`}
              >
                <Download className="w-3 h-3 text-green-600" />
                <span className="typography-caption text-green-600">
                  {evidence.asset ? assetService.getFileIcon(evidence.asset) : '📎'} File
                </span>
              </button>
            )}
          </div>
        )}
      </div>
    );
  };


  const columns: ColumnDef<LedgerEntry>[] = [
    {
      id: 'stt',
      header: 'STT',
      size: 48,
      maxSize: 48,
      cell: ({ row }) => {
        const index = row.index;
        const page = pagination?.page || currentPage;
        const size = pagination?.pageSize || pageSize;
        const stt = (page - 1) * size + index + 1;
        return (
          <span className="text-xs text-muted-foreground tabular-nums text-center block">
            {stt}
          </span>
        );
      },
    },
    {
      accessorKey: 'date',
      header: 'Ngày',
      size: 90,
      maxSize: 90,
      cell: ({ row }) => (
        <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap">
          {formatDate(row.original.date)}
        </span>
      ),
    },
    {
      accessorKey: 'debit',
      header: () => <div className="text-right">Ghi Nợ</div>,
      size: 110,
      cell: ({ row }) => (
        <div className="text-right">
          {row.original.debit > 0 ? (
            <span className="text-sm font-semibold tabular-nums text-red-600">
              {formatCurrency(row.original.debit)}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground/40">—</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: 'credit',
      header: () => <div className="text-right">Ghi Có</div>,
      size: 110,
      cell: ({ row }) => (
        <div className="text-right">
          {row.original.credit > 0 ? (
            <span className="text-sm font-semibold tabular-nums text-emerald-600">
              {formatCurrency(row.original.credit)}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground/40">—</span>
          )}
        </div>
      ),
    },
    {
      accessorKey: 'net_amount',
      header: () => <div className="text-right">Số Dư</div>,
      size: 120,
      cell: ({ row }) => {
        const account = row.original.account;
        const amount = row.original.net_amount;
        let colorClass = 'text-foreground';
        if (account === 'cash' || account === 'receivable' || account === 'revenue') {
          colorClass = amount > 0 ? 'text-emerald-600' : amount < 0 ? 'text-red-600' : 'text-muted-foreground';
        } else if (account === 'expense') {
          colorClass = amount > 0 ? 'text-orange-600' : 'text-muted-foreground';
        } else if (account === 'payable') {
          colorClass = amount > 0 ? 'text-red-600' : amount < 0 ? 'text-emerald-600' : 'text-muted-foreground';
        }
        return (
          <div className="text-right">
            <span className={`text-sm font-semibold tabular-nums ${colorClass}`}>
              {formatCurrency(amount)}
            </span>
          </div>
        );
      },
    },
    {
      accessorKey: 'account',
      header: 'Loại TK',
      size: 100,
      maxSize: 120,
      cell: ({ row }) => getAccountBadge(row.original.account),
    },
    {
      accessorKey: 'party',
      header: 'Đối Tượng',
      size: 130,
      maxSize: 180,
      cell: ({ row }) => {
        const projectName = getProjectName(row.original.project_id);
        return (
          <div>
            <div className="text-sm text-foreground">{row.original.party}</div>
            {projectName && (
              <div className="text-xs text-muted-foreground mt-0.5">Dự án: {projectName}</div>
            )}
          </div>
        );
      },
    },
    {
      accessorKey: 'description',
      header: 'Diễn Giải',
      size: 180,
      maxSize: 220,
      cell: ({ row }) => (
        <div>
          <div className="text-sm text-foreground line-clamp-2 leading-snug" title={row.original.description}>
            {row.original.description}
          </div>
          {row.original.reference && (
            <div className="text-xs text-amber-600 italic mt-0.5 truncate" title={row.original.reference}>
              {row.original.reference}
            </div>
          )}
        </div>
      ),
    },
    {
      id: 'evidence',
      header: 'Chứng từ',
      size: 80,
      cell: ({ row }) => renderEvidenceFiles(row.original),
    },
  ];

  // Mobile fields configuration - optimized for better information display
  const mobileFields: MobileField<LedgerEntry>[] = [
    {
      label: 'Ngày',
      key: 'date',
      priority: 1,
      render: (entry) => (
        <div className="flex items-center gap-1.5">
          <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="typography-body-small font-medium text-foreground">{formatDate(entry.date)}</span>
        </div>
      ),
    },
    {
      label: 'Loại TK',
      key: 'account',
      priority: 1,
      render: (entry) => getAccountBadge(entry.account),
    },
    {
      label: 'Dự án',
      key: 'project',
      priority: 2,
      render: (entry) => {
        const projectName = getProjectName(entry.project_id);
        if (!projectName) return <span className="text-slate-400 text-xs">-</span>;
        return (
          <div className="flex items-center gap-1.5">
            <Building className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="typography-body-small text-foreground line-clamp-1">{projectName}</span>
          </div>
        );
      },
    },
    {
      label: 'Số tiền',
      key: 'amount',
      priority: 1,
      render: (entry) => {
        const account = entry.account;
        const amount = entry.net_amount;

        // Apply intuitive color scheme
        let colorClass = 'text-foreground';
        let bgClass = '';

        if (account === 'cash' || account === 'receivable' || account === 'revenue') {
          if (amount > 0) {
            colorClass = 'text-emerald-700';
            bgClass = 'bg-emerald-50';
          } else if (amount < 0) {
            colorClass = 'text-red-600';
            bgClass = 'bg-red-50';
          }
        } else if (account === 'expense') {
          if (amount > 0) {
            colorClass = 'text-orange-600';
            bgClass = 'bg-orange-50';
          } else if (amount < 0) {
            colorClass = 'text-muted-foreground';
          }
        } else if (account === 'payable') {
          if (amount > 0) {
            colorClass = 'text-red-600';
            bgClass = 'bg-red-50';
          } else if (amount < 0) {
            colorClass = 'text-emerald-600';
            bgClass = 'bg-emerald-50';
          }
        }

        return (
          <div className="space-y-1.5">
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div className={entry.debit > 0 ? 'text-red-600' : 'text-slate-400'}>
                <div className="font-medium">Nợ</div>
                <div className="font-semibold">{formatCurrency(entry.debit)}</div>
              </div>
              <div className={entry.credit > 0 ? 'text-emerald-600' : 'text-slate-400'}>
                <div className="font-medium">Có</div>
                <div className="font-semibold">{formatCurrency(entry.credit)}</div>
              </div>
            </div>
            <div className={`typography-data-small font-bold ${colorClass} ${bgClass} px-2 py-1 rounded text-center`}>
              Số dư: {formatCurrency(entry.net_amount)}
            </div>
          </div>
        );
      },
    },
    {
      label: 'Diễn giải',
      key: 'description',
      priority: 2,
      render: (entry) => (
        <div className="space-y-0.5">
          <div className="typography-body-small text-foreground line-clamp-2 leading-tight">
            {entry.description}
          </div>
          {entry.reference && (
            <div className="typography-caption text-orange-600 italic line-clamp-1">
              {entry.reference}
            </div>
          )}
        </div>
      ),
    },
    {
      label: 'Chứng từ',
      key: 'evidence',
      priority: 2,
      render: (entry) => renderEvidenceFiles(entry),
    },
  ];

  const rowActions: RowAction<LedgerEntry>[] = [
    {
      label: 'Đảo Ngược',
      icon: <Undo2 className="w-4 h-4" />,
      onClick: onReverse,
      variant: 'default' as const,
    },
  ];

  return (
    <ResponsiveTable
        data={entries}
        columns={columns}
        mobileFields={mobileFields}
        rowTitle={(entry) => {
          const account = entry.account;
          const amount = entry.net_amount;

          // Apply intuitive color scheme
          let amountColorClass = 'text-foreground';

          if (account === 'cash' || account === 'receivable' || account === 'revenue') {
            if (amount > 0) {
              amountColorClass = 'text-emerald-700';
            } else if (amount < 0) {
              amountColorClass = 'text-red-600';
            }
          } else if (account === 'expense') {
            if (amount > 0) {
              amountColorClass = 'text-orange-600';
            } else if (amount < 0) {
              amountColorClass = 'text-muted-foreground';
            }
          } else if (account === 'payable') {
            if (amount > 0) {
              amountColorClass = 'text-red-600';
            } else if (amount < 0) {
              amountColorClass = 'text-emerald-600';
            }
          }

          return (
            <div className="flex items-start justify-between gap-2 min-w-0">
              <div className="flex-1 min-w-0">
                <div className="typography-body-medium font-semibold text-foreground truncate">{entry.party}</div>
                <div className="typography-caption text-muted-foreground mt-0.5">{formatDate(entry.date)}</div>
              </div>
              <div className={`typography-data-medium font-bold ${amountColorClass} flex-shrink-0`}>
                {formatCurrency(Math.max(entry.debit, entry.credit))}
              </div>
            </div>
          );
        }}
        rowSubtitle={(entry) => (
          <div className="flex items-center gap-2 mt-1">
            {getAccountBadge(entry.account)}
            <div className="typography-body-small text-muted-foreground flex-1 truncate">
              {entry.description}
            </div>
          </div>
        )}
        rowActions={rowActions}
        getRowId={(entry) => entry.id.toString()}
        onRowClick={onRowClick}
        pagination={pagination}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
        emptyState={
          <div className="text-center py-12">
            <div className="typography-headline-medium text-foreground mb-2">Không tìm thấy bút toán nào</div>
            <div className="typography-body-medium text-muted-foreground mb-6">
              Thử điều chỉnh bộ lọc của bạn hoặc thêm bút toán mới để bắt đầu.
            </div>
            {onAddEntry && !isMobile && (
              <Button
                onClick={onAddEntry}
                className="flex items-center gap-2 bg-primary-600 hover:bg-primary-700 text-white font-medium"
              >
                <Plus className="h-4 w-4" />
                Thêm Bút Toán Mới
              </Button>
            )}
          </div>
        }
      />
  );
}
