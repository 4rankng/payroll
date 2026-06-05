import {
  Column,
  ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
  SortingState,
  OnChangeFn,
  Row,
} from "@tanstack/react-table";
import { CSSProperties, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ChevronDown, ChevronUp, ArrowUpDown, ArrowUp, ArrowDown, Inbox } from "lucide-react";
import { cn } from "@/lib/utils";
import { PaginationControls } from "@/components/ui/pagination-controls";

const toCssSize = (value?: number | string) =>
  value === undefined
    ? undefined
    : typeof value === "number"
    ? `${value}px`
    : value;

const getColumnSizingStyles = <TData, TValue>(column: Column<TData, TValue>): CSSProperties => {
  // Use minWidth so columns can grow to fill available space instead of causing overflow
  const declaredSize = column.columnDef.size;
  if (!declaredSize) return {};
  return { minWidth: toCssSize(declaredSize) };
};

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  mobileBreakpoint?: "sm" | "md" | "lg";
  primaryColumns?: string[]; // Columns to always show on mobile
  caption?: string; // WCAG accessibility
  className?: string;
  onRowClick?: (row: TData) => void;
  getRowClassName?: (row: TData) => string; // Custom row styling
  // External sorting props (server-side sorting)
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  // External pagination props (optional)
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  showPagination?: boolean; // Control whether to show pagination controls
  /** When true, removes the inner border/bg wrapper so the table sits flush
   *  inside an existing card/panel container. Pagination gets a top-border
   *  separator and horizontal padding instead. */
  embedded?: boolean;
}

// Mobile Card Row Component
function MobileCardRow<TData>({
  row,
  columns,
  primaryColumns,
  onRowClick,
  getRowClassName
}: {
  row: Row<TData>;
  columns: ColumnDef<TData, unknown>[];
  primaryColumns: string[];
  onRowClick?: (row: TData) => void;
  getRowClassName?: (row: TData) => string;
}) {
  const [isExpanded, setIsExpanded] = useState(false);

  const primaryCols = columns.filter((col) =>
    primaryColumns.includes((col as { accessorKey?: string }).accessorKey || col.id || "")
  );
  const secondaryCols = columns.filter((col) =>
    !primaryColumns.includes((col as { accessorKey?: string }).accessorKey || col.id || "")
  );

  return (
    <Card
      className={cn(
        "mb-3 border-border/40 rounded-2xl bg-card/90 backdrop-blur-sm shadow-sm transition-all duration-300 hover:shadow-md hover:border-border/80 touch-manipulation",
        onRowClick && "cursor-pointer",
        getRowClassName?.(row.original)
      )}
      onClick={(e) => {
        // Prevent row click if clicking on interactive elements
        const target = e.target as HTMLElement;
        const isInteractiveElement = target.closest('button') ||
          target.closest('[role="button"]') ||
          target.closest('a') ||
          target.closest('input') ||
          target.closest('select') ||
          target.closest('textarea');

        if (!isInteractiveElement) {
          onRowClick?.(row.original);
        }
      }}
    >
      <CardContent className="p-5 touch-manipulation">
        {/* Primary columns - always visible */}
        <div className="space-y-4">
          {primaryCols.map((column, index) => {
            const cell = row.getVisibleCells().find(c => c.column.id === column.id);
            if (!cell) return null;

            return (
              <div key={column.id} className={cn(
                "flex justify-between items-start gap-4 min-h-[24px]",
                index === 0 && "pb-3 border-b border-border/40"
              )}>
                <span className={cn(
                  "typography-body-medium flex-shrink-0 min-w-0 leading-relaxed",
                  index === 0 ? "typography-body-large text-foreground" : "text-muted-foreground font-medium"
                )}>
                  {typeof column.header === 'string' ? column.header : 'Field'}:
                </span>
                <div className={cn(
                  "text-right min-w-0 flex-1 leading-relaxed",
                  index === 0 && "typography-body-large"
                )}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </div>
              </div>
            );
          })}
        </div>

        {/* Secondary columns - collapsible */}
        {secondaryCols.length > 0 && (
          <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
            <CollapsibleTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="mt-4 w-full justify-between hover:bg-accent/50 min-h-[44px] rounded-lg"
                aria-expanded={isExpanded}
                aria-label={isExpanded ? "Ẩn thông tin chi tiết" : "Xem thông tin chi tiết"}
              >
                <span className="typography-body-medium">
                  {isExpanded ? "Ẩn chi tiết" : "Xem chi tiết"}
                </span>
                <div className="flex items-center gap-1">
                  {isExpanded ? (
                    <ChevronUp className="h-4 w-4" />
                  ) : (
                    <ChevronDown className="h-4 w-4" />
                  )}
                </div>
              </Button>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="pt-4 space-y-4 border-t border-border/40">
                {secondaryCols.map((column) => {
                  const cell = row.getVisibleCells().find(c => c.column.id === column.id);
                  if (!cell) return null;

                  return (
                    <div key={column.id} className="flex justify-between items-start gap-4 min-h-[24px]">
                      <span className="typography-body-medium text-muted-foreground font-medium flex-shrink-0 leading-relaxed">
                        {typeof column.header === 'string' ? column.header : 'Field'}:
                      </span>
                      <div className="typography-body-medium text-right min-w-0 flex-1 leading-relaxed">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </div>
                    </div>
                  );
                })}
              </div>
            </CollapsibleContent>
          </Collapsible>
        )}
      </CardContent>
    </Card>
  );
}

