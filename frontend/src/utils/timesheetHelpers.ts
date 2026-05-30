import { TimesheetItem } from "@/types/timesheet";

export const getTimesheetStatusColor = (status: string): string => {
  switch (status) {
    case "Đã duyệt":
      return "bg-success text-white";
    case "Chờ duyệt":
      return "bg-warning text-white";
    case "Cần sửa":
      return "bg-destructive text-white";
    case "Loại":
      return "bg-destructive text-white";
    default:
      return "bg-secondary text-foreground";
  }
};

export const calculateTimesheetStats = (timesheets: TimesheetItem[]) => {
  if (!timesheets || !Array.isArray(timesheets)) {
    return {
      totalActualDays: 0,
      totalOvertime: 0,
      pendingApproval: 0,
      efficiency: 0,
      totalEmployees: 0,
    };
  }

  const totalActualDays = timesheets.reduce((sum, item) => {
    const days = Number(item?.actualDays || 0);
    return sum + (isNaN(days) ? 0 : days);
  }, 0);

  const totalOvertime = timesheets.reduce((sum, item) => {
    const overtime = Number(item?.overtime || 0);
    return sum + (isNaN(overtime) ? 0 : overtime);
  }, 0);

  const pendingApproval = timesheets.filter(item => item?.status === "Chờ duyệt").length;

  // Calculate efficiency based on actual vs expected work days
  const expectedDays = timesheets.reduce((sum, item) => {
    const days = Number(item?.workDays || 0);
    return sum + (isNaN(days) ? 0 : days);
  }, 0);

  const efficiency = expectedDays > 0 ? Math.round((totalActualDays / expectedDays) * 100) : 0;

  return {
    totalActualDays: isNaN(totalActualDays) ? 0 : totalActualDays,
    totalOvertime: isNaN(totalOvertime) ? 0 : totalOvertime,
    pendingApproval: isNaN(pendingApproval) ? 0 : pendingApproval,
    efficiency: isNaN(efficiency) ? 0 : efficiency,
    totalEmployees: timesheets.length,
  };
};

export const formatTimeDisplay = (hours: number): string => {
  return `${hours}h`;
};

export const formatDaysDisplay = (days: number): string => {
  return `${days} ngày`;
};

export const calculateTotalHours = (actualDays: number, overtime: number): number => {
  // Assuming 8 hours per work day
  return (actualDays * 8) + overtime;
};

/**
 * Determines if a date is a Saturday
 * @param dateString Date in YYYY-MM-DD format
 * @returns true if the date is Saturday
 */
export const isSaturday = (dateString: string): boolean => {
  const date = new Date(dateString);
  return date.getDay() === 6;
};

/**
 * Determines if a date is a Sunday
 * @param dateString Date in YYYY-MM-DD format
 * @returns true if the date is Sunday
 */
export const isSunday = (dateString: string): boolean => {
  const date = new Date(dateString);
  return date.getDay() === 0;
};

/**
 * Determines if a date requires dayType specification
 * @param dateString Date in YYYY-MM-DD format
 * @returns true if dayType should be specified (Saturday)
 */
export const requiresDayType = (dateString: string): boolean => {
  return isSaturday(dateString);
};
