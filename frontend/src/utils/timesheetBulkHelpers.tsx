import { ReactNode } from 'react';
import { Timesheet } from '@/types/api/timesheet.types';
import { formatDate } from '@/components/timesheet/utils/timesheetHelpers';
import { formatCurrency } from '@/utils/formatters';

export interface BulkApprovalStats {
  totalCount: number;
  uniqueEmployeeCount: number;
  uniqueProjectCount: number;
  totalHours: number;
  totalAmount: number;
  employeeNames: string[];
  projectNames: string[];
}

export function calculateBulkStats(timesheets: Timesheet[]): BulkApprovalStats {
  const uniqueEmployees = new Set<string>();
  const uniqueProjects = new Set<string>();
  const employeeNames = new Set<string>();
  const projectNames = new Set<string>();

  let totalHours = 0;
  let totalAmount = 0;

  timesheets.forEach(timesheet => {
    if (timesheet.employee_id) uniqueEmployees.add(timesheet.employee_id.toString());
    if (timesheet.project_id) uniqueProjects.add(timesheet.project_id.toString());
    if (timesheet.employeeName) employeeNames.add(timesheet.employeeName);
    if (timesheet.projectName) projectNames.add(timesheet.projectName);

    totalHours += timesheet.hours_worked || 0;
    totalAmount += timesheet.amount || 0;
  });

  return {
    totalCount: timesheets.length,
    uniqueEmployeeCount: uniqueEmployees.size,
    uniqueProjectCount: uniqueProjects.size,
    totalHours,
    totalAmount,
    employeeNames: Array.from(employeeNames),
    projectNames: Array.from(projectNames)
  };
}

export function createGlobalBulkApprovalContent(
  pendingTimesheets: Timesheet[]
): ReactNode {
  const stats = calculateBulkStats(pendingTimesheets);

  return (
    <div className="space-y-3">
      <p>Bạn sắp duyệt tất cả <strong>{stats.totalCount}</strong> bảng công đang chờ duyệt</p>

      <div className="bg-blue-50 p-3 rounded border border-blue-200">
        <div className="typography-body-medium space-y-1">
          <div className="flex items-center gap-2">
            <span>📊</span>
            <span>Tổng số bản ghi: <strong>{stats.totalCount}</strong></span>
          </div>
          <div className="flex items-center gap-2">
            <span>👥</span>
            <span>Nhân viên: <strong>{stats.uniqueEmployeeCount}</strong></span>
          </div>
          <div className="flex items-center gap-2">
            <span>🏢</span>
            <span>Dự án: <strong>{stats.uniqueProjectCount}</strong></span>
          </div>
          <div className="flex items-center gap-2">
            <span>⏱️</span>
            <span>Tổng giờ: <strong>{stats.totalHours} giờ</strong></span>
          </div>
          <div className="flex items-center gap-2">
            <span>💰</span>
            <span>Tổng tiền: <strong>{formatCurrency(stats.totalAmount)}</strong></span>
          </div>
        </div>
      </div>

      <div className="bg-amber-50 p-3 rounded border border-amber-200">
        <div className="flex items-start gap-2 typography-body-medium text-amber-700">
          <span className="typography-body-large">⚠️</span>
          <span>Tất cả bảng công sẽ chuyển sang trạng thái "Đã duyệt". Hành động này không thể hoàn thành.</span>
        </div>
      </div>
    </div>
  );
}

export function createProjectBulkApprovalContent(
  projectName: string,
  pendingTimesheets: Timesheet[]
): ReactNode {
  const stats = calculateBulkStats(pendingTimesheets);

  return (
    <div className="space-y-3">
      <p>Bạn có chắc chắn muốn duyệt tất cả bảng công pending cho:</p>

      <div className="bg-muted p-3 rounded border">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <span className="typography-title-large">🏢</span>
            <span className="font-medium">Dự án: {projectName}</span>
          </div>
          <div className="typography-body-medium text-muted-foreground space-y-1">
            <div>📊 {stats.totalCount} bảng công sẽ được duyệt</div>
            <div>👥 {stats.uniqueEmployeeCount} nhân viên</div>
            <div>⏱️ {stats.totalHours} giờ làm việc</div>
            <div>💰 {formatCurrency(stats.totalAmount)}</div>
          </div>
        </div>
      </div>

      {stats.employeeNames.length <= 5 && (
        <div className="bg-blue-50 p-2 rounded typography-body-small">
          <div className="font-medium mb-1">Nhân viên:</div>
          <div className="text-blue-700">{stats.employeeNames.join(', ')}</div>
        </div>
      )}

      <div className="bg-amber-50 p-2 rounded">
        <div className="flex items-start gap-2 typography-body-small text-amber-700">
          <span>⚠️</span>
          <span>Hành động này không thể hoàn thành</span>
        </div>
      </div>
    </div>
  );
}

