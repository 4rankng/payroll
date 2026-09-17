import type { ColumnDef } from "@tanstack/react-table";
import type { MobileField } from "@/components/ui/responsive-table";
import type {
  AdvancePaymentListItem,
  FlexPayEmployeeListItem,
} from "@/types/api/advance-payment.types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { AdvanceRequestToggle } from "@/components/advance-payment/AdvanceRequestToggle";
import { Wallet, Users, ArrowUpDown, ArrowUp, ArrowDown, MoreHorizontal, RotateCcw, X } from "lucide-react";
import { EmptyState } from "@/components/shared/EmptyState";
import { formatCurrency } from "@/utils/formatters";
import { format, differenceInMinutes } from "date-fns";
import { vi } from "date-fns/locale";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  getVietnameseAdvancePaymentStatus,
  getAdvancePaymentStatusColor,
} from "@/utils/advancePaymentHelpers";

export interface ActionContext {
  onCancel: (id: number) => void;
  onRetry?: (id: number) => void;
  isRetrying?: boolean;
  retryingId?: number;
  /** Set of request IDs currently polling for disbursement status */
  pollingIds?: Set<number>;
}

export function getAdvancePaymentColumns(
  ctx: ActionContext,
): ColumnDef<AdvancePaymentListItem>[] {
  return [
    {
      accessorKey: "employeeName",
      header: "Nhân viên",
      size: 130,
      cell: ({ row }) => (
        <div>
          <div className="text-xs font-medium leading-tight">{row.original.employeeName}</div>
          <div className="text-xs text-muted-foreground mt-0.5 tabular-nums">
            {row.original.employeeCCCD}
          </div>
          <div className="text-xs text-muted-foreground mt-0.5 truncate max-w-[120px]">
            {row.original.projectName || "-"}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "requestAmount",
      header: "Yêu cầu",
      size: 100,
      cell: ({ row }) => (
        <span className="text-xs font-semibold tabular-nums">
          {formatCurrency(row.original.requestAmount)}
        </span>
      ),
    },
    {
      accessorKey: "fee",
      header: "Phí",
      size: 80,
      cell: ({ row }) => (
        <div className="text-xs text-muted-foreground tabular-nums">
          {formatCurrency(row.original.fee)}
        </div>
      ),
    },
    {
      accessorKey: "netAmount",
      header: "Thực nhận",
      size: 100,
      cell: ({ row }) => (
        <div className="text-xs font-semibold text-emerald-700 tabular-nums">
          {formatCurrency(row.original.netAmount)}
        </div>
      ),
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      size: 120,
      cell: ({ row }) => {
        const status = row.original.status;
        const isPolling = ctx.pollingIds?.has(row.original.id);
        return (
          <div className="flex items-center gap-1.5">
            <Badge variant="outline" className={`${getAdvancePaymentStatusColor(status)} text-xs px-1.5 py-0 h-5 whitespace-nowrap`}>
              {getVietnameseAdvancePaymentStatus(status)}
            </Badge>
            {isPolling && (
              <span className="h-3.5 w-3.5 shrink-0 animate-spin rounded-full border-2 border-blue-300 border-t-blue-600" />
            )}
          </div>
        );
      },
    },
    {
      accessorKey: "paidAt",
      header: "Thanh toán",
      size: 130,
      cell: ({ row }) => {
        const paidAt = row.original.paidAt ?? row.original.completedAt;
        if (!paidAt) return <span className="text-xs text-muted-foreground">-</span>;
        return (
          <div className="text-xs text-muted-foreground tabular-nums whitespace-nowrap">
            {format(new Date(paidAt), "dd/MM/yyyy HH:mm", { locale: vi })}
          </div>
        );
      },
    },
    {
      id: "latency",
      header: "Độ trễ",
      size: 80,
      cell: ({ row }) => {
        const completedAt = row.original.paidAt ?? row.original.completedAt;
        const { createdAt } = row.original;
        if (!completedAt) return <span className="text-xs text-muted-foreground">-</span>;
        const diffMs = new Date(completedAt).getTime() - new Date(createdAt).getTime();
        const totalSecs = Math.round(diffMs / 1000);
        const mins = Math.floor(totalSecs / 60);
        if (mins < 1) {
          return <span className="text-xs tabular-nums">{totalSecs} giây</span>;
        }
        if (mins < 60) {
          return <span className="text-xs tabular-nums">{mins} phút</span>;
        }
        const hours = Math.floor(mins / 60);
        const remainMins = mins % 60;
        return <span className="text-xs tabular-nums">{hours} giờ {remainMins} phút</span>;
      },
    },
    {
      accessorKey: "createdAt",
      header: "Ngày tạo",
      size: 130,
      cell: ({ row }) => (
        <div className="text-xs text-muted-foreground tabular-nums whitespace-nowrap">
          {format(new Date(row.original.createdAt), "dd/MM/yyyy HH:mm", {
            locale: vi,
          })}
        </div>
      ),
    },
    {
      id: "actions",
      header: "",
      size: 40,
      cell: ({ row }) => {
        const { status, id } = row.original;
        const isCancellable = status === "PENDING" || status === "APPROVED";
        const isRetryable = status === "APPROVED" || status === "FAILED";
        const hasActions = isCancellable || (isRetryable && ctx.onRetry);
        if (!hasActions) return null;

        const isRetrying = ctx.isRetrying && ctx.retryingId === id;

        return (
          <div className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" aria-label={`Thao tác yêu cầu ứng lương của ${row.original.employeeName}`} className="h-9 w-9 shrink-0">
                  <MoreHorizontal className="h-3.5 w-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-40">
                {isRetryable && ctx.onRetry && (
                  <DropdownMenuItem
                    onClick={() => ctx.onRetry!(id)}
                    disabled={isRetrying}
                    className="text-blue-600 focus:text-blue-700"
                  >
                    <RotateCcw className="mr-2 h-3.5 w-3.5" />
                    {isRetrying ? "Đang gửi..." : "Thử lại"}
                  </DropdownMenuItem>
                )}
                {isCancellable && (
                  <DropdownMenuItem
                    onClick={() => ctx.onCancel(id)}
                    className="text-red-700 focus:text-red-800"
                  >
                    <X className="mr-2 h-3.5 w-3.5" />
                    Hủy yêu cầu
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
    },
  ];
}

export interface SortContext {
  sortBy?: string;
  sortOrder?: "ASC" | "DESC";
  onSort: (field: string) => void;
}

function SortHeader({ label, field, ctx }: { label: string; field: string; ctx: SortContext }) {
  const isActive = ctx.sortBy === field;
  const Icon = isActive ? (ctx.sortOrder === "ASC" ? ArrowUp : ArrowDown) : ArrowUpDown;
  return (
    <button
      className="flex items-center gap-1 hover:text-foreground transition-colors"
      onClick={() => ctx.onSort(field)}
    >
      {label}
      <Icon className={`h-3 w-3 ${isActive ? "text-foreground" : "text-muted-foreground/50"}`} />
    </button>
  );
}

// Shared kill-switch renderer: one place for the fallback semantics so the
// desktop column, the mobile card fields, and the employee list card cannot
// drift. Rendered only when the row carries a project (the PATCH needs its id).
export function renderAdvanceToggle(row: FlexPayEmployeeListItem) {
  if (!row.project) return "-";
  return (
    <AdvanceRequestToggle
      projectId={row.project.id}
      employeeId={row.employeeId}
      employeeName={row.fullname}
      enabled={row.project.advance_request_enabled !== false}
    />
  );
}

export function getFlexPayColumns(sort: SortContext): ColumnDef<FlexPayEmployeeListItem>[] {
  return [
  {
    accessorKey: "fullname",
    header: "Nhân viên",
    size: 130,
    cell: ({ row }) => (
      <div>
        <div className="text-xs font-medium leading-tight">{row.original.fullname}</div>
        <div className="text-xs text-muted-foreground mt-0.5 tabular-nums">{row.original.cccd}</div>
        <div className="text-xs text-muted-foreground mt-0.5 truncate max-w-[120px]">
          {row.original.project?.name || "-"}
        </div>
      </div>
    ),
  },
  {
    id: "bank",
    header: "Ngân hàng",
    size: 140,
    cell: ({ row }) => {
      const bank = row.original.bank;
      if (!bank || !bank.accountNumber) {
        return <span className="text-xs text-muted-foreground">-</span>;
      }
      return (
        <div>
          <div className="text-xs truncate max-w-[140px]">{bank.bankName || "-"}</div>
          <div className="text-xs text-muted-foreground tabular-nums mt-0.5">
            {bank.accountNumber}
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "maxAdvanceAmount",
    header: () => <SortHeader label="Hạn mức" field="max_advance_amount" ctx={sort} />,
    size: 90,
    cell: ({ row }) => (
      <div className="text-xs font-medium tabular-nums">
        {formatCurrency(row.original.maxAdvanceAmount)}
      </div>
    ),
  },
  {
    accessorKey: "utilizedAmount",
    header: () => <SortHeader label="Đã dùng" field="utilized_amount" ctx={sort} />,
    size: 90,
    cell: ({ row }) => (
      <div className="text-xs text-orange-700 tabular-nums">
        {formatCurrency(row.original.utilizedAmount)}
      </div>
    ),
  },
  {
    accessorKey: "pendingAmount",
    header: () => <SortHeader label="Chờ xử lý" field="pending_amount" ctx={sort} />,
    size: 90,
    cell: ({ row }) => (
      <div className="text-xs text-amber-700 tabular-nums">
        {formatCurrency(row.original.pendingAmount)}
      </div>
    ),
  },
  {
    accessorKey: "availableAmount",
    header: () => <SortHeader label="Còn lại" field="available_amount" ctx={sort} />,
    size: 90,
    cell: ({ row }) => (
      <div className="text-xs font-semibold text-emerald-700 tabular-nums">
        {formatCurrency(row.original.availableAmount)}
      </div>
    ),
  },
  {
    accessorKey: "totalFeeGenerated",
    header: "Phí thu",
    size: 80,
    cell: ({ row }) => (
      <div className="text-xs text-muted-foreground tabular-nums">
        {formatCurrency(row.original.totalFeeGenerated)}
      </div>
    ),
  },
  {
    id: "advanceRequest",
    header: "Ứng lương",
    size: 110,
    cell: ({ row }) => renderAdvanceToggle(row.original),
  },
];
}

export const requestMobileFields: MobileField<AdvancePaymentListItem>[] = [
  {
    key: "requestAmount",
    label: "Số tiền",
    priority: 1,
    render: (row) => (
      <div className="flex flex-col items-end gap-1">
        <span className="text-[15px] font-bold text-foreground tabular-nums leading-none">
          {formatCurrency(row.requestAmount)}
        </span>
        <Badge
          variant="outline"
          className={`text-xs px-1.5 py-0.5 font-semibold leading-none h-auto ${getAdvancePaymentStatusColor(row.status)}`}
        >
          {getVietnameseAdvancePaymentStatus(row.status)}
        </Badge>
      </div>
    ),
  },
  {
    key: "netAmount",
    label: "Thực nhận",
    priority: 2,
    render: (row) => (
      <div className="text-[13px] font-semibold text-emerald-700 tabular-nums">
        {formatCurrency(row.netAmount)}
      </div>
    ),
  },
  {
    key: "fee",
    label: "Phí",
    priority: 2,
    render: (row) => (
      <div className="text-[13px] text-muted-foreground tabular-nums">
        {formatCurrency(row.fee)}
      </div>
    ),
  },
  {
    key: "createdAt",
    label: "Ngày tạo",
    priority: 2,
    render: (row) => (
      <div className="text-[13px] text-muted-foreground tabular-nums">
        {format(new Date(row.createdAt), "dd/MM/yyyy HH:mm", { locale: vi })}
      </div>
    ),
  },
  {
    key: "paidAt",
    label: "Thanh toán",
    priority: 2,
    render: (row) => {
      const paidAt = row.paidAt ?? row.completedAt;
      if (!paidAt) return <span className="text-[13px] text-muted-foreground">—</span>;
      return (
        <div className="text-[13px] text-muted-foreground tabular-nums">
          {format(new Date(paidAt), "dd/MM/yyyy HH:mm", { locale: vi })}
        </div>
      );
    },
  },
  {
    key: "latency",
    label: "Độ trễ",
    priority: 3,
    render: (row) => {
      const paidAt = row.paidAt ?? row.completedAt;
      if (!paidAt) return <span className="text-[13px] text-muted-foreground">—</span>;
      const diffMs = new Date(paidAt).getTime() - new Date(row.createdAt).getTime();
      const totalSecs = Math.round(diffMs / 1000);
      const mins = Math.floor(totalSecs / 60);
      if (mins < 1) return <span className="text-[13px]">{totalSecs}s</span>;
      if (mins < 60) return <span className="text-[13px]">{mins} phút</span>;
      const hours = Math.floor(mins / 60);
      const remainMins = mins % 60;
      return <span className="text-[13px]">{hours}g {remainMins}p</span>;
    },
  },
];

export const flexPayMobileFields: MobileField<FlexPayEmployeeListItem>[] = [

  {
    key: "maxAdvanceAmount",
    label: "Hạn mức",
    priority: 1,
    render: (row) => (
      <div className="text-[13px] font-medium text-slate-500">{formatCurrency(row.maxAdvanceAmount)}</div>
    ),
  },
  {
    key: "availableAmount",
    label: "Còn lại",
    priority: 1,
    render: (row) => (
      <div className="text-[14px] font-bold text-emerald-700">
        {formatCurrency(row.availableAmount)}
      </div>
    ),
  },
  {
    key: "utilizedAmount",
    label: "Đã dùng",
    priority: 2,
    render: (row) => (
      <div className="text-orange-700">
        {formatCurrency(row.utilizedAmount)}
      </div>
    ),
  },
  {
    key: "bank",
    label: "Ngân hàng",
    priority: 2,
    render: (row) => {
      const bank = row.bank;
      if (!bank || !bank.accountNumber) return "-";
      return (
        <div className="text-sm">
          <div className="truncate">{bank.bankName || "-"}</div>
          <div className="text-xs text-muted-foreground">
            {bank.accountNumber}
          </div>
        </div>
      );
    },
  },
  {
    key: "advanceRequest",
    label: "Ứng lương",
    priority: 2,
    render: renderAdvanceToggle,
  },
];

export const requestEmptyState = (
  <EmptyState
    title="Chưa có yêu cầu nào"
    description="Yêu cầu ứng lương sẽ hiển thị tại đây sau khi nhân viên gửi."
    size="sm"
  />
);

export const flexPayEmptyState = (
  <EmptyState
    title="Chưa có dữ liệu"
    description="Vui lòng nhập file bảng lương để xem danh sách nhân viên."
    size="sm"
  />
);
