import { useState, useMemo } from "react";
import {
  Search,
  ChevronLeft,
  ChevronRight,
  ChevronRightIcon,
  Building2,
  CreditCard,
  User,
  Phone,
  Mail,
  BadgeCheck,
} from "lucide-react";
import { useFlexPayEmployeesInfinite } from "@/hooks/api/useAdvancePayments";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { FlexPayEmployeeListItem } from "@/types/api/advance-payment.types";
import EditAdvPartnerUserSheet from "./EditAdvPartnerUserSheet";

// ─── Avatar helpers ───────────────────────────────────────────────────────────

const AVATAR_GRADIENTS = [
  "from-violet-500 to-purple-600",
  "from-sky-500 to-blue-600",
  "from-emerald-500 to-green-600",
  "from-amber-500 to-orange-600",
  "from-rose-500 to-red-600",
  "from-indigo-500 to-blue-700",
];

function getInitials(name: string): string {
  return (
    name
      .split(" ")
      .filter(Boolean)
      .slice(-2)
      .map((w) => w[0])
      .join("")
      .toUpperCase() || "?"
  );
}

function getGradient(name: string): string {
  return AVATAR_GRADIENTS[name.charCodeAt(0) % AVATAR_GRADIENTS.length];
}

// ─── Skeletons ────────────────────────────────────────────────────────────────

function RowSkeleton() {
  return (
    <tr className="border-b border-border/30">
      {Array.from({ length: 10 }).map((_, i) => (
        <td key={i} className="px-4 py-3.5">
          <Skeleton className="h-4 w-full rounded" />
        </td>
      ))}
    </tr>
  );
}

function CardSkeleton() {
  return (
    <div className="bg-card rounded-xl border border-border/50 overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3.5">
        <Skeleton className="h-9 w-9 rounded-xl shrink-0" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-3 w-20" />
        </div>
        <Skeleton className="h-8 w-8 rounded-lg" />
      </div>
      <div className="border-t border-border/40 bg-muted/20 px-4 py-2.5 flex gap-3">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-3 w-24" />
      </div>
    </div>
  );
}

// ─── Empty state ──────────────────────────────────────────────────────────────

function EmptyState({ hasSearch }: { hasSearch: boolean }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 text-center">
      <div className="w-14 h-14 rounded-2xl bg-muted/60 flex items-center justify-center mb-4">
        <User className="w-7 h-7 text-muted-foreground/50" />
      </div>
      <p className="font-semibold text-base">Không tìm thấy nhân viên nào</p>
      <p className="text-sm text-muted-foreground mt-1 max-w-xs">
        {hasSearch
          ? "Không có kết quả phù hợp với từ khoá tìm kiếm."
          : "Chưa có nhân viên FlexPay nào."}
      </p>
    </div>
  );
}

// ─── Desktop table helpers ────────────────────────────────────────────────────

function ColHead({
  children,
  icon: Icon,
}: {
  children: React.ReactNode;
  icon?: React.ElementType;
}) {
  return (
    <th className="px-4 py-3 text-left text-[11px] font-bold uppercase tracking-wider text-muted-foreground bg-muted/30 whitespace-nowrap">
      <span className="inline-flex items-center gap-1.5">
        {Icon && <Icon className="w-3 h-3 opacity-60" />}
        {children}
      </span>
    </th>
  );
}

// ─── Mobile card ──────────────────────────────────────────────────────────────

