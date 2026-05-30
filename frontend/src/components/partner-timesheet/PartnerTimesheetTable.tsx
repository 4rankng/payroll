import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { FileUp, Download, Clock, Calendar, User } from "lucide-react";
import type { Row } from "@tanstack/react-table";

interface TimesheetRecord {
  id: number;
  employeeName: string;
  workDays: number;
  actualDays: number;
  overtime: number;
  late: number;
  absent: number;
  totalHours: number;
}

interface PartnerTimesheetTableProps {
  timesheetData: TimesheetRecord[];
  selectedMonth: string;
  onMonthChange: (month: string) => void;
  onUpload: () => void;
}

export const PartnerTimesheetTable = ({
  timesheetData,
  selectedMonth,
  onMonthChange,
  onUpload
}: PartnerTimesheetTableProps) => {
  const columns = [
    {
      accessorKey: "employeeName",
      header: "Nhân viên",
      cell: ({ row }: { row: Row<TimesheetRecord> }) => (
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
            <User className="w-4 h-4 text-primary" />
          </div>
          <p className="typography-label-large text-foreground">{row.getValue("employeeName")}</p>
        </div>
      ),
    },
    {
      accessorKey: "workDays",
      header: "Ngày làm",
      cell: ({ row }: { row: Row<TimesheetRecord> }) => (
        <div className="text-center">
          <span className="typography-data typography-label-medium">{row.getValue("workDays")}</span>
        </div>
      ),
    },
    {
      accessorKey: "actualDays",
      header: "Ngày thực tế",
      cell: ({ row }: { row: Row<TimesheetRecord> }) => (
        <div className="text-center">
          <span className="typography-data typography-label-medium text-financial-approved">{row.getValue("actualDays")}</span>
        </div>
      ),
    },
    {
      accessorKey: "overtime",
      header: "OT (giờ)",
      cell: ({ row }: { row: Row<TimesheetRecord> }) => (
        <div className="text-center">
          <span className="typography-data typography-label-medium text-financial-pending">{row.getValue("overtime")} giờ</span>
        </div>
      ),
    },
    {
      accessorKey: "totalHours",
      header: "Tổng giờ",
      cell: ({ row }: { row: Row<TimesheetRecord> }) => (
        <div className="text-center">
          <span className="typography-data typography-label-large text-foreground">{row.getValue("totalHours")} giờ</span>
        </div>
      ),
    },
  ];

  const mobileFields = [
    {
      key: 'employeeName',
      label: 'Nhân viên',
      priority: 1,
      render: (row: TimesheetRecord) => (
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center">
            <User className="w-4 h-4 text-primary" />
          </div>
          <p className="typography-body-medium">{row.employeeName}</p>
        </div>
      )
    },
    {
      key: 'actualDays',
      label: 'Ngày làm',
      priority: 1,
      render: (row: TimesheetRecord) => (
        <div className="flex items-center gap-2">
          <Calendar className="w-3 h-3 text-financial-approved" />
          <span className="typography-data typography-body-medium text-financial-approved">{row.actualDays}/{row.workDays}</span>
        </div>
      )
    },
    {
      key: 'totalHours',
      label: 'Tổng giờ',
      priority: 1,
      render: (row: TimesheetRecord) => (
        <div className="flex items-center gap-2">
          <Clock className="w-3 h-3 text-primary" />
          <span className="typography-data typography-body-medium text-foreground">{row.totalHours} giờ</span>
        </div>
      )
    },
    {
      key: 'overtime',
      label: 'Tăng ca',
      priority: 2,
      render: (row: TimesheetRecord) => (
        <div className="flex items-center gap-2">
          <div className="p-1 bg-financial-pending/10 rounded">
            <Clock className="w-3 h-3 text-financial-pending" />
          </div>
          <span className="typography-data typography-body-medium text-financial-pending">{row.overtime} giờ</span>
        </div>
      )
    },
    {
      key: 'late',
      label: 'Muộn',
      priority: 3,
      render: (row: TimesheetRecord) => (
        <span className="typography-data typography-body-medium text-financial-negative">{row.late} lần</span>
      )
    },
    {
      key: 'absent',
      label: 'Nghỉ',
      priority: 3,
      render: (row: TimesheetRecord) => (
        <span className="typography-data typography-body-medium text-financial-negative">{row.absent} ngày</span>
      )
    }
  ];

  return (
    <Card className="bg-gradient-to-br from-white to-slate-50 border-0 shadow-xl">
      <CardHeader>
        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <CardTitle>Bảng công tháng {selectedMonth}</CardTitle>
            <CardDescription>
              Quản lý thông tin chấm công nhân viên
            </CardDescription>
          </div>
          <div className="flex flex-col sm:flex-row gap-3">
            <Select value={selectedMonth} onValueChange={onMonthChange}>
              <SelectTrigger className="w-full sm:w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="2025-08">Tháng 8/2025</SelectItem>
                <SelectItem value="2025-07">Tháng 7/2025</SelectItem>
              </SelectContent>
            </Select>
            <Button variant="outline" onClick={onUpload} className="w-full sm:w-auto">
              <FileUp className="w-4 h-4 mr-2" />
              Nhập công Excel
            </Button>
            <Button variant="outline" className="w-full sm:w-auto">
              <Download className="w-4 h-4 mr-2" />
              Tải mẫu
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ResponsiveTable
          data={timesheetData}
          columns={columns}
          searchKey="employeeName"
          searchPlaceholder="Tìm kiếm nhân viên..."
          mobileFields={mobileFields}
          rowTitle={(row: TimesheetRecord) => (
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-full flex items-center justify-center">
                <User className="w-5 h-5 text-white" />
              </div>
              <div>
                <p className="typography-body-large">{row.employeeName}</p>
                <p className="typography-body-medium text-muted-foreground">{row.actualDays}/{row.workDays} ngày</p>
              </div>
            </div>
          )}
          rowSubtitle={(row: TimesheetRecord) => (
            <div className="flex items-center gap-4 typography-body-medium text-muted-foreground">
              <div className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                <span>{row.totalHours} giờ</span>
              </div>
              <div className="flex items-center gap-1">
                <span className="text-orange-600">OT: {row.overtime} giờ</span>
              </div>
            </div>
          )}
          getRowId={(row: TimesheetRecord) => String(row.id)}
          emptyState={
            <div className="text-center py-12">
              <Calendar className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h3 className="mt-4 typography-title-large">Không có dữ liệu bảng công</h3>
              <p className="mt-2 typography-body-medium text-muted-foreground">
                Hãy nhập bảng công để bắt đầu quản lý.
              </p>
            </div>
          }
        />
      </CardContent>
    </Card>
  );
};
