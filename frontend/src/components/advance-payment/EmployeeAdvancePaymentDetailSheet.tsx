import { format } from "date-fns";
import { vi } from "date-fns/locale";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetClose,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Wallet, X } from "lucide-react";
import {
  getVietnameseAdvancePaymentStatus,
  getAdvancePaymentStatusColor,
} from "@/utils/advancePaymentHelpers";
import { formatCurrency } from "@/utils/formatters";
import { useAdvancePayments } from "@/hooks/api/useAdvancePayments";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { EmptyState } from '@/components/shared/EmptyState';
import type {
  AdvancePaymentListItem,
  FlexPayEmployeeListItem,
} from "@/types/api/advance-payment.types";

interface EmployeeAdvancePaymentDetailSheetProps {
  employee: FlexPayEmployeeListItem | null;
  onClose: () => void;
}

/**
 * Side sheet showing advance payment details for a specific flex-pay employee.
 * Displays limit usage, bank info, and request history.
 */
export const EmployeeAdvancePaymentDetailSheet = ({
  employee,
  onClose,
}: EmployeeAdvancePaymentDetailSheetProps) => {
  const isMobile = useIsMobile();
  const { data: employeeRequests } = useAdvancePayments(
    { search: employee?.cccd, pageSize: 50 },
    { enabled: employee !== null },
  );

  return (
    <Sheet
      open={employee !== null}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        className="w-full sm:max-w-[420px] p-0 flex flex-col bg-[#f7f8fa]"
      >
        {employee && (
          <EmployeeDetailContent
            employee={employee}
            requests={(employeeRequests?.data ?? []) as AdvancePaymentListItem[]}
          />
        )}
      </SheetContent>
    </Sheet>
  );
};

// ---------------------------------------------------------------------------
// Internal sub-components
// ---------------------------------------------------------------------------

interface EmployeeDetailContentProps {
  employee: FlexPayEmployeeListItem;
  requests: AdvancePaymentListItem[];
}

function EmployeeDetailContent({ employee: emp, requests }: EmployeeDetailContentProps) {
  // If maxAdvanceAmount is 0, bar is full (100%) to indicate no limit set / fully used
  const usedPct =
    emp.maxAdvanceAmount > 0
      ? Math.min(
          100,
          Math.round(
            ((emp.utilizedAmount + emp.pendingAmount) / emp.maxAdvanceAmount) * 100,
          ),
        )
      : 100;
  const isOverLimit = usedPct >= 100;

  return (
    <>
      {/* ── Header ── */}
      <EmployeeDetailHeader employee={emp} />

      {/* ── Scrollable body ── */}
      <div className="flex-1 overflow-y-auto">
        {/* ── Limit card ── */}
        <div className="px-4 pt-4 pb-2">
          <AdvanceLimitCard
            employee={emp}
            usedPct={usedPct}
            isOverLimit={isOverLimit}
          />
        </div>

        {/* ── Info row ── */}
        <div className="px-4 pt-2 pb-2">
          <InfoCard label="Phí thu" value={formatCurrency(emp.totalFeeGenerated)} />
        </div>

        {/* ── Bank info ── */}
        {emp.bank?.accountNumber && (
          <div className="px-4 pb-2">
            <BankInfoCard
              bankName={emp.bank.bankName}
              accountNumber={emp.bank.accountNumber}
            />
          </div>
        )}

        {/* ── Requests list ── */}
        <div className="px-4 pt-2 pb-6">
          <RequestsList requests={requests} />
        </div>
      </div>
    </>
  );
}

function EmployeeDetailHeader({ employee: emp }: { employee: FlexPayEmployeeListItem }) {
  return (
    <div className="bg-white px-5 pt-5 pb-4 border-b shrink-0" style={{ paddingTop: "max(20px, calc(20px + env(safe-area-inset-top)))" }}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <SheetTitle className="text-[17px] font-bold text-slate-900 leading-tight">
            {emp.fullname}
          </SheetTitle>
          <p className="text-xs font-mono text-slate-500 mt-0.5 tracking-wide">
            {emp.cccd}
          </p>
          <div className="flex items-center gap-1.5 mt-2.5 flex-wrap">
            <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
              {emp.project?.name || "Chưa có dự án"}
            </span>
            {emp.bank?.bankName && (
              <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
                {emp.bank.bankName}
              </span>
            )}
          </div>
        </div>
        <SheetClose asChild>
          <button
            className="shrink-0 mt-0.5 h-7 w-7 rounded-full bg-slate-100 flex items-center justify-center text-slate-500 hover:bg-slate-200 transition-colors"
            aria-label="Đóng"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </SheetClose>
      </div>
    </div>
  );
}

interface AdvanceLimitCardProps {
  employee: FlexPayEmployeeListItem;
  usedPct: number;
  isOverLimit: boolean;
}