export function DataTable<TData, TValue>({
  columns,
  data,
  mobileBreakpoint = "md",
  primaryColumns = [],
  caption,
  className,
  onRowClick,
  getRowClassName,
  sorting: externalSorting,
  onSortingChange,
  pagination,
  onPageChange,
  onPageSizeChange,
  showPagination = true,
  embedded = false,
}: DataTableProps<TData, TValue>) {
  const sorting = externalSorting ?? [];

  // Ensure data is always an array to prevent undefined errors
  const safeData = data || [];

  const table = useReactTable({
    data: safeData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,
    enableSorting: !!onSortingChange,
    onSortingChange,
    // Use external pagination only if pagination prop is provided
    manualPagination: pagination ? true : false,
    pageCount: pagination?.totalPages ?? -1,
    state: {
      sorting,
      ...(pagination && {
        pagination: {
          pageIndex: pagination.page - 1, // TanStack uses 0-based indexing
          pageSize: pagination.pageSize,
        },
      }),
    },
  });

  const breakpointClass = {
    sm: "sm:block",
    md: "md:block",
    lg: "lg:block"
  }[mobileBreakpoint];

  return (
    <div className={cn(embedded ? "flex flex-col" : "space-y-4", className)}>
      {/* Desktop Table - Hidden on mobile */}
      <div className={cn("hidden", breakpointClass)}>
        <div className={cn(
          "relative overflow-x-auto scrollbar-thin scrollbar-thumb-muted scrollbar-track-transparent",
          !embedded && "rounded-2xl border border-border/40 bg-card/90 backdrop-blur-sm shadow-sm"
        )}>
          <Table className="w-full">
              {caption && <caption className="sr-only">{caption}</caption>}
              <TableHeader className="sticky top-0 bg-background/95 backdrop-blur-sm z-10">
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="border-b hover:bg-transparent">
                    {headerGroup.headers.map((header) => {
                      const canSort = header.column.getCanSort();
                      const sortDirection = header.column.getIsSorted();

                      return (
                        <TableHead
                          key={header.id}
                          scope="col"
                          className="whitespace-nowrap text-left typography-label-medium text-muted-foreground border-0"
                          style={getColumnSizingStyles(header.column)}
                        >
                          {header.isPlaceholder ? null : canSort ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="-ml-3 h-8 data-[state=open]:bg-accent hover:bg-accent/50"
                              onClick={() => header.column.toggleSorting(sortDirection === "asc")}
                            >
                              <span>
                                {flexRender(
                                  header.column.columnDef.header,
                                  header.getContext()
                                )}
                              </span>
                              {sortDirection === "desc" ? (
                                <ArrowDown className="ml-2 h-4 w-4" />
                              ) : sortDirection === "asc" ? (
                                <ArrowUp className="ml-2 h-4 w-4" />
                              ) : (
                                <ArrowUpDown className="ml-2 h-4 w-4" />
                              )}
                            </Button>
                          ) : (
                            flexRender(
                              header.column.columnDef.header,
                              header.getContext()
                            )
                          )}
                        </TableHead>
                      );
                    })}
                  </TableRow>
                ))}
              </TableHeader>
              <TableBody>
                {table.getRowModel().rows?.length ? (
                  table.getRowModel().rows.map((row) => (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && "selected"}
                      className={cn(
                        "hover:bg-muted/50 border-b",
                        onRowClick && "cursor-pointer hover:bg-accent/50"
                      )}
                      onClick={(e) => {
                        // Prevent row click if clicking on interactive elements
                        const target = e.target as HTMLElement;
                        const isInteractiveElement = target.closest('button') ||
                          target.closest('[role="button"]') ||
                          target.closest('a') ||
                          target.closest('input') ||
                          target.closest('select') ||
                          target.closest('textarea');

                        if (!isInteractiveElement) {
                          onRowClick?.(row.original);
                        }
                      }}
                    >
                      {row.getVisibleCells().map((cell, cellIndex) => (
                        <TableCell
                          key={cell.id}
                          className={cn(
                            "align-middle",
                            // Apply strip styling to first cell only
                            cellIndex === 0 && getRowClassName?.(row.original)
                          )}
                          style={getColumnSizingStyles(cell.column)}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="h-24 text-center">
                      Không có dữ liệu
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
        </div>
      </div>

      {/* Mobile Card View - Shown on mobile */}
      <div className={cn("block", breakpointClass.replace('block', 'hidden'))}>
        {table.getRowModel().rows?.length ? (
          <div className="space-y-4 px-1">
            {table.getRowModel().rows.map((row) => (
              <MobileCardRow
                key={row.id}
                row={row}
                columns={columns}
                primaryColumns={primaryColumns}
                onRowClick={onRowClick}
                getRowClassName={getRowClassName}
              />
            ))}
          </div>
        ) : (
          <Card className="mx-1">
            <CardContent className="p-8 text-center">
              <div className="flex flex-col items-center gap-3">
                <div className="w-12 h-12 rounded-full bg-muted flex items-center justify-center">
                  <Inbox className="w-6 h-6 text-muted-foreground" />
                </div>
                <p className="text-muted-foreground typography-body-medium">Không có dữ liệu</p>
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Pagination - Use PaginationControls component */}
      {showPagination && pagination && onPageChange && onPageSizeChange && (
        <PaginationControls
          pagination={pagination}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
          className={embedded ? "border-t border-border/60 px-4 sm:px-5 py-2.5" : undefined}
        />
      )}
    </div>
  );
}
