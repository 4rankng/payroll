import { type ColumnDef } from "@tanstack/react-table";
import { format } from "date-fns";
import { formatCurrency } from "@/utils/formatters";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";

export function getAdminAttendanceColumns(): ColumnDef<AdminAttendanceResponse>[] {
  return [
    {
      accessorKey: "employee_name",
      header: "Nhân viên",
      size: 150,
      cell: ({ row }) => (
        <div>
          <div className="text-xs font-medium leading-tight">{row.original.employee_name}</div>
          <div className="text-xs text-muted-foreground/70 mt-0.5 truncate max-w-[140px]">
            {row.original.project_name || "-"}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "date",
      header: "Ngày",
      size: 100,
      cell: ({ row }) => {
        try {
          return <span className="text-xs font-medium">{format(new Date(row.original.date), "dd/MM/yyyy")}</span>;
        } catch {
          return <span className="text-xs">{row.original.date}</span>;
        }
      },
    },
    {
      id: "check_in",
      header: "Check-in",
      size: 130,
      cell: ({ row }) => {
        if (!row.original.check_in_time) return <span className="text-xs text-muted-foreground">-</span>;
        try {
          return (
            <div>
              <div className="text-xs font-medium text-slate-700">{format(new Date(row.original.check_in_time), "HH:mm")}</div>
              <div className="text-[10px] text-muted-foreground truncate max-w-[120px]">{row.original.check_in_gate || "Chưa xác định"}</div>
            </div>
          );
        } catch {
          return <span className="text-xs">{row.original.check_in_time}</span>;
        }
      }
    },
    {
      id: "check_out",
      header: "Check-out",
      size: 130,
      cell: ({ row }) => {
        if (!row.original.check_out_time) return <span className="text-xs text-muted-foreground">-</span>;
        try {
          return (
            <div>
              <div className="text-xs font-medium text-slate-700">{format(new Date(row.original.check_out_time), "HH:mm")}</div>
              <div className="text-[10px] text-muted-foreground truncate max-w-[120px]">{row.original.check_out_gate || "Chưa xác định"}</div>
            </div>
          );
        } catch {
          return <span className="text-xs">{row.original.check_out_time}</span>;
        }
      }
    },
    {
      accessorKey: "earning_amount",
      header: "Thu nhập",
      size: 110,
      cell: ({ row }) => {
        const amount = row.original.earning_amount;
        if (amount == null) return <span className="text-xs text-muted-foreground">-</span>;
        return <span className="text-xs font-bold text-[#00B14F]">{formatCurrency(amount)}</span>;
      },
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      size: 110,
      cell: ({ row }) => {
        const s = row.original.status;
        let badgeClass = "bg-gray-100 text-gray-600 border-gray-200";
        let label = "Không rõ";
        
        if (s === "checked_in") {
          badgeClass = "bg-blue-50 text-blue-700 border-blue-200";
          label = "Đang làm";
        } else if (s === "completed") {
          badgeClass = "bg-green-50 text-green-700 border-green-200";
          label = "Hoàn thành";
        } else if (s === "orphaned") {
          badgeClass = "bg-red-50 text-red-700 border-red-200";
          label = "Thiếu Check-out";
        } else if (s === "rejected") {
          badgeClass = "bg-orange-50 text-orange-700 border-orange-200";
          label = "Đã từ chối";
        }

        return (
          <div className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold border ${badgeClass}`}>
            {label}
          </div>
        );
      },
    },
  ];
}

export const attendanceMobileFields = [
  {
    key: "date",
    label: "Ngày",
    render: (row: AdminAttendanceResponse) => {
      try {
        return format(new Date(row.date), "dd/MM/yyyy");
      } catch {
        return row.date;
      }
    }
  },
  {
    key: "check_in_time",
    label: "Check-in",
    render: (row: AdminAttendanceResponse) => row.check_in_time ? format(new Date(row.check_in_time), "HH:mm") : "-"
  },
  {
    key: "check_out_time",
    label: "Check-out",
    render: (row: AdminAttendanceResponse) => row.check_out_time ? format(new Date(row.check_out_time), "HH:mm") : "-"
  },
  {
    key: "earning_amount",
    label: "Thu nhập",
    render: (row: AdminAttendanceResponse) => row.earning_amount != null ? formatCurrency(row.earning_amount) : "-"
  }
];

export const attendanceEmptyState = {
  title: "Chưa có dữ liệu chấm công",
  description: "Không tìm thấy bản ghi chấm công nào phù hợp với bộ lọc hiện tại.",
};
