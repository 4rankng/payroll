import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Users, Wallet } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
} from "@/components/ui/sheet";
import { MobileSheetHeader } from "@/components/shared/MobileSheetHeader";
import { MobileCard } from "@/components/shared/MobileCard";
import { MobileProgressBar } from "@/components/shared/MobileProgressBar";
import { MobileFilterPill } from "@/components/shared/MobileFilterPill";
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
            const requests = (employeeRequests?.data ??
              []) as AdvancePaymentListItem[];

            return (
              <>
                <MobileSheetHeader
                  icon={Users}
                  title={emp.fullname}
                  description={emp.cccd}
                  onClose={onClose}
                />

                <div className="flex-1 overflow-y-auto">
                  {/* Quick info pills */}
                  <div className="flex items-center gap-2 px-4 pt-3">
                    <MobileFilterPill>
                      {emp.project?.name || "Chưa có dự án"}
                    </MobileFilterPill>
                    {emp.bank?.accountNumber && (
                      <MobileFilterPill>
                        {emp.bank.bankName || "Ngân hàng"}
                      </MobileFilterPill>
                    )}
                  </div>

                  {/* Advance usage card */}
                  <div className="px-4 py-3">
                    <MobileCard padding="lg" className="space-y-3">
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
                      <MobileProgressBar
                        value={usedPct}
                        label="Đã sử dụng"
                        displayValue={`${usedPct}%`}
                      />
                      <div className="grid grid-cols-1 gap-2 pt-1 sm:grid-cols-3">
                        <div className="rounded-lg bg-muted/40 px-2.5 py-2">
                          <p className="text-xs text-muted-foreground">Hạn mức</p>
                          <p className="text-xs font-semibold tabular-nums">{formatCurrency(emp.maxAdvanceAmount)}</p>
                        </div>
                        <div className="rounded-lg bg-orange-50 px-2.5 py-2">
                          <p className="text-xs text-muted-foreground">Đã dùng</p>
                          <p className="text-xs font-semibold text-orange-600 tabular-nums">{formatCurrency(emp.utilizedAmount)}</p>
                        </div>
                        <div className="rounded-lg bg-amber-50 px-2.5 py-2">
                          <p className="text-xs text-muted-foreground">Chờ xử lý</p>
                          <p className="text-xs font-semibold text-amber-600 tabular-nums">{formatCurrency(emp.pendingAmount)}</p>
                        </div>
                      </div>
                    </MobileCard>
                  </div>

                  {/* Info grid */}
                  <div className="px-4 space-y-2 pb-3">
                    {emp.bank?.accountNumber && (
                      <MobileCard>
                        <p className="text-xs font-medium text-muted-foreground mb-1">
                          Ngân hàng
                        </p>
                        <p className="text-sm">{emp.bank.bankName || "-"}</p>
                        <p className="text-xs text-muted-foreground tabular-nums">
                          {emp.bank.accountNumber}
                        </p>
                      </MobileCard>
                    )}
                    <div className="grid grid-cols-1 gap-2">
                      <MobileCard className="text-center">
                        <p className="text-xs text-muted-foreground">Phí thu</p>
                        <p className="text-sm font-semibold tabular-nums mt-0.5">
                          {formatCurrency(emp.totalFeeGenerated)}
                        </p>
                      </MobileCard>
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
                          <MobileCard key={req.id}>
                            <div className="flex items-center gap-2 mb-0.5">
                              <span className="text-sm font-semibold tabular-nums">
                                {formatCurrency(req.requestAmount)}
                              </span>
                              <Badge
                                variant="outline"
                                className={`${getAdvancePaymentStatusColor(req.status)} text-[11px] px-1.5 py-0 h-4`}
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
                          </MobileCard>
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