function EmployeeCard({
  emp,
  onEdit,
}: {
  emp: FlexPayEmployeeListItem;
  onEdit: () => void;
}) {
  const hasBank = !!emp.bank;
  const initials = getInitials(emp.fullname);
  const gradient = getGradient(emp.fullname);

  return (
    <button
      type="button"
      onClick={onEdit}
      className="w-full text-left bg-card rounded-xl border border-border/50 shadow-sm overflow-hidden active:scale-[0.99] transition-transform cursor-pointer"
    >
      <div className="flex items-center gap-3 px-4 py-3.5">
        <div
          className={cn(
            "h-9 w-9 rounded-xl flex items-center justify-center shrink-0 text-white text-[12px] font-bold bg-gradient-to-br select-none",
            gradient
          )}
          aria-hidden
        >
          {initials}
        </div>

        <div className="flex-1 min-w-0">
          <div className="font-semibold text-[14px] text-foreground leading-tight truncate">
            {emp.fullname}
          </div>
          {emp.username ? (
            <div className="font-mono text-[11px] text-muted-foreground/80 mt-0.5">
              @{emp.username}
            </div>
          ) : (
            <div className="text-[11px] text-muted-foreground/40 italic mt-0.5">
              chưa có tài khoản
            </div>
          )}
        </div>

        <ChevronRightIcon className="w-4 h-4 text-muted-foreground/40 shrink-0" />
      </div>

      {/* Info strip — project · bank · account */}
      <div className="border-t border-border/40 bg-muted/25 px-4 py-2 flex items-center gap-0 flex-wrap min-h-[34px]">
        {/* Project */}
        <span className="inline-flex items-center gap-1 text-[11px] font-medium text-emerald-700 dark:text-emerald-400 pr-2.5">
          <Building2 className="w-3 h-3 shrink-0 text-emerald-600/80" />
          {emp.project?.name || "—"}
        </span>

        {/* Bank */}
        {hasBank ? (
          <>
            <span className="text-border mr-2.5" aria-hidden>·</span>
            <span className="inline-flex items-center gap-1 text-[11px] font-medium text-blue-700 dark:text-blue-400 pr-2.5">
              <CreditCard className="w-3 h-3 shrink-0 text-blue-500/80" />
              {emp.bank!.bankName}
            </span>
          </>
        ) : (
          <>
            <span className="text-border mr-2.5" aria-hidden>·</span>
            <span className="text-[11px] text-muted-foreground/50 italic">
              Chưa có ngân hàng
            </span>
          </>
        )}

        {/* Account number */}
        {emp.bank?.accountNumber && (
          <>
            <span className="text-border mr-2.5" aria-hidden>·</span>
            <span className="font-mono text-[11px] text-muted-foreground/70 tabular-nums">
              {emp.bank.accountNumber}
            </span>
          </>
        )}
      </div>
    </button>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────
const AdvPartnerUsersPage = () => {
  const [search, setSearch] = useState("");
  const [editEmployee, setEditEmployee] = useState<FlexPayEmployeeListItem | null>(null);

  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useFlexPayEmployeesInfinite({
      pageSize: 50,
      search: search || undefined,
    });

  const employees = useMemo(() => {
    if (!data?.pages) return [];
    const all = data.pages.flatMap((p) => (p.data as FlexPayEmployeeListItem[]) || []);
    const seen = new Set<number>();
    return all.filter((emp) => {
      if (seen.has(emp.employeeId)) return false;
      seen.add(emp.employeeId);
      return true;
    });
  }, [data?.pages]);

  // ── Loading ──────────────────────────────────────────────────────────────
  if (isLoading && employees.length === 0) {
    return (
      <div className="p-4 lg:p-6 space-y-5 max-w-[1400px] mx-auto">
        <div className="space-y-2">
          <Skeleton className="h-8 w-52" />
          <Skeleton className="h-4 w-80" />
        </div>

        <div className="flex flex-col gap-3 sm:hidden">
          {Array.from({ length: 5 }).map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>

        <div className="hidden sm:block bg-card rounded-2xl border border-border/40 shadow-sm overflow-hidden">
          <div className="px-4 py-3 border-b border-border/40">
            <Skeleton className="h-9 w-72 rounded-[10px]" />
          </div>
          <table className="w-full">
            <thead>
              <tr>
                {Array.from({ length: 10 }).map((_, i) => (
                  <th key={i} className="px-4 py-3 bg-muted/30">
                    <Skeleton className="h-3 w-16" />
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {Array.from({ length: 6 }).map((_, i) => (
                <RowSkeleton key={i} />
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }

  // ── Pagination footer (shared) ────────────────────────────────────────────
  const PaginationFooter = () => (
    <div className="flex items-center justify-between px-4 py-3 border-t border-border/30 bg-muted/10 text-[13px] text-muted-foreground">
      <span>
        Hiển thị{" "}
        <strong className="font-mono text-foreground">{employees.length}</strong>{" "}
        nhân viên
      </span>
      <div className="flex items-center gap-1">
        <button
          disabled
          className="w-8 h-8 rounded-lg grid place-items-center hover:bg-muted disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
        <button className="w-8 h-8 rounded-lg grid place-items-center font-mono font-semibold bg-foreground text-white text-[12px]">
          1
        </button>
        {hasNextPage && (
          <button
            onClick={() => fetchNextPage()}
            className="w-8 h-8 rounded-lg grid place-items-center font-mono text-muted-foreground hover:bg-muted transition-colors text-[12px]"
          >
            2
          </button>
        )}
        <button
          disabled={!hasNextPage}
          onClick={() => fetchNextPage()}
          className="w-8 h-8 rounded-lg grid place-items-center hover:bg-muted disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 space-y-4 max-w-[1400px] mx-auto">

        {/* ── Header ─────────────────────────────────────────────────────── */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h1 className="text-[20px] sm:text-2xl font-bold tracking-tight text-foreground leading-tight">
              Nhân viên
            </h1>
            <p className="text-[13px] text-muted-foreground mt-0.5 leading-snug">
              Thông tin, ngân hàng và dự án
            </p>
          </div>
          {employees.length > 0 && (
            <div className="shrink-0 mt-0.5 px-2.5 py-1 rounded-lg bg-muted/60 text-[12px] font-semibold text-muted-foreground tabular-nums">
              {employees.length}
            </div>
          )}
        </div>

        {/* ── Search bar ──────────────────────────────────────────────────── */}
        <div className="relative w-full sm:max-w-[380px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground/60 pointer-events-none" />
          <input
            type="text"
            placeholder="Tìm theo tên, CCCD..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 h-10 rounded-xl border border-border/70 bg-card text-[13px] text-foreground placeholder:text-muted-foreground/50 outline-none focus:border-foreground/30 focus:ring-4 focus:ring-foreground/5 transition-all shadow-sm"
          />
        </div>

        {/* ── Mobile card list ────────────────────────────────────────────── */}
        <div className="flex flex-col gap-3 sm:hidden">
          {employees.length === 0 && !isLoading ? (
            <EmptyState hasSearch={!!search} />
          ) : (
            <>
              {employees.map((emp) => (
                <EmployeeCard
                  key={emp.employeeId}
                  emp={emp}
                  onEdit={() => setEditEmployee(emp)}
                />
              ))}

              {hasNextPage && (
                <button
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="w-full py-3 rounded-xl border border-border/50 bg-card text-[13px] font-medium text-muted-foreground hover:bg-muted/40 disabled:opacity-50 transition-colors flex items-center justify-center gap-2"
                >
                  {isFetchingNextPage ? (
                    <>
                      <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-muted-foreground border-t-transparent" />
                      Đang tải...
                    </>
                  ) : (
                    "Tải thêm nhân viên"
                  )}
                </button>
              )}

              {employees.length > 0 && !hasNextPage && (
                <p className="text-center text-[11px] text-muted-foreground/60 pb-1">
                  {employees.length} nhân viên
                </p>
              )}
            </>
          )}
        </div>

        {/* ── Desktop table ───────────────────────────────────────────────── */}
        <div className="hidden sm:block bg-card rounded-2xl border border-border/40 shadow-sm overflow-hidden">
          <div className="px-4 pt-3 pb-1">
            <p className="text-[11px] text-muted-foreground/50">Nhấn vào hàng để xem chi tiết</p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px]">
              <thead>
                <tr>
                  <th className="w-10 px-4 py-3 text-left text-[11px] font-bold uppercase tracking-wider text-muted-foreground bg-muted/30">
                    #
                  </th>
                  <ColHead icon={User}>Nhân viên</ColHead>
                  <ColHead icon={Mail}>Email</ColHead>
                  <ColHead icon={Building2}>Dự án</ColHead>
                  <ColHead icon={CreditCard}>Ngân hàng</ColHead>
                  <ColHead icon={CreditCard}>Số tài khoản</ColHead>
                  <ColHead icon={BadgeCheck}>CCCD</ColHead>
                </tr>
              </thead>
              <tbody>
                {employees.map((emp, i) => (
                  <tr
                    key={emp.employeeId}
                    onClick={() => setEditEmployee(emp)}
                    className="border-b border-border/30 hover:bg-muted/25 transition-colors cursor-pointer group"
                  >
                    <td className="px-4 py-3.5 font-mono text-xs text-muted-foreground">
                      {String(i + 1).padStart(2, "0")}
                    </td>
                    <td className="px-4 py-3.5 min-w-[160px]">
                      <div className="flex items-center gap-2.5">
                        <div
                          className={cn(
                            "h-7 w-7 rounded-lg flex items-center justify-center shrink-0 text-white text-[10px] font-bold bg-gradient-to-br select-none",
                            getGradient(emp.fullname)
                          )}
                          aria-hidden
                        >
                          {getInitials(emp.fullname)}
                        </div>
                        <div className="min-w-0">
                          <div className="font-semibold text-[13px] text-foreground truncate leading-tight">
                            {emp.fullname}
                          </div>
                          {emp.username ? (
                            <div className="font-mono text-[11px] text-muted-foreground">
                              @{emp.username}
                            </div>
                          ) : (
                            <div className="text-[11px] text-muted-foreground/50 italic">
                              chưa có tài khoản
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3.5 min-w-[140px]">
                      <div className="text-[12px] text-foreground/80 truncate">
                        {emp.email || "—"}
                      </div>
                    </td>
                    <td className="px-4 py-3.5">
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-foreground/80 font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                        {emp.project?.name || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3.5">
                      <span
                        className={cn(
                          "text-[12px]",
                          emp.bank
                            ? "text-foreground/80"
                            : "text-muted-foreground/60 italic"
                        )}
                      >
                        {emp.bank?.bankName || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3.5">
                      <span
                        className={cn(
                          "font-mono text-[12px]",
                          emp.bank?.accountNumber
                            ? "text-foreground/80"
                            : "text-muted-foreground/60 italic"
                        )}
                      >
                        {emp.bank?.accountNumber || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3.5">
                      <span className="font-mono text-[12px] text-foreground/80">
                        {emp.cccd || "—"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {employees.length === 0 && !isLoading && (
            <EmptyState hasSearch={!!search} />
          )}

          {employees.length > 0 && <PaginationFooter />}
        </div>

        {isFetchingNextPage && (
          <div className="hidden sm:flex items-center justify-center py-4 gap-2 text-sm text-muted-foreground">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-foreground border-t-transparent" />
            Đang tải thêm...
          </div>
        )}
      </div>

      {/* Edit sheet */}
      {editEmployee && (
        <EditAdvPartnerUserSheet
          employeeId={editEmployee.employeeId}
          fullname={editEmployee.fullname}
          username={editEmployee.username || undefined}
          onClose={() => setEditEmployee(null)}
        />
      )}
    </div>
  );
};

export default AdvPartnerUsersPage;