function AdvanceLimitCard({ employee: emp, usedPct, isOverLimit }: AdvanceLimitCardProps) {
  const progressColor = isOverLimit
    ? "bg-red-400"
    : usedPct > 70
      ? "bg-amber-400"
      : "bg-emerald-400";

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      {/* Top: remaining amount hero */}
      <div className="px-4 pt-4 pb-3">
        <p className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-1">
          Còn lại có thể ứng
        </p>
        <p
          className={`text-2xl font-bold tabular-nums ${
            emp.availableAmount > 0 ? "text-slate-900" : "text-slate-500"
          }`}
        >
          {formatCurrency(emp.availableAmount)}
        </p>
      </div>

      {/* Progress bar */}
      <div className="px-4 pb-3">
        <div className="h-2 bg-slate-100 rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-500 ${progressColor}`}
            style={{ width: `${usedPct}%` }}
            role="progressbar"
            aria-valuenow={usedPct}
            aria-valuemin={0}
            aria-valuemax={100}
          />
        </div>
        <div className="flex items-center justify-between mt-1.5">
          <span className="text-xs text-slate-500">Đã sử dụng</span>
          <span
            className={`text-xs font-semibold tabular-nums ${
              isOverLimit ? "text-red-600" : "text-slate-600"
            }`}
          >
            {usedPct}%
          </span>
        </div>
      </div>

      {/* Divider */}
      <div className="border-t border-slate-200 mx-4" />

      {/* Limit breakdown */}
      <div className="grid grid-cols-1 divide-y divide-slate-200 sm:grid-cols-3 sm:divide-x sm:divide-y-0">
        <LimitBreakdownCell label="Hạn mức" value={formatCurrency(emp.maxAdvanceAmount)} />
        <LimitBreakdownCell
          label="Đã dùng"
          value={formatCurrency(emp.utilizedAmount)}
          valueClassName="text-orange-700"
        />
        <LimitBreakdownCell
          label="Chờ xử lý"
          value={formatCurrency(emp.pendingAmount)}
          valueClassName="text-amber-700"
        />
      </div>
    </div>
  );
}

function LimitBreakdownCell({
  label,
  value,
  valueClassName = "text-slate-700",
}: {
  label: string;
  value: string;
  valueClassName?: string;
}) {
  return (
    <div className="px-3 py-3 text-center">
      <p className="text-xs text-slate-500 mb-1">{label}</p>
      <p className={`text-xs font-bold tabular-nums ${valueClassName}`}>{value}</p>
    </div>
  );
}

function InfoCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm px-3 py-2.5">
      <p className="text-xs text-slate-500 mb-0.5">{label}</p>
      <p className="text-sm font-bold text-slate-800 tabular-nums">{value}</p>
    </div>
  );
}

function BankInfoCard({
  bankName,
  accountNumber,
}: {
  bankName: string | null;
  accountNumber: string;
}) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm px-3 py-2.5 flex items-center justify-between">
      <div>
        <p className="text-xs text-slate-500 mb-0.5">Ngân hàng</p>
        <p className="text-sm font-semibold text-slate-800">{bankName || "—"}</p>
      </div>
      <p className="text-xs font-mono text-slate-500 tabular-nums">{accountNumber}</p>
    </div>
  );
}

function RequestsList({ requests }: { requests: AdvancePaymentListItem[] }) {
  return (
    <>
      <div className="flex items-center justify-between mb-2.5">
        <p className="text-xs font-semibold text-slate-700">Yêu cầu ứng lương</p>
        {requests.length > 0 && (
          <span className="text-xs text-slate-500 tabular-nums">
            {requests.length} yêu cầu
          </span>
        )}
      </div>

      {requests.length === 0 ? (
        <div className="bg-white rounded-xl border border-slate-200 px-3">
          <EmptyState title="Chưa có yêu cầu nào" size="sm" className="py-4" />
        </div>
      ) : (
        <div className="space-y-2">
          {requests.map((req) => (
            <RequestCard key={req.id} request={req} />
          ))}
        </div>
      )}
    </>
  );
}

function RequestCard({ request: req }: { request: AdvancePaymentListItem }) {
  const statusColor = getAdvancePaymentStatusColor(req.status);
  const leftBarColor =
    req.status === "COMPLETED" || req.status === "APPROVED"
      ? "bg-emerald-400"
      : req.status === "PENDING"
        ? "bg-amber-400"
        : req.status === "FAILED"
          ? "bg-red-400"
          : "bg-slate-300";

  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden flex">
      <div className={`w-1 shrink-0 ${leftBarColor}`} aria-hidden="true" />
      <div className="flex-1 min-w-0 px-3 py-2.5">
        <div className="flex items-center justify-between gap-2 mb-1">
          <span className="text-sm font-bold text-slate-900 tabular-nums">
            {formatCurrency(req.requestAmount)}
          </span>
          <Badge variant="outline" className={`${statusColor} text-xs px-1.5 py-0 h-4 shrink-0`}>
            {getVietnameseAdvancePaymentStatus(req.status)}
          </Badge>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-500 flex-wrap">
          <span className="tabular-nums">
            {format(new Date(req.createdAt), "dd/MM/yyyy HH:mm", { locale: vi })}
          </span>
          <span className="text-slate-200" aria-hidden="true">·</span>
          <span>
            Nhận{" "}
            <span className="font-medium text-slate-600 tabular-nums">
              {formatCurrency(req.netAmount)}
            </span>
          </span>
          <span className="text-slate-200" aria-hidden="true">·</span>
          <span>
            Phí <span className="tabular-nums">{formatCurrency(req.fee)}</span>
          </span>
        </div>
      </div>
    </div>
  );
}
