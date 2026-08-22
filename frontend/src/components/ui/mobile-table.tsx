import * as React from "react";
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { MoreHorizontal } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { EmptyState } from "@/components/shared/EmptyState";

type MobileField<T> = {
  key: keyof T | string;
  label: string;
  priority?: number; // 1 = primary (shown in trigger), 2+ = secondary (shown in content)
  render?: (row: T) => React.ReactNode;
};

type RowAction<T> = {
  label: string;
  onClick: (row: T) => void;
  icon?: React.ReactNode;
  variant?: "default" | "destructive";
};

type MobileTableProps<T extends Record<string, unknown>> = {
  data: T[];
  fields: MobileField<T>[];
  rowTitle?: (row: T) => React.ReactNode; // Main title in trigger
  rowSubtitle?: (row: T) => React.ReactNode; // Subtitle in trigger
  rowActions?: RowAction<T>[]; // Actions for overflow menu
  getRowId?: (row: T) => string;
  getRowClassName?: (row: T) => string; // Optional per-row accent class
  emptyState?: React.ReactNode;
  className?: string;
  accordionType?: "single" | "multiple"; // Single = only one open, Multiple = many can be open
  onRowClick?: (row: T) => void; // Called when row is clicked (outside of accordion functionality)
  // Pagination props
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
};

export function MobileTable<T extends Record<string, unknown>>({
  data,
  fields,
  rowTitle,
  rowSubtitle,
  rowActions = [],
  getRowId,
  getRowClassName,
  emptyState,
  className,
  accordionType = "single",
  onRowClick,
  pagination,
  onPageChange,
  onPageSizeChange,
}: MobileTableProps<T>) {
  // Separate fields by priority
  const primaryFields = React.useMemo(
    () => fields.filter(f => (f.priority ?? 2) === 1),
    [fields]
  );

  const secondaryFields = React.useMemo(
    () => fields.filter(f => (f.priority ?? 2) !== 1),
    [fields]
  );

  // Render empty state
  if (!data?.length) {
    return (
      <div data-slot="mobile-table-empty" className={cn("rounded-xl border border-dashed border-border/70 bg-card/80 px-4", className)}>
        {emptyState ?? <EmptyState title="Không có dữ liệu" description="Chưa có dữ liệu để hiển thị." size="sm" />}
      </div>
    );
  }

  // Get field value from row
  const getFieldValue = (row: T, field: MobileField<T>): React.ReactNode => {
    if (field.render) {
      return field.render(row);
    }
    const value = row[field.key as keyof T];
    // Handle objects that might be rendered as children
    if (value && typeof value === 'object' && !React.isValidElement(value)) {
      if ('name' in value && value.name) return String(value.name);
      if ('code' in value && value.code) return String(value.code);
      if ('id' in value && value.id) return String(value.id);
      return JSON.stringify(value);
    }
    return value as unknown as string;
  };

  // Generate row ID - ensure uniqueness by combining with index as fallback
  const generateRowId = (row: T, index: number) => {
    if (getRowId) {
      const customId = getRowId(row);
      return `${customId}-${index}`;
    }
    if (row.id) return `${String(row.id)}-${index}`;
    return `row-${index}`;
  };

  return (
    <div
      role="list"
      data-slot="mobile-table"
      className={cn("w-full", className)}
      aria-label="Data table mobile view"
    >
      <Accordion
        type={accordionType}
        collapsible={accordionType === "single"}
        className="w-full"
      >
        {data.map((row, index) => {
          const rowId = generateRowId(row, index);
          const extraClass = getRowClassName?.(row) ?? "";

          return (
            <AccordionItem
              value={rowId}
              key={rowId}
              role="listitem"
              data-slot="mobile-table-row"
              className={cn(
                "mb-2.5 overflow-hidden rounded-xl border border-border/50 bg-card shadow-sm transition-all duration-200 touch-manipulation hover:border-border hover:shadow-md",
                extraClass,
              )}
            >
              <AccordionTrigger
                hideChevron
                className="min-h-[64px] px-4 py-3.5 transition-transform touch-manipulation hover:no-underline active:scale-[0.99]"
                onClick={(e) => {
                  if (onRowClick) {
                    e.preventDefault();
                    e.stopPropagation();
                    onRowClick(row);
                  }
                }}
              >
                <div className="flex w-full items-start justify-between gap-3 text-left">
                  {/* Left: Title & Subtitle */}
                  {(rowTitle || rowSubtitle) && (
                    <div className="min-w-0 flex-1 space-y-1">
                      {rowTitle && (
                        <div className="text-foreground leading-tight">
                          {rowTitle(row)}
                        </div>
                      )}
                      {rowSubtitle && (
                        <div className="text-muted-foreground">
                          {rowSubtitle(row)}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Right (or Left if no title): Primary fields */}
                  {primaryFields.length > 0 && (
                    <div className={cn("flex min-w-0 shrink-0 flex-col gap-1.5", (rowTitle || rowSubtitle) ? "max-w-[46%] items-end text-right" : "flex-1 items-start text-left")}>
                      {primaryFields.map((field) => {
                        const value = getFieldValue(row, field);
                        if (!value) return null;
                        return (
                          <div key={String(field.key)} className="max-w-full">
                            {value}
                          </div>
                        );
                      })}
                    </div>
                  )}

                  {/* Row actions dropdown */}
                  {rowActions.length > 0 && (
                    <div className="shrink-0 ml-1">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-11 w-11 p-0 touch-manipulation"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <MoreHorizontal className="h-4 w-4" />
                            <span className="sr-only">Mở menu hành động</span>
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {rowActions.map((action, actionIndex) => (
                            <DropdownMenuItem
                              key={actionIndex}
                              onClick={(e) => {
                                e.stopPropagation();
                                action.onClick(row);
                              }}
                              className={cn(
                                action.variant === "destructive" && "text-destructive focus:text-destructive"
                              )}
                            >
                              {action.icon && <span className="mr-2">{action.icon}</span>}
                              {action.label}
                            </DropdownMenuItem>
                          ))}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  )}
                </div>
              </AccordionTrigger>

              {/* Secondary fields in accordion content */}
              {secondaryFields.length > 0 && (
                <AccordionContent className="pb-0">
                  <div className="mx-4 border-t border-border/40 pt-3 pb-4">
                    <dl className="grid grid-cols-2 gap-x-4 gap-y-3.5">
                      {secondaryFields.map((field) => {
                        const value = getFieldValue(row, field);
                        if (!value) return null;
                        return (
                          <div key={String(field.key)} className="min-w-0">
                            <dt className="text-[11px] font-semibold text-muted-foreground/70 uppercase tracking-[0.06em] mb-1">{field.label}</dt>
                            <dd className="min-w-0 break-words text-[13px] leading-relaxed text-foreground">{value}</dd>
                          </div>
                        );
                      })}
                    </dl>
                  </div>
                </AccordionContent>
              )}
            </AccordionItem>
          );
        })}
      </Accordion>

      {/* Pagination controls */}
      {pagination && onPageChange && onPageSizeChange && (
        <PaginationControls
          pagination={pagination}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
          className="border-t border-border/50 pt-3 mt-2"
        />
      )}
    </div>
  );
}

// Export type for external use
export type { MobileField, RowAction, MobileTableProps };
