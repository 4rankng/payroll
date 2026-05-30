export interface TimesheetRecord {
  id: number;
  employeeName: string;
  workDays: number;
  actualDays: number;
  overtime: number;
  late: number;
  absent: number;
  totalHours: number;
}

export interface TimesheetStats {
  totalActualDays: number;
  totalOvertime: number;
  efficiency: number;
}