export function createGroupBulkApprovalContent(
  employeeName: string,
  employeeCode: string,
  date: string,
  entries: Timesheet[]
): ReactNode {
  const stats = calculateBulkStats(entries);

  return (
    <div className="space-y-3">
      <p>Duyệt tất cả bảng công cho:</p>

      <div className="bg-muted p-3 rounded border">
        <div className="space-y-2 typography-body-medium">
          <div className="flex items-center gap-2">
            <span>👤</span>
            <span className="font-medium">{employeeName}</span>
            <span className="text-muted-foreground">({employeeCode})</span>
          </div>
          <div className="flex items-center gap-2">
            <span>📅</span>
            <span>{formatDate(date)}</span>
          </div>
          <div className="grid grid-cols-2 gap-4 pt-2 border-t">
            <div>
              <div className="typography-body-small text-muted-foreground">Số bản ghi</div>
              <div className="font-medium">{stats.totalCount}</div>
            </div>
            <div>
              <div className="typography-body-small text-muted-foreground">Tổng giờ</div>
              <div className="font-medium text-green-600">{stats.totalHours} giờ</div>
            </div>
            <div>
              <div className="typography-body-small text-muted-foreground">Thành tiền</div>
              <div className="font-medium">{formatCurrency(stats.totalAmount)}</div>
            </div>
            <div>
              <div className="typography-body-small text-muted-foreground">Dự án</div>
              <div className="font-medium">{stats.uniqueProjectCount}</div>
            </div>
          </div>
        </div>
      </div>

      {stats.projectNames.length > 1 && (
        <div className="bg-blue-50 p-2 rounded typography-body-small">
          <div className="font-medium mb-1">Các dự án:</div>
          <div className="text-blue-700">{stats.projectNames.join(', ')}</div>
        </div>
      )}
    </div>
  );
}

export function createBulkRejectContent(
  context: 'global' | 'project' | 'group',
  timesheets: Timesheet[],
  contextInfo?: {
    projectName?: string;
    employeeName?: string;
    employeeCode?: string;
    date?: string;
  }
): ReactNode {
  const stats = calculateBulkStats(timesheets);

  const getTitle = () => {
    switch (context) {
      case 'global':
        return `tất cả ${stats.totalCount} bảng công pending`;
      case 'project':
        return `tất cả bảng công pending của dự án "${contextInfo?.projectName}"`;
      case 'group':
        return `tất cả bảng công của ${contextInfo?.employeeName} ngày ${formatDate(contextInfo?.date || '')}`;
      default:
        return 'các bảng công được chọn';
    }
  };

  return (
    <div className="space-y-3">
      <p>Bạn có chắc chắn muốn loại {getTitle()}?</p>

      <div className="bg-red-50 p-3 rounded border border-red-200">
        <div className="typography-body-medium space-y-1 text-red-700">
          <div>📊 {stats.totalCount} bảng công sẽ bị loại</div>
          <div>👥 {stats.uniqueEmployeeCount} nhân viên bị ảnh hưởng</div>
          <div>⏱️ {stats.totalHours} giờ làm việc</div>
          <div>💰 {formatCurrency(stats.totalAmount)}</div>
        </div>
      </div>

      <div className="bg-amber-50 p-3 rounded border border-amber-200">
        <div className="flex items-start gap-2 typography-body-medium text-amber-700">
          <span>⚠️</span>
          <span>Nhân viên sẽ được thông báo về việc loại. Hành động này không thể hoàn thành.</span>
        </div>
      </div>
    </div>
  );
}
