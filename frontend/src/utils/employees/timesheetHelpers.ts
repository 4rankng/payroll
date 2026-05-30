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
  return {
    id: record.id,
    amount: formatTimesheetCurrency(record.amount),
    projectName: 'projectName' in record ? record.projectName : `Project ${record.project_id}`,
    date: formatTimesheetDate(record.date),
    hours: formatTimesheetHours(record.hours_worked),
    paytype: record.paytype,
    rate: formatTimesheetRate(record.payrate),
    hourType: record.hour_type,
    dayType: record.day_type,
    status: record.status
  };
};