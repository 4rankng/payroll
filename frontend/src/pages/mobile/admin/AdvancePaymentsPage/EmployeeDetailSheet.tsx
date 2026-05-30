import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Users, Wallet, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet";
import { useAdvancePayments } from "@/hooks/api/useAdvancePayments";
import {
  getVietnameseAdvancePaymentStatus,
  getAdvancePaymentStatusColor,
} from "@/utils/advancePaymentHelpers";
import { formatCurrency } from "@/utils/formatters";
import type {
  FlexPayEmployeeListItem,
  AdvancePaymentListItem,
} from "@/types/api/advance-payment.types";

interface EmployeeDetailSheetProps {
  employee: FlexPayEmployeeListItem | null;
  onClose: () => void;
}

export function EmployeeDetailSheet({
  employee,
  onClose,
}: EmployeeDetailSheetProps) {
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
      <SheetContent side="bottom" className="h-[92dvh] flex flex-col p-0">
        {employee &&
          (() => {
            const emp = employee;
            const usedPct =
              emp.maxAdvanceAmount > 0
                ? Math.round(
                    (emp.utilizedAmount / emp.maxAdvanceAmount) * 100,
                  )
                : 0;
            const barColor =
              usedPct > 80 ? "bg-orange-400" : "bg-emerald-400";
            const requests = (employeeRequests?.data ??
              []) as AdvancePaymentListItem[];

            return (
              <>
                {/* Header */}
                <div className="flex items-center justify-between px-4 pt-5 pb-3 border-b shrink-0">
                  <div className="flex items-center gap-2">
                    <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/10">
                      <Users className="h-4 w-4 text-primary" />
                    </div>
                    <div>
                      <SheetTitle className="text-base">
                        {emp.fullname}
                      </SheetTitle>
                      <SheetDescription className="text-xs font-mono">
                        {emp.cccd}
                      </SheetDescription>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={onClose}
                    aria-label="Đóng"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>

                <div className="flex-1 overflow-y-auto">
                  {/* Quick info pills */}
                  <div className="flex items-center gap-2 px-4 pt-3">
                    <span className="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-xs text-muted-foreground">
                      {emp.project?.name || "Chưa có dự án"}
                    </span>
                    {emp.bank?.accountNumber && (
                      <span className="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-xs text-muted-foreground">
                        {emp.bank.bankName || "Ngân hàng"}
                      </span>
                    )}
                  </div>

                  {/* Advance usage card */}
                  <div className="px-4 py-3">
                    <div className="rounded-xl border bg-card p-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <p className="text-xs font-medium text-muted-foreground">
                          Hạn mức ứng lương
                        </p>
                        <p className="text-lg font-bold tabular-nums">
                          {formatCurrency(emp.availableAmount)}
                        </p>
                      </div>
                      <p className="text-xs text-muted-foreground -mt-2">
                        Số tiền còn lại
                      </p>
                      <div>
                        <div className="flex items-center justify-between text-xs mb-1.5">
                          <span className="text-muted-foreground">
                            Đã sử dụng
                          </span>
                          <span className="font-medium tabular-nums">
                            {usedPct}%
                          </span>
                        </div>
                        <div className="h-2 bg-muted rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${barColor}`}
                            style={{ width: `${Math.min(100, usedPct)}%` }}
                          />
                        </div>
                      </div>
                      <div className="grid grid-cols-3 gap-2 pt-1">
                        <div className="text-center">
                          <p className="text-xs text-muted-foreground">
                            Hạn mức
                          </p>
                          <p className="text-xs font-semibold tabular-nums">
                            {formatCurrency(emp.maxAdvanceAmount)}
                          </p>
                        </div>
                        <div className="text-center">
                          <p className="text-xs text-muted-foreground">
                            Đã dùng
                          </p>
                          <p className="text-xs font-semibold text-orange-600 tabular-nums">
                            {formatCurrency(emp.utilizedAmount)}
                          </p>
                        </div>
                        <div className="text-center">
                          <p className="text-xs text-muted-foreground">
                            Chờ xử lý
                          </p>
                          <p className="text-xs font-semibold text-amber-600 tabular-nums">
                            {formatCurrency(emp.pendingAmount)}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Info grid */}
                  <div className="px-4 space-y-2 pb-3">
                    {emp.bank?.accountNumber && (
                      <div className="rounded-xl border bg-card p-3">
                        <p className="text-xs font-medium text-muted-foreground mb-1">
                          Ngân hàng
                        </p>
                        <p className="text-sm">{emp.bank.bankName || "-"}</p>
                        <p className="text-xs text-muted-foreground tabular-nums">
                          {emp.bank.accountNumber}
                        </p>
                      </div>
                    )}
                    <div className="grid grid-cols-1 gap-2">
                      <div className="rounded-xl border bg-card p-3 text-center">
                        <p className="text-xs text-muted-foreground">
                          Phí thu
                        </p>
                        <p className="text-sm font-semibold tabular-nums mt-0.5">
                          {formatCurrency(emp.totalFeeGenerated)}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Requests list */}
                  <div className="border-t">
                    <div className="px-4 py-3">
                      <div className="flex items-center justify-between">
                        <p className="text-xs font-medium text-muted-foreground">
                          Yêu cầu ứng lương
                        </p>
                        <span className="text-xs text-muted-foreground tabular-nums">
                          {requests.length} yêu cầu
                        </span>
                      </div>
                    </div>

                    {requests.length === 0 ? (
                      <div className="px-4 pb-6 text-center">
                        <Wallet className="mx-auto h-8 w-8 text-muted-foreground/40 mb-2" />
                        <p className="text-xs text-muted-foreground">
                          Chưa có yêu cầu nào
                        </p>
                      </div>
                    ) : (
                      <div className="px-4 pb-4 space-y-2">
                        {requests.map((req) => (
                          <div
                            key={req.id}
                            className="rounded-lg border bg-card p-3"
                          >
                            <div className="flex items-center gap-2 mb-0.5">
                              <span className="text-sm font-semibold tabular-nums">
                                {formatCurrency(req.requestAmount)}
                              </span>
                              <Badge
                                variant="outline"
                                className={`${getAdvancePaymentStatusColor(req.status)} text-[10px] px-1.5 py-0 h-4`}
                              >
                                {getVietnameseAdvancePaymentStatus(req.status)}
                              </Badge>
                            </div>
                            <div className="flex items-center gap-3 text-xs text-muted-foreground">
                              <span>Phí: {formatCurrency(req.fee)}</span>
                              <span>
                                Thực nhận: {formatCurrency(req.netAmount)}
                              </span>
                            </div>
                            <p className="text-xs text-muted-foreground mt-0.5">
                              {format(
                                new Date(req.createdAt),
                                "dd/MM/yyyy HH:mm",
                                { locale: vi },
                              )}
                            </p>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </>
            );
          })()}
      </SheetContent>
    </Sheet>
  );
}
