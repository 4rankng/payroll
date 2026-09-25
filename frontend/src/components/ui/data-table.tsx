import {
  Column,
  ColumnDef,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
  SortingState,
  OnChangeFn,
  Row,
  VisibilityState,
} from "@tanstack/react-table";
import { CSSProperties, useEffect, useRef, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ChevronDown, ChevronUp, ArrowUpDown, ArrowUp, ArrowDown, Columns3 } from "lucide-react";
import { cn } from "@/lib/utils";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { EmptyState } from "@/components/shared/EmptyState";

const COLUMN_VISIBILITY_STORAGE_PREFIX = "table-cols:";

function readPersistedColumnVisibility(key: string): VisibilityState {
  try {
    const raw = localStorage.getItem(COLUMN_VISIBILITY_STORAGE_PREFIX + key);
    return raw ? (JSON.parse(raw) as VisibilityState) : {};
  } catch {
    return {};
  }
}

function persistColumnVisibility(key: string, visibility: VisibilityState) {
  try {
    localStorage.setItem(COLUMN_VISIBILITY_STORAGE_PREFIX + key, JSON.stringify(visibility));
  } catch {
    // Private-mode / quota errors: persistence is best-effort.
  }
}

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
  emptyState?: React.ReactNode;
  /** "dense" = data-first tablet/desktop mode: 12px text, 40px rows, tighter
   *  padding. "comfortable" (default) keeps the current airy spacing. */
  density?: "comfortable" | "dense";
  /** Freezes the first (identifier) column: sticky left + opaque background
   *  so row context survives horizontal scrolling (NN/g mobile-tables). */
  stickyFirstColumn?: boolean;
  /** When set, shows a column show/hide menu persisted to localStorage under
   *  this key (per user, per table). */
  columnVisibilityKey?: string;
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
  emptyState,
  density = "comfortable",
  stickyFirstColumn = false,
  columnVisibilityKey,
}: DataTableProps<TData, TValue>) {
  const sorting = externalSorting ?? [];
  const isDense = density === "dense";
  const scrollRef = useRef<HTMLDivElement>(null);
  const [isScrolledHorizontally, setIsScrolledHorizontally] = useState(false);

  // Column show/hide, persisted per user + table when columnVisibilityKey is set.
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(() =>
    columnVisibilityKey ? readPersistedColumnVisibility(columnVisibilityKey) : {}
  );
  useEffect(() => {
    if (columnVisibilityKey) {
      persistColumnVisibility(columnVisibilityKey, columnVisibility);
    }
  }, [columnVisibility, columnVisibilityKey]);

  const handleScroll = (event: React.UIEvent<HTMLDivElement>) => {
    const el = event.currentTarget;
    setIsScrolledHorizontally(el.scrollWidth > el.clientWidth + 1 && el.scrollLeft < el.scrollWidth - el.clientWidth - 1);
  };

  // Ensure data is always an array to prevent undefined errors
  const safeData = data || [];

  const table = useReactTable({
    data: safeData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    manualSorting: true,
    enableSorting: !!onSortingChange,
    enableHiding: !!columnVisibilityKey,
    onSortingChange,
    state: {
      sorting,
      columnVisibility,
      ...(pagination && {
        pagination: {
          pageIndex: pagination.page - 1, // TanStack uses 0-based indexing
          pageSize: pagination.pageSize,
        },
      }),
    },
    onColumnVisibilityChange: setColumnVisibility,
    // Use external pagination only if pagination prop is provided
    manualPagination: pagination ? true : false,
    pageCount: pagination?.totalPages ?? -1,
  });

  const breakpointClass = {
    sm: "sm:block",
    md: "md:block",
    lg: "lg:block"
  }[mobileBreakpoint];

  const cellPadding = isDense ? "px-2.5 py-1.5" : "px-4 py-3";
  const headerPadding = isDense ? "px-2.5 py-2" : "px-4 py-3";

  return (
    <div
      data-slot="data-table"
      className={cn(embedded ? "flex flex-col" : "space-y-3", className)}
    >
      {columnVisibilityKey && (
        <div className="flex items-center justify-end">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="outline"
                size="sm"
                className="h-9 gap-2 text-xs"
                aria-label="Chọn cột hiển thị"
              >
                <Columns3 className="h-4 w-4" />
                Cột
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="max-h-72 overflow-y-auto">
              {table
                .getAllLeafColumns()
                .filter((col) => col.getCanHide() && col.id !== "actions")
                .map((col) => (
                  <DropdownMenuCheckboxItem
                    key={col.id}
                    checked={col.getIsVisible()}
                    onCheckedChange={(value) => col.toggleVisibility(!!value)}
                    onSelect={(e) => e.preventDefault()}
                    className="text-xs"
                  >
                    {typeof col.columnDef.header === "string" ? col.columnDef.header : col.id}
                  </DropdownMenuCheckboxItem>
                ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
      <div className={cn("hidden", breakpointClass)}>
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          data-slot="data-table-desktop"
          className={cn(
            "relative overflow-x-auto scrollbar-thin scrollbar-thumb-muted scrollbar-track-transparent",
            !embedded && "rounded-2xl border border-border/40 bg-card/90 backdrop-blur-sm shadow-sm"
          )}
        >
          <Table className="w-full">
              {caption && <caption className="sr-only">{caption}</caption>}
              <TableHeader className="sticky top-0 bg-background/95 backdrop-blur-sm z-10">
                {table.getHeaderGroups().map((headerGroup) => (
                  <TableRow key={headerGroup.id} className="border-b hover:bg-transparent">
                    {headerGroup.headers.map((header, headerIndex) => {
                      const canSort = header.column.getCanSort();
                      const sortDirection = header.column.getIsSorted();
                      const isFirstColumn = stickyFirstColumn && headerIndex === 0;

                      return (
                        <TableHead
                          key={header.id}
                          scope="col"
                          className={cn(
                            "whitespace-nowrap text-left typography-label-medium text-muted-foreground border-0",
                            isDense && "h-9 text-[11px] uppercase tracking-wide font-semibold",
                            headerPadding,
                            isFirstColumn &&
                              "sticky left-0 z-20 bg-background shadow-[4px_0_8px_-4px_rgba(0,0,0,0.15)]"
                          )}
                          style={getColumnSizingStyles(header.column)}
                        >
                          {header.isPlaceholder ? null : canSort ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              className={cn(
                                "-ml-3 h-9 data-[state=open]:bg-accent hover:bg-accent/50",
                                isDense && "h-7 text-[11px]"
                              )}
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
                  table.getRowModel().rows.map((row, rowIndex) => (
                    <TableRow
                      key={row.id}
                      data-state={row.getIsSelected() && "selected"}
                      className={cn(
                        "group hover:bg-muted/50 border-b",
                        isDense && "h-10 odd:bg-muted/30",
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
                      {row.getVisibleCells().map((cell, cellIndex) => {
                        const isFirstCell = stickyFirstColumn && cellIndex === 0;
                        return (
                          <TableCell
                            key={cell.id}
                            className={cn(
                              "align-middle",
                              isDense && "text-[12px] leading-4",
                              cellPadding,
                              isFirstCell && getRowClassName?.(row.original),
                              isFirstCell &&
                                "sticky left-0 z-10 bg-card shadow-[4px_0_8px_-4px_rgba(0,0,0,0.12)]",
                              rowIndex % 2 === 1 && stickyFirstColumn && "bg-muted/30",
                              rowIndex % 2 === 1 && stickyFirstColumn && "odd:bg-muted/30"
                            )}
                            style={getColumnSizingStyles(cell.column)}
                          >
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        );
                      })}
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={columns.length} className="text-center">
                      {emptyState ?? (
                        <EmptyState
                          title="Không có dữ liệu"
                          description="Thử thay đổi bộ lọc hoặc tìm kiếm khác."
                          size="sm"
                        />
                      )}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          {stickyFirstColumn && isScrolledHorizontally && (
            <div
              aria-hidden="true"
              className="pointer-events-none absolute inset-y-0 right-0 w-10 bg-gradient-to-l from-black/[0.06] to-transparent"
            />
          )}
        </div>
      </div>

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
          <Card className="mx-1 rounded-2xl border-dashed bg-card/80">
            <CardContent className="px-4">
              {emptyState ?? (
                <EmptyState
                  title="Không có dữ liệu"
                  description="Thử đổi bộ lọc hoặc tìm kiếm khác."
                  size="sm"
                />
              )}
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
