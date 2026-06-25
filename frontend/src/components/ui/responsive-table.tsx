import React from "react";
import { useMediaQuery } from '@/hooks/useBreakpoint';
import { DataTable } from "@/components/ui/data-table";
import { MobileTable, type MobileField, type RowAction } from "@/components/ui/mobile-table";
import { ColumnDef, SortingState, OnChangeFn } from "@tanstack/react-table";
import { cn } from "@/lib/utils";

interface ResponsiveTableProps<TData = Record<string, unknown>> {
  // Common props
  data: TData[];
  className?: string;

  // Desktop table props
  columns: ColumnDef<TData>[];

  // Mobile table props
  mobileFields: MobileField<TData>[];
  rowTitle?: (row: TData) => React.ReactNode;
  rowSubtitle?: (row: TData) => React.ReactNode;
  rowActions?: RowAction<TData>[];
  getRowId?: (row: TData) => string;
  onRowClick?: (row: TData) => void;

  // Search functionality
  searchKey?: string;
  searchPlaceholder?: string;

  // Pagination props (external/controlled pagination)
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;

  // Server-side sorting props
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;

  // Optional props
  emptyState?: React.ReactNode;
  accordionType?: "single" | "multiple";
  showPagination?: boolean;
  caption?: string;
  getRowClassName?: (row: TData) => string;
  /** Pass true when the table sits inside an existing card/panel so the
   *  inner border wrapper is removed and pagination gets a flush separator. */
  embedded?: boolean;

  // Breakpoint for switching (default: lg = 1024px)
  breakpoint?: "sm" | "md" | "lg" | "xl";
}

export function ResponsiveTable<TData = Record<string, unknown>>({
  data,
  className,
  columns,
  mobileFields,
  rowTitle,
  rowSubtitle,
  rowActions,
  getRowId,
  onRowClick,
  pagination,
  onPageChange,
  onPageSizeChange,
  sorting,
  onSortingChange,
  emptyState,
  accordionType = "single",
  showPagination = true,
  breakpoint = "lg",
  caption,
  getRowClassName,
  embedded = false,
}: ResponsiveTableProps<TData>) {
  // Determine breakpoint value
  const breakpointQuery = {
    sm: "(min-width: 640px)",
    md: "(min-width: 768px)",
    lg: "(min-width: 1024px)",
    xl: "(min-width: 1280px)",
  }[breakpoint];

  const isDesktop = useMediaQuery(breakpointQuery);

  // Render desktop table for tablets and above
  if (isDesktop) {
    return (
      <div className={cn("w-full", className)}>
        <DataTable
          columns={columns}
          data={data}
          onRowClick={onRowClick}
          pagination={pagination}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
          showPagination={showPagination}
          sorting={sorting}
          onSortingChange={onSortingChange}
          caption={caption}
          getRowClassName={getRowClassName}
          embedded={embedded}
        />
      </div>
    );
  }

  // Render mobile table for phones
  return (
    <div className={cn("w-full", className)}>
      <MobileTable
        data={data as unknown as Record<string, unknown>[]}
        fields={mobileFields as unknown as MobileField<Record<string, unknown>>[]}
        rowTitle={rowTitle as unknown as (row: Record<string, unknown>) => React.ReactNode}
        rowSubtitle={rowSubtitle as unknown as (row: Record<string, unknown>) => React.ReactNode}
        rowActions={rowActions as unknown as RowAction<Record<string, unknown>>[]}
        getRowId={getRowId as unknown as (row: Record<string, unknown>) => string}
        onRowClick={onRowClick as unknown as (row: Record<string, unknown>) => void}
        emptyState={emptyState}
        accordionType={accordionType}
        pagination={pagination}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
      />
    </div>
  );
}
// Re-export types for convenience
export type { MobileField, RowAction } from "@/components/ui/mobile-table";
