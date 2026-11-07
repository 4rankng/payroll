import { Badge } from "@/components/ui/badge";
import { Target } from "lucide-react";
import { Project } from "@/types/api/project.types";

export const formatProjectCurrency = (amount: number): string => {
  return new Intl.NumberFormat('vi-VN').format(amount) + ' ₫';
};

export const getStatusColor = (status: string): string => {
  switch (status) {
    case "active":
      // Staff day console: semantic status tokens (dark: variants dropped —
      // staff surfaces are light-only per ADR-013).
      return "bg-success/10 text-success";
    case "completed":
      return "bg-info/10 text-info";
    case "draft":
      return "bg-warning/10 text-warning";
    case "cancelled":
      return "bg-destructive/10 text-destructive";
    case "paused":
      return "bg-warning/15 text-warning";
    default:
      return "bg-secondary text-foreground";
  }
};

export const getPriorityColor = (priority: string): 'destructive' | 'outline' | 'secondary' => {
  switch (priority) {
    case "high":
      return "destructive";
    case "medium":
      return "outline";
    case "low":
      return "secondary";
    default:
      return "outline";
  }
};

export const calculateProjectStats = (projects: Project[]) => {
  const totalBudget = projects.reduce((sum, project) => sum + project.budget, 0);
  const totalSpent = projects.reduce((sum, project) => sum + project.spent, 0);
  const activeProjects = projects.filter(p => p.status === "active").length;
  const completedProjects = projects.filter(p => p.status === "completed").length;
  const completionRate = projects.length > 0 ? Math.round((completedProjects / projects.length) * 100) : 0;

  return {
    totalProjects: projects.length,
    activeProjects,
    totalBudget,
    totalSpent,
    completionRate,
    averageProgress: projects.length > 0 ? Math.round(projects.reduce((sum, p) => sum + p.progress, 0) / projects.length) : 0,
  };
};

export const formatBudgetDisplay = (amount: number): string => {
  return formatProjectCurrency(amount);
};

/**
 * Get color class for employee count display
 * Returns different colors based on team size
 */
export const getEmployeeCountColor = (count: number): string => {
  if (count === 0) {
    return "text-muted-foreground";
  }
  if (count <= 10) {
    return "text-blue-700 dark:text-blue-400";
  }
  if (count <= 20) {
    return "text-green-700 dark:text-green-400";
  }
  return "text-teal-700 dark:text-teal-400";
};

/**
 * Format employee count for display
 * Returns '-' for zero, otherwise returns the count
 */
export const formatEmployeeCount = (count: number | undefined | null): string => {
  if (!count || count === 0) {
    return "-";
  }
  return count.toString();
};

/**
 * Format a project's salary period for display in the projects table.
 * Defaults to the full current month when both period markers are zero.
 */
export const formatSalaryPeriodLabel = (
  salaryPeriodFrom?: number | null,
  salaryPeriodTo?: number | null,
): string => {
  const now = new Date();
  const currentMonthIndex = now.getMonth();
  const currentYear = now.getFullYear();
  const lastDayOfCurrentMonth = new Date(currentYear, currentMonthIndex + 1, 0).getDate();

  const fromValue = salaryPeriodFrom ?? 0;
  const toValue = salaryPeriodTo ?? 0;

  const formatDayMonth = (day: number, month: number) => `${day.toString().padStart(2, "0")}/${month
    .toString()
    .padStart(2, "0")}`;

  if (fromValue === 0 && toValue === 0) {
    const currentMonthNumber = currentMonthIndex + 1;
    return `${formatDayMonth(1, currentMonthNumber)} - ${formatDayMonth(
      lastDayOfCurrentMonth,
      currentMonthNumber,
    )}`;
  }

  const startDay = fromValue > 0 ? fromValue : 1;
  const endDay = toValue > 0 ? toValue : lastDayOfCurrentMonth;
  const previousMonthNumber = ((currentMonthIndex + 11) % 12) + 1;
  const currentMonthNumber = currentMonthIndex + 1;

  return `${formatDayMonth(startDay, previousMonthNumber)} - ${formatDayMonth(endDay, currentMonthNumber)}`;
};
