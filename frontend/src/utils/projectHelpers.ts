import { Badge } from "@/components/ui/badge";
import { Target } from "lucide-react";
import { Project } from "@/types/api/project.types";

export const formatProjectCurrency = (amount: number): string => {
  return new Intl.NumberFormat('vi-VN').format(amount) + ' đ';
};

export const getStatusColor = (status: string): string => {
  switch (status) {
    case "active":
      return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300";
    case "completed":
      return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300";
    case "draft":
      return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300";
    case "cancelled":
      return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300";
    case "paused":
      return "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300";
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
  if (amount >= 1000000000) {
    return `${Math.round(amount / 1000000000)}B`;
  }
  if (amount >= 1000000) {
    return `${Math.round(amount / 1000000)}M`;
  }
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
    return "text-blue-600 dark:text-blue-400";
  }
  if (count <= 20) {
    return "text-green-600 dark:text-green-400";
  }
  return "text-teal-600 dark:text-teal-400";
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
