import type { PayRate, DayType } from "@/types/api/payrate.types";
import type {
  TimesheetFormData,
  TimesheetFormOptions,
} from "@/types/timesheet";
import {
  getPositionsFromRates,
  getAllHourTypes,
} from "@/components/payrates/types";
import { requiresDayType } from "@/utils/timesheetHelpers";

export const WEEKDAY = ["CN", "T2", "T3", "T4", "T5", "T6", "T7"] as const;

export const isWeekendDate = (s: string): boolean => {
  const d = new Date(s).getDay();
  return d === 0 || d === 6;
};

export const getHourStatus = (
  totalHours: number,
): "normal" | "exceeded" | "excessive" => {
  if (totalHours > 16) return "excessive";
  if (totalHours > 12) return "exceeded";
  return "normal";
};

export const getDayTypePreview = (date: string): DayType => {
  const dateObj = new Date(date);
  const dayOfWeek = dateObj.getDay();

  // Sunday (0)
  if (dayOfWeek === 0) {
    return "ngày nghỉ";
  }

  // Saturday (6) - will be handled by user selection in actual form
  if (dayOfWeek === 6) {
    return "ngày thường"; // Default for preview, actual selection handled elsewhere
  }

  // Monday-Friday (1-5)
  return "ngày thường";
};

export const calculateFormOptions = (
  payRateData: PayRate | undefined,
  formData: TimesheetFormData,
): TimesheetFormOptions => {
  if (!payRateData?.rates) {
    return {
      positions: [],
      hourTypes: [],
      dayTypePreview: getDayTypePreview(formData.date),
    };
  }

  const positions = getPositionsFromRates(payRateData.rates);
  const hourTypes = getAllHourTypes(payRateData.rates);
  const dayTypePreview = getDayTypePreview(formData.date);

  let ratePreview: number | undefined;
  if (
    formData.position &&
    formData.hour_type &&
    payRateData.rates[formData.position]
  ) {
    const dayRates = payRateData.rates[formData.position][dayTypePreview];
    if (dayRates && dayRates[formData.hour_type]) {
      ratePreview = dayRates[formData.hour_type];
    }
  }

  return {
    positions,
    hourTypes,
    dayTypePreview,
    ratePreview,
  };
};

export const createTimesheetEntry = (
  formData: TimesheetFormData,
  saturdayDayType: "Ngày thường" | "Ngày nghỉ",
) => {
  const entry: Record<string, unknown> = {
    projectId: formData.project_id,
    employeeId: formData.employee_id,
    date: formData.date,
    hoursWorked: formData.hours_worked,
    hourType: formData.hour_type,
  };

  if (requiresDayType(formData.date)) {
    entry.dayType = saturdayDayType;
  }

  return entry;
};

export const resetFormForNewEntry = (
  currentData: TimesheetFormData,
): TimesheetFormData => ({
  ...currentData,
  employee_id: 0,
  date: new Date().toISOString().split("T")[0],
  hours_worked: 8,
  notes: "",
});

interface TimesheetData {
  employee_id: number;
  project_id: number;
  date: string;
  hours_worked: number;
  position: string;
  hour_type: string;
}

export const createInitialFormData = (
  timesheet?: TimesheetData,
  employeeId?: number,
  projectId?: number,
): TimesheetFormData => {
  if (timesheet) {
    return {
      employee_id: timesheet.employee_id,
      project_id: timesheet.project_id,
      date: timesheet.date,
      hours_worked: timesheet.hours_worked,
      position: timesheet.position,
      hour_type: timesheet.hour_type,
      notes: "",
    };
  }

  return {
    employee_id: employeeId || 0,
    project_id: projectId || 0,
    date: new Date().toISOString().split("T")[0],
    hours_worked: 8,
    position: "",
    hour_type: "",
    notes: "",
  };
};
