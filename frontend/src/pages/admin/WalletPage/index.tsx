import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowRightLeft,
  Loader2,
  RefreshCw,
  TrendingDown,
  Wallet as WalletIcon,
  CheckCircle2,
} from "lucide-react";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import WalletTransactionsList from "@/components/wallet/WalletTransactionsList";
import CreateManualDisbursementDialog from "@/components/wallet/CreateManualDisbursementDialog";
import { walletService } from "@/services/api/wallet.service";
import type { WalletBalance } from "@/types/api/wallet.types";
import { showErrorNotification } from "@/utils/error-handler";
import { formatVietnameseDateTime } from "@/utils/vietnamese";
import { formatCurrency as formatVND } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const BALANCE_QUERY_KEY = ["wallet", "balance"] as const;

/* ------------------------------------------------------------------ */
/*  Watermark stat tile — matches the design language used across the
/*  partner / admin dashboards.                                        */
/* ------------------------------------------------------------------ */
interface StatTileProps {
  label: string;
  value: string;
  hint?: string;
  icon: typeof WalletIcon;
  iconText: string;
  watermark: string;
  isLoading?: boolean;
}

function StatTile({ label, value, hint, icon: Icon, iconText, watermark, isLoading }: StatTileProps) {
  // h-full + vertical center so when stretched alongside the (taller) hero
  // the content sits in the middle of the card instead of clinging to the top.
  return (
    <div className="group relative h-full flex flex-col justify-center rounded-2xl border border-border/60 bg-card p-5 shadow-sm overflow-hidden transition-colors hover:bg-muted/40">
      <Icon
        className={cn(
          "absolute right-4 top-1/2 -translate-y-1/2 h-20 w-20 pointer-events-none",
          "transition-transform duration-300 group-hover:scale-105",
          watermark,
        )}
        strokeWidth={1.5}
      />
      <div className="relative pr-20">
        <div className="flex items-center gap-1.5">
          <Icon className={cn("h-3.5 w-3.5 shrink-0", iconText)} strokeWidth={2.2} />
          <span className="text-[10.5px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight truncate">
            {label}
          </span>
        </div>
        {isLoading ? (
          <div className="mt-2 h-7 w-32 bg-muted rounded animate-pulse" />
        ) : (
          <p className="mt-1.5 font-financial text-[26px] font-bold tabular-nums tracking-[-0.02em] leading-none text-foreground">
            {value}
          </p>
        )}
        {hint && !isLoading && (
          <p className="mt-2 text-[11.5px] text-muted-foreground leading-snug">{hint}</p>
        )}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Hero balance card — primary, takes prominent space.                */
/* ------------------------------------------------------------------ */
interface HeroBalanceProps {
  balance: WalletBalance | undefined;
  asOf: string | null;
  syncing: boolean;
  onSync: (e: React.MouseEvent) => void;
}

function HeroBalance({ balance, asOf, syncing, onSync }: HeroBalanceProps) {
  return (
    <section
      aria-label="Số dư ví"
      className={cn(
        "relative h-full flex flex-col justify-center overflow-hidden rounded-2xl border border-border/60 bg-card p-5 shadow-sm",
        // Subtle amber wash so the hero reads as the headline element.
        "bg-[radial-gradient(140%_90%_at_0%_0%,rgba(245,158,11,0.05)_0%,transparent_55%),linear-gradient(135deg,rgba(248,250,252,1)_0%,rgba(255,255,255,1)_100%)]",
      )}
    >
      {/* Watermark icon — bleeds bottom-right */}
      <WalletIcon
        className="absolute -right-4 -bottom-4 h-40 w-40 text-amber-500/[0.06] pointer-events-none"
        strokeWidth={1.2}
      />

      <div className="relative">
        <div className="flex items-center gap-2 mb-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-amber-500 shadow-[0_0_0_3px_rgba(245,158,11,0.18)]" />
          <span className="text-[10.5px] font-bold uppercase tracking-[0.12em] text-amber-700">
            Số dư khả dụng
          </span>
        </div>
        <p className="font-financial text-[36px] sm:text-[40px] font-bold leading-none tracking-[-0.02em] tabular-nums text-foreground">
          {balance ? formatVND(balance.available) : "—"}
        </p>
        <div className="mt-3 flex items-center gap-2 text-[11.5px] text-muted-foreground">
          <RefreshCw className="h-3 w-3" />
          <span>{asOf ? `Cập nhật ${asOf}` : "Đang tải..."}</span>
          <button
            type="button"
            onClick={onSync}
            disabled={syncing}
            className="ml-1 inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[11px] font-medium text-amber-700 hover:bg-amber-50 transition-colors disabled:opacity-40"
            title="Đồng bộ số dư với nhà cung cấp"
          >
            {syncing ? (
              <>
                <Loader2 className="h-3 w-3 animate-spin" />
                Đang đồng bộ
              </>
            ) : (
              <>Đồng bộ ngay</>
            )}
          </button>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Main page                                                          */
/* ------------------------------------------------------------------ */
export default function WalletPage() {
  const queryClient = useQueryClient();
  const [disbursementOpen, setDisbursementOpen] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [adjusting, setAdjusting] = useState(false);
  const [mismatch, setMismatch] = useState<{ provider: number; local: number } | null>(null);

  const { data: balance, isLoading: balanceLoading } = useQuery<WalletBalance>({
    queryKey: BALANCE_QUERY_KEY,
    queryFn: () => walletService.getBalance(),
  });

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ["wallet"] });
  };

  const asOf = balance?.as_of ? formatVietnameseDateTime(balance.as_of) : null;

  const handleSync = async (e: React.MouseEvent) => {
    e.stopPropagation();
    setSyncing(true);
    try {
      const result = await walletService.syncBalance();
      if (result.provider_balance === result.local_balance) {
        toast.success(`Số dư khớp: ${formatVND(result.local_balance)}`);
        invalidateAll();
      } else {
        setMismatch({ provider: result.provider_balance, local: result.local_balance });
      }
    } catch (err) {
      showErrorNotification(err, "Không thể đồng bộ số dư");
    } finally {
      setSyncing(false);
    }
  };

  const handleAdjust = async () => {
    if (!mismatch) return;
    setAdjusting(true);
    const diff = mismatch.provider - mismatch.local;
    try {
      await walletService.adjustBalance({
        amount: diff,
        reason: `Số dư nhà cung cấp: ${formatVND(mismatch.provider)}, Hệ thống: ${formatVND(mismatch.local)}, Chênh lệch: ${formatVND(diff)}`,
      });
      toast.success("Đã điều chỉnh số dư thành công");
      setMismatch(null);
      invalidateAll();
    } catch (err) {
      showErrorNotification(err, "Điều chỉnh số dư thất bại");
    } finally {
      setAdjusting(false);
    }
  };

  const diff = mismatch ? mismatch.provider - mismatch.local : 0;

  return (
    <div className="min-h-full bg-background">
      <div className="max-w-[1280px] mx-auto px-4 md:px-6 py-5 md:py-6 space-y-5">

        {/* Page header */}
        <header className="flex items-start justify-between gap-3 flex-wrap">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-amber-50 border border-amber-200">
              <WalletIcon className="h-4 w-4 text-amber-600" />
            </div>
            <div>
              <h1 className="text-base font-semibold text-foreground leading-tight tracking-tight">
                Quản lý ví
              </h1>
              <p className="text-xs text-muted-foreground mt-0.5">
                Theo dõi số dư, chuyển khoản và đối soát giao dịch
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleSync}
              disabled={syncing}
              className="gap-1.5 h-9"
            >
              {syncing ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <RefreshCw className="h-3.5 w-3.5" />
              )}
              Đồng bộ
            </Button>
            <Button
              size="sm"
              onClick={() => setDisbursementOpen(true)}
              className="gap-1.5 h-9"
            >
              <ArrowRightLeft className="h-3.5 w-3.5" />
              Chuyển tiền
            </Button>
          </div>
        </header>

        {/* Hero balance + Đang chi trả — same-height cards via grid + h-full */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-stretch">
          <div className="md:col-span-2">
            <HeroBalance balance={balance} asOf={asOf} syncing={syncing} onSync={handleSync} />
          </div>
          <StatTile
            label="Đang chi trả"
            value={balance ? formatVND(balance.pending_out) : "—"}
            hint="Chuyển khoản đang xử lý"
            icon={TrendingDown}
            iconText="text-amber-600"
            watermark="text-amber-500/15"
            isLoading={balanceLoading}
          />
        </div>

        {/* Transactions */}
        <WalletTransactionsList />
      </div>

      {/* Dialogs */}
      <CreateManualDisbursementDialog
        open={disbursementOpen}
        onOpenChange={setDisbursementOpen}
        onSuccess={invalidateAll}
      />

      {/* Mismatch confirmation */}
      <AlertDialog open={!!mismatch} onOpenChange={(open) => !open && setMismatch(null)}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
            <div className="flex items-center gap-2.5 mb-1">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 border border-amber-200">
                <RefreshCw className="h-3.5 w-3.5 text-amber-600" />
              </div>
              <AlertDialogTitle className="text-sm">Phát hiện chênh lệch số dư</AlertDialogTitle>
            </div>
            <AlertDialogDescription asChild>
              <div className="space-y-3">
                <div className="grid grid-cols-3 gap-2">
                  <div className="rounded-lg border border-border/60 bg-muted/40 p-2.5">
                    <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Nhà cung cấp</p>
                    <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-foreground">
                      {formatVND(mismatch?.provider ?? 0)}
                    </p>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/40 p-2.5">
                    <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-semibold">Hệ thống</p>
                    <p className="mt-0.5 font-financial text-sm font-bold tabular-nums text-foreground">
                      {formatVND(mismatch?.local ?? 0)}
                    </p>
                  </div>
                  <div
                    className={cn(
                      "rounded-lg border p-2.5",
                      diff > 0 ? "border-emerald-200 bg-emerald-50" : "border-rose-200 bg-rose-50",
                    )}
                  >
                    <p className="text-[10px] uppercase tracking-wider font-semibold text-muted-foreground">Chênh lệch</p>
                    <p
                      className={cn(
                        "mt-0.5 font-financial text-sm font-bold tabular-nums",
                        diff > 0 ? "text-emerald-700" : "text-rose-700",
                      )}
                    >
                      {diff > 0 ? "+" : ""}
                      {formatVND(diff)}
                    </p>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  Tạo bản ghi điều chỉnh{" "}
                  <strong className={cn("font-semibold", diff > 0 ? "text-emerald-700" : "text-rose-700")}>
                    {diff > 0 ? "+" : ""}
                    {formatVND(diff)}
                  </strong>{" "}
                  để khớp số dư với nhà cung cấp?
                </p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="h-9 text-xs" disabled={adjusting}>
              Bỏ qua
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleAdjust}
              disabled={adjusting}
              className="h-9 text-xs gap-1.5"
            >
              {adjusting ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <CheckCircle2 className="h-3 w-3" />
                  Điều chỉnh
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
