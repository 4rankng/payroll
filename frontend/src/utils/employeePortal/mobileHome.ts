import { formatCurrency, formatNumber } from "@/utils/formatters";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";
import type { EmployeeProfile } from "@/types/api/auth.types";

export type EmployeeActionIcon =
  | "advance"
  | "attendance"
  | "history"
  | "account"
  | "timesheet"
  | "notification"
  | "limit";

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
  return `Tháng ${value.slice(5, 7)}/${value.slice(0, 4)}`;
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
  hasBankInfo,
}: FlexibleEmployeeHomeInput): EmployeeHomeViewModel {
  const pendingRequests = history.filter((item) => item.status === "PENDING").length;
  const payrollPeriod = formatPayrollMonth(info.forMonth);
  const quickActions: EmployeeQuickAction[] = [
    {
      id: "advance",
      label: "Ứng lương",
      icon: "advance",
      targetId: info.canRequest ? "employee-advance-request" : "employee-limit",
      disabled: !info.canRequest && !info.canRequestReason,
    },
    hasCheckIn
      ? {
          id: "attendance",
          label: "Chấm công",
          icon: "attendance",
          targetId: "employee-check-in",
        }
      : {
          id: "limit",
          label: "Hạn mức",
          icon: "limit",
          targetId: "employee-limit",
        },
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
  if (!hasBankInfo) {
    nudges.push({
      id: "bank-missing",
      title: "Kiểm tra tài khoản nhận tiền",
      description: "Thiếu thông tin ngân hàng, hãy báo quản lý cập nhật.",
      icon: "account",
      tone: "amber",
      targetId: "employee-bank",
    });
  } else {
    nudges.push({
      id: "bank-ready",
      title: "Kiểm tra tài khoản nhận tiền",
      description: "Xác nhận đúng tài khoản trước khi gửi yêu cầu.",
      icon: "account",
      tone: "slate",
      targetId: "employee-bank",
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
    amountLabel: "Có thể ứng",
    amount: formatCurrency(info.remainingAmount),
    amountDescription: info.canRequest
      ? `Sẵn sàng gửi yêu cầu cho ${payrollPeriod.toLowerCase()}.`
      : info.canRequestTitle || "Chưa thể gửi yêu cầu lúc này.",
    periodLabel: payrollPeriod,
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
  hasBankInfo,
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
    {
      id: hasBankInfo ? "bank-ready" : "bank-missing",
      title: "Kiểm tra tài khoản nhận tiền",
      description: hasBankInfo
        ? "Tài khoản nhận lương đã sẵn sàng để đối chiếu."
        : "Thiếu thông tin ngân hàng, hãy báo quản lý cập nhật.",
      icon: "account",
      tone: hasBankInfo ? "slate" : "amber",
      targetId: "employee-bank",
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
