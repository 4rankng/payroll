import { formatCurrency, formatNumber } from "@/utils/formatters";
import { formatAdvancePeriodDisplay } from "@/utils/advancePaymentHelpers";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";
import type { EmployeeProfile } from "@/types/api/auth.types";

export type EmployeeActionIcon =
  | "advance"
  | "attendance"
  | "history"
  | "account"
  | "timesheet"
  | "notification";

export type EmployeeNudgeTone = "employee" | "amber" | "slate";

export interface EmployeeQuickAction {
  id: string;
  label: string;
  icon: EmployeeActionIcon;
  targetId?: string;
  intent?: "notifications";
  disabled?: boolean;
}

export interface EmployeeNudge {
  id: string;
  title: string;
  description: string;
  icon: EmployeeActionIcon;
  tone: EmployeeNudgeTone;
  targetId?: string;
  intent?: "notifications";
}

export interface EmployeeWalletMetric {
  label: string;
  value: string;
  tone?: EmployeeNudgeTone;
}

export interface EmployeeHomeViewModel {
  eyebrow: string;
  title: string;
  amountLabel: string;
  amount: string;
  amountDescription: string;
  periodLabel?: string;
  metrics: EmployeeWalletMetric[];
  quickActions: EmployeeQuickAction[];
  nudges: EmployeeNudge[];
}

interface FlexibleEmployeeHomeInput {
  info: AdvancePaymentInfo;
  history: AdvancePaymentHistoryItem[];
  hasCheckIn: boolean;
  hasBankInfo: boolean;
}

interface RegularEmployeeHomeInput {
  monthLabel: string;
  monthlyTotalSalary: number;
  monthlyTotalHours: number;
  totalPayable: number;
  totalPaid: number;
  workDayCount: number;
  totalRecords: number;
  hasBankInfo: boolean;
  unreadCount?: number;
}

export function formatPayrollMonth(value?: string): string {
  if (!value || !/^\d{4}-\d{2}/.test(value)) return "Kỳ lương hiện tại";
  return formatAdvancePeriodDisplay(value);
}

export function maskBankAccountNumber(value?: string | null): string | undefined {
  const normalized = value?.replace(/\s+/g, "") ?? "";
  if (!normalized) return undefined;
  if (normalized.length <= 4) return normalized;
  return `**** **** ${normalized.slice(-4)}`;
}

export function getEmployeeAccountHolder(profile?: EmployeeProfile | null): string | undefined {
  return profile?.bank_account_name?.trim() || profile?.fullname?.trim();
}

export function hasEmployeeBankInfo(profile?: EmployeeProfile | null): boolean {
  return Boolean(profile?.bank_account_number && profile.bank?.branch_name);
}

export function createFlexibleEmployeeHomeModel({
  info,
  history,
  hasCheckIn,
}: FlexibleEmployeeHomeInput): EmployeeHomeViewModel {
  const pendingRequests = history.filter((item) => item.status === "PENDING").length;
  const requestBlocked = !info.canRequest;
  const blockedDescription =
    info.canRequestReason || info.canRequestTitle || "Chưa thể gửi yêu cầu lúc này.";
  const quickActions: EmployeeQuickAction[] = [
    ...(info.canRequest
      ? [
          {
            id: "advance",
            label: "Ứng lương",
            icon: "advance" as const,
            targetId: "employee-advance-request",
          },
        ]
      : []),
    ...(hasCheckIn
      ? [
          {
            id: "attendance",
            label: "Chấm công",
            icon: "attendance" as const,
            targetId: "employee-check-in",
          },
        ]
      : []),
    {
      id: "history",
      label: "Lịch sử",
      icon: "history",
      targetId: "employee-history",
    },
    {
      id: "account",
      label: "Tài khoản",
      icon: "account",
      targetId: "employee-bank",
    },
  ];

  const nudges: EmployeeNudge[] = [];
  if (hasCheckIn) {
    nudges.push({
      id: "check-in-today",
      title: "Chấm công hôm nay",
      description: "Kiểm tra vị trí và ca làm trước khi bắt đầu.",
      icon: "attendance",
      tone: "employee",
      targetId: "employee-check-in",
    });
  }
  if (pendingRequests > 0) {
    nudges.unshift({
      id: "pending-request",
      title: "Yêu cầu đang chờ",
      description: `${pendingRequests} yêu cầu đang được xử lý.`,
      icon: "history",
      tone: "amber",
      targetId: "employee-history",
    });
  }

  return {
    eyebrow: "Ví lương",
    title: "Gói lương sớm",
    amountLabel: requestBlocked ? "Hạn mức còn lại" : "Có thể ứng",
    amount: formatCurrency(info.remainingAmount),
    amountDescription: info.canRequest
      ? "Sẵn sàng gửi yêu cầu ứng lương."
      : blockedDescription,
    metrics: [
      { label: "Đã nhận", value: formatCurrency(info.completedAmount), tone: "employee" },
      { label: "Đang chờ", value: formatCurrency(info.pendingAmount), tone: "amber" },
      { label: "Hạn mức", value: formatCurrency(info.maxAdvanceAmount), tone: "slate" },
    ],
    quickActions,
    nudges,
  };
}

export function createRegularEmployeeHomeModel({
  monthLabel,
  monthlyTotalSalary,
  monthlyTotalHours,
  totalPayable,
  totalPaid,
  workDayCount,
  totalRecords,
  unreadCount,
}: RegularEmployeeHomeInput): EmployeeHomeViewModel {
  const nudges: EmployeeNudge[] = [
    {
      id: "timesheet-current-month",
      title: "Xem bảng công tháng này",
      description: `${totalRecords} dòng công trong ${monthLabel.toLowerCase()}.`,
      icon: "timesheet",
      tone: "employee",
      targetId: "employee-timesheets",
    },
  ];

  if (unreadCount && unreadCount > 0) {
    nudges.unshift({
      id: "unread-notifications",
      title: "Thông báo mới",
      description: `${unreadCount > 9 ? "9+" : unreadCount} thông báo cần xem.`,
      icon: "notification",
      tone: "amber",
      intent: "notifications",
    });
  }

  return {
    eyebrow: "Tổng quan lương",
    title: "Kỳ lương của bạn",
    amountLabel: "Tổng lương trong kỳ",
    amount: formatCurrency(monthlyTotalSalary),
    amountDescription: `${workDayCount} ngày làm việc trong ${monthLabel.toLowerCase()}.`,
    periodLabel: monthLabel,
    metrics: [
      {
        label: "Tổng công",
        value: `${monthlyTotalHours % 1 === 0 ? monthlyTotalHours : formatNumber(monthlyTotalHours, 1)} giờ`,
        tone: "employee",
      },
      { label: "Có thể trả", value: formatCurrency(totalPayable), tone: "slate" },
      { label: "Đã nhận", value: formatCurrency(totalPaid), tone: "amber" },
    ],
    quickActions: [
      {
        id: "timesheet",
        label: "Bảng công",
        icon: "timesheet",
        targetId: "employee-timesheets",
      },
      {
        id: "account",
        label: "Tài khoản",
        icon: "account",
        targetId: "employee-bank",
      },
      {
        id: "notification",
        label: "Thông báo",
        icon: "notification",
        intent: "notifications",
      },
    ],
    nudges,
  };
}
