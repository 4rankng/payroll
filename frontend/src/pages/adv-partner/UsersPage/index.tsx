import { useState, useMemo } from "react";
import {
  Building02,
  CheckVerified01,
  ChevronRight,
  CreditCard01,
  Mail01,
  SearchLg,
  User01,
} from "@untitledui/icons";
import { useFlexPayEmployeesInfinite } from "@/hooks/api/useAdvancePayments";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ErrorState } from "@/components/ui/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState as SharedEmptyState } from "@/components/shared/EmptyState";
import { cn } from "@/lib/utils";
import type { FlexPayEmployeeListItem } from "@/types/api/advance-payment.types";
import EditAdvPartnerUserSheet from "./EditAdvPartnerUserSheet";

// ─── Avatar helpers ───────────────────────────────────────────────────────────
// Decorative per-person identity palette (not chrome) — kept verbatim so the
// established directory look survives the token migration.

const AVATAR_GRADIENTS = [
  "from-teal-500 to-cyan-600",
  "from-sky-500 to-blue-600",
  "from-emerald-500 to-green-600",
  "from-amber-500 to-orange-600",
  "from-rose-500 to-red-600",
  "from-lime-500 to-emerald-600",
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
    <tr className="border-b border-utility-gray-100">
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
    <div className="bg-card rounded-lg border border-utility-gray-200 overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3.5">
        <Skeleton className="h-9 w-9 rounded-xl shrink-0" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-3 w-20" />
        </div>
        <Skeleton className="h-8 w-8 rounded-lg" />
      </div>
      <div className="border-t border-utility-gray-100 bg-utility-gray-25 px-4 py-2.5 flex gap-3">
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
    <SharedEmptyState
      title="Không tìm thấy nhân viên nào"
      description={hasSearch
        ? "Không có kết quả phù hợp với từ khoá tìm kiếm."
        : "Chưa có nhân viên FlexPay nào."}
      size="sm"
    />
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
    <th className="px-4 py-3 text-left text-xs font-semibold text-fg-tertiary bg-utility-gray-50 whitespace-nowrap">
      <span className="inline-flex items-center gap-1.5">
        {Icon && <Icon className="size-3.5 text-fg-quaternary" aria-hidden="true" />}
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
      aria-label={`Chỉnh sửa nhân viên ${emp.fullname}`}
      className="w-full text-left bg-card rounded-lg border border-utility-gray-200 shadow-xs overflow-hidden active:scale-[0.99] transition-transform cursor-pointer outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2"
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
          <div className="break-words font-semibold text-[14px] text-fg-primary leading-snug line-clamp-2">
            {emp.fullname}
          </div>
          {emp.username ? (
            <div className="break-all font-mono text-xs text-fg-tertiary mt-0.5">
              @{emp.username}
            </div>
          ) : (
            <div className="text-xs text-fg-tertiary italic mt-0.5">
              chưa có tài khoản
            </div>
          )}
        </div>

        <ChevronRight className="size-4 text-fg-quaternary shrink-0" aria-hidden="true" />
      </div>

      {/* Info strip — project · bank · account */}
      <div className="border-t border-utility-gray-100 bg-utility-gray-25 px-4 py-2 flex items-center gap-0 flex-wrap min-h-[34px]">
        {/* Project */}
        <span className="inline-flex items-center gap-1 text-xs font-medium text-fg-brand-secondary pr-2.5">
          <Building02 className="size-3 shrink-0 text-fg-brand-secondary/80" aria-hidden="true" />
          {emp.project?.name || "—"}
        </span>

        {/* Bank */}
        {hasBank ? (
          <>
            <span className="text-utility-gray-300 mr-2.5" aria-hidden>·</span>
            <span className="inline-flex items-center gap-1 text-xs font-medium text-fg-secondary pr-2.5">
              <CreditCard01 className="size-3 shrink-0 text-fg-quaternary" aria-hidden="true" />
              {emp.bank!.bankName}
            </span>
          </>
        ) : (
          <>
            <span className="text-utility-gray-300 mr-2.5" aria-hidden>·</span>
            <span className="text-xs text-fg-tertiary italic">
              Chưa có ngân hàng
            </span>
          </>
        )}

        {/* Account number */}
        {emp.bank?.accountNumber && (
          <>
            <span className="text-utility-gray-300 mr-2.5" aria-hidden>·</span>
            <span className="break-all font-mono text-xs text-fg-tertiary tabular-nums">
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

  const { data, isLoading, isError, isFetchNextPageError, refetch, fetchNextPage, hasNextPage, isFetchingNextPage } =
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
          <Skeleton className="h-4 w-full max-w-80" />
        </div>

        <div className="flex flex-col gap-3 sm:hidden">
          {Array.from({ length: 5 }).map((_, i) => (
            <CardSkeleton key={i} />
          ))}
        </div>

        <div className="hidden sm:block bg-card rounded-lg border border-utility-gray-200 shadow-xs overflow-hidden">
          <div className="px-4 py-3 border-b border-utility-gray-200">
            <Skeleton className="h-9 w-72 rounded-lg" />
          </div>
          <table className="w-full">
            <thead>
              <tr>
                {Array.from({ length: 10 }).map((_, i) => (
                  <th key={i} className="px-4 py-3 bg-utility-gray-50">
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


  return (
    <div className="min-h-full pb-[calc(6rem+env(safe-area-inset-bottom))] sm:pb-0">
      <div className="p-4 lg:p-6 space-y-4 max-w-[1400px] mx-auto">

        {/* ── Header ─────────────────────────────────────────────────────── */}
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h1 className="font-display text-display-xs sm:text-display-sm font-semibold tracking-tight text-fg-primary leading-tight">
              Nhân viên
            </h1>
            <p className="text-[13px] text-fg-tertiary mt-0.5 leading-snug">
              Thông tin, ngân hàng và dự án
            </p>
          </div>
          {employees.length > 0 && (
            <div className="shrink-0 mt-0.5 px-2.5 py-1 rounded-lg bg-utility-gray-100 text-xs font-semibold text-fg-secondary tabular-nums">
              {employees.length}
            </div>
          )}
        </div>

        {/* ── Search bar ──────────────────────────────────────────────────── */}
        <div className="relative w-full sm:max-w-[380px]">
          <SearchLg className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-fg-quaternary pointer-events-none" aria-hidden="true" />
          <Input
            type="search"
            aria-label="Tìm nhân viên theo tên hoặc CCCD"
            placeholder="Tìm theo tên, CCCD..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>

        {isError && (
          <div role="alert" className="motion-safe:animate-fade-in-up">
            <ErrorState
              message={isFetchNextPageError
                ? "Không thể tải thêm nhân viên. Danh sách đã tải vẫn được giữ lại."
                : "Không thể tải danh sách nhân viên. Vui lòng thử lại."}
              onRetry={() => void (isFetchNextPageError ? fetchNextPage() : refetch())}
            />
          </div>
        )}

        {/* ── Mobile card list ────────────────────────────────────────────── */}
        <div className="flex flex-col gap-3 sm:hidden">
          {employees.length === 0 && !isLoading && !isError ? (
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

              {!isError && hasNextPage && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="w-full"
                >
                  {isFetchingNextPage ? (
                    <>
                      <span className="h-3.5 w-3.5 animate-spin motion-reduce:animate-none rounded-full border-2 border-fg-primary border-t-transparent" aria-hidden="true" />
                      Đang tải...
                    </>
                  ) : (
                    "Tải thêm nhân viên"
                  )}
                </Button>
              )}

              {!isError && employees.length > 0 && !hasNextPage && (
                <p className="text-center text-xs text-fg-tertiary pb-1">
                  {employees.length} nhân viên
                </p>
              )}
            </>
          )}
        </div>

        {/* ── Desktop table ───────────────────────────────────────────────── */}
        <div className="hidden sm:block bg-card rounded-lg border border-utility-gray-200 shadow-xs overflow-hidden">
          <div className="px-4 pt-3 pb-1">
            <p className="text-xs text-fg-tertiary">Nhấn vào hàng để xem chi tiết</p>
          </div>
          <div role="region" aria-label="Danh sách nhân viên" tabIndex={0} className="overflow-x-auto outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2">
            <table className="w-full min-w-[1040px] border-collapse text-[13px]">
              <thead>
                <tr>
                  <th className="w-10 px-4 py-3 text-left text-xs font-semibold text-fg-tertiary bg-utility-gray-50">
                    #
                  </th>
                  <ColHead icon={User01}>Nhân viên</ColHead>
                  <ColHead icon={Mail01}>Email</ColHead>
                  <ColHead icon={Building02}>Dự án</ColHead>
                  <ColHead icon={CreditCard01}>Ngân hàng</ColHead>
                  <ColHead icon={CreditCard01}>Số tài khoản</ColHead>
                  <ColHead icon={CheckVerified01}>CCCD</ColHead>
                </tr>
              </thead>
              <tbody>
                {employees.map((emp, i) => (
                  <tr
                    key={emp.employeeId}
                    onClick={() => setEditEmployee(emp)}
                    className="border-b border-utility-gray-100 hover:bg-utility-gray-50 transition-colors cursor-pointer group"
                  >
                    <td className="px-4 py-3.5 font-mono text-xs text-fg-tertiary">
                      {String(i + 1).padStart(2, "0")}
                    </td>
                    <td className="px-4 py-3.5 min-w-[220px]">
                      <div className="flex items-center gap-2.5">
                        <div
                          className={cn(
                            "h-7 w-7 rounded-lg flex items-center justify-center shrink-0 text-white text-xs font-bold bg-gradient-to-br select-none",
                            getGradient(emp.fullname)
                          )}
                          aria-hidden
                        >
                          {getInitials(emp.fullname)}
                        </div>
                        <div className="min-w-0">
                          <button
                            type="button"
                            className="min-h-11 text-left font-semibold text-[13px] text-fg-primary leading-tight outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 rounded-md"
                            onClick={(event) => { event.stopPropagation(); setEditEmployee(emp); }}
                            aria-label={`Chỉnh sửa nhân viên ${emp.fullname}`}
                          >
                            {emp.fullname}
                          </button>
                          {emp.username ? (
                            <div className="font-mono text-xs text-fg-tertiary">
                              @{emp.username}
                            </div>
                          ) : (
                            <div className="text-xs text-fg-tertiary italic">
                              chưa có tài khoản
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3.5 min-w-[140px]">
                      <div className="text-[12px] text-fg-secondary truncate">
                        {emp.email || "—"}
                      </div>
                    </td>
                    <td className="px-4 py-3.5 min-w-[180px]">
                      <span className="inline-flex items-center gap-1.5 text-[12px] text-fg-secondary font-medium">
                        <span className="w-1.5 h-1.5 rounded-full bg-utility-brand-500 shrink-0" aria-hidden="true" />
                        {emp.project?.name || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3.5">
                      <span
                        className={cn(
                          "text-[12px]",
                          emp.bank
                            ? "text-fg-secondary"
                            : "text-fg-tertiary italic"
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
                            ? "text-fg-secondary"
                            : "text-fg-tertiary italic"
                        )}
                      >
                        {emp.bank?.accountNumber || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3.5">
                      <span className="font-mono text-[12px] text-fg-secondary">
                        {emp.cccd || "—"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {employees.length === 0 && !isLoading && !isError && (
            <EmptyState hasSearch={!!search} />
          )}

          {employees.length > 0 && !isError && (
            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-utility-gray-200 bg-utility-gray-25 px-4 py-3 text-sm text-fg-tertiary">
              <span>Đã tải <strong className="font-mono text-fg-primary">{employees.length}</strong> nhân viên</span>
              {hasNextPage && (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? "Đang tải..." : "Tải thêm nhân viên"}
                </Button>
              )}
            </div>
          )}
        </div>

        {isFetchingNextPage && (
          <div className="hidden sm:flex items-center justify-center py-4 gap-2 text-sm text-fg-tertiary">
            <span className="h-4 w-4 animate-spin motion-reduce:animate-none rounded-full border-2 border-fg-primary border-t-transparent" aria-hidden="true" />
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
