import { Button } from "@/components/ui/button";

interface TimesheetTablePaginationProps {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
  onPageChange: (page: number) => void;
  label?: string;
}

export function TimesheetTablePagination({
  page,
  pageSize,
  totalPages,
  totalRecords,
  onPageChange,
  label = "nhân viên",
}: TimesheetTablePaginationProps) {
  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-between">
      <p className="text-xs text-muted-foreground tabular-nums">
        {Math.min((page - 1) * pageSize + 1, totalRecords)}–
        {Math.min(page * pageSize, totalRecords)} / {totalRecords} {label}
      </p>
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="sm"
          className="h-8 px-3 text-xs"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
        >
          Trước
        </Button>
        <span className="px-2 text-xs font-medium tabular-nums text-muted-foreground">
          {page} / {totalPages}
        </span>
        <Button
          variant="outline"
          size="sm"
          className="h-8 px-3 text-xs"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
        >
          Sau
        </Button>
      </div>
    </div>
  );
}
