import { format } from "date-fns";
import { formatCurrency } from "@/utils/employeeHelpers";
import type { Timesheet } from "@/types/api/timesheet.types";
import type { EmployeeTimesheetEntry } from "@/types/api/employee.types";

export const formatTimesheetDate = (dateString: string): string => {
  return format(new Date(dateString), 'dd/MM/yyyy');
};

export const formatTimesheetCurrency = (amount: number): string => {
  return formatCurrency(amount);
};

export const formatTimesheetHours = (hours: number): string => {
  return `${hours} giờ`;
};

export const formatTimesheetRate = (rate: number): string => {
  return `${formatCurrency(rate)}/giờ`;
};

export const getTimesheetDisplayData = (record: Timesheet | EmployeeTimesheetEntry) => {
  const rec = record as unknown as {
    id: number;
    amount: number;
    projectName?: string;
    project?: { name?: string; id?: number };
    project_id?: number;
    date: string;
    hours_worked: number;
    paytype: string;
    rate?: number;
    hourType?: string;
    dayType?: string;
    status: string;
    payment_status?: string;
    paid_amount?: number;
    position?: string;
    hour_type?: string;
    day_type?: string;
  };
  return {
    id: rec.id,
    amount: formatTimesheetCurrency(rec.amount),
    projectName: rec.projectName || rec.project?.name || `Project ${rec.project_id || rec.project?.id || ''}`,
    date: formatTimesheetDate(rec.date),
    hours: formatTimesheetHours(rec.hours_worked),
    paytype: rec.paytype,
    rate: formatTimesheetRate(rec.payrate),
    hourType: rec.hour_type,
    dayType: rec.day_type,
    status: rec.status
  };
};