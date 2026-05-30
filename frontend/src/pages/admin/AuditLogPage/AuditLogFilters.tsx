import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Filter, X, ChevronDown } from 'lucide-react';
import { VIETNAMESE_AUDIT_LABELS } from '@/types/api/audit.types';
import type { AuditLogsQueryParams } from '@/types/api/audit.types';

const ALL_ACTIONS = Object.keys(VIETNAMESE_AUDIT_LABELS.actions) as string[];
const ALL_ENTITY_TYPES = Object.keys(VIETNAMESE_AUDIT_LABELS.entities) as string[];

interface AuditLogFiltersProps {
  filters: AuditLogsQueryParams;
  onChange: (filters: AuditLogsQueryParams) => void;
}

function ActionMultiSelect({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (v: string[]) => void;
}) {
  const toggle = (action: string) => {
    onChange(
      selected.includes(action)
        ? selected.filter((a) => a !== action)
        : [...selected, action]
    );
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 text-xs gap-1.5">
          Hành động
          {selected.length > 0 && (
            <Badge className="h-4 min-w-4 px-1 text-[10px] bg-primary text-primary-foreground border-0">
              {selected.length}
            </Badge>
          )}
          <ChevronDown className="w-3 h-3 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-52 max-h-72 overflow-y-auto">
        <DropdownMenuLabel className="text-xs">Lọc theo hành động</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {ALL_ACTIONS.map((action) => (
          <DropdownMenuCheckboxItem
            key={action}
            checked={selected.includes(action)}
            onCheckedChange={() => toggle(action)}
            className="text-xs"
          >
            {(VIETNAMESE_AUDIT_LABELS.actions as Record<string, string>)[action] ?? action}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function EntityTypeMultiSelect({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (v: string[]) => void;
}) {
  const toggle = (et: string) => {
    onChange(
      selected.includes(et)
        ? selected.filter((e) => e !== et)
        : [...selected, et]
    );
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 text-xs gap-1.5">
          Đối tượng
          {selected.length > 0 && (
            <Badge className="h-4 min-w-4 px-1 text-[10px] bg-primary text-primary-foreground border-0">
              {selected.length}
            </Badge>
          )}
          <ChevronDown className="w-3 h-3 text-muted-foreground" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-52 max-h-72 overflow-y-auto">
        <DropdownMenuLabel className="text-xs">Lọc theo đối tượng</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {ALL_ENTITY_TYPES.map((et) => (
          <DropdownMenuCheckboxItem
            key={et}
            checked={selected.includes(et)}
            onCheckedChange={() => toggle(et)}
            className="text-xs"
          >
            {(VIETNAMESE_AUDIT_LABELS.entities as Record<string, string>)[et] ?? et}
          </DropdownMenuCheckboxItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function FilterBody({ filters, onChange }: AuditLogFiltersProps) {
  const activeCount =
    (filters.action?.length ?? 0) +
    (filters.entityType?.length ?? 0) +
    (filters.fromDate ? 1 : 0) +
    (filters.toDate ? 1 : 0);

  const reset = () => onChange({ page: 1, pageSize: filters.pageSize });

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Input
        type="date"
        value={filters.fromDate ?? ''}
        onChange={(e) => onChange({ ...filters, fromDate: e.target.value || undefined, page: 1 })}
        className="h-8 text-xs w-36"
      />
      <span className="text-muted-foreground text-xs">—</span>
      <Input
        type="date"
        value={filters.toDate ?? ''}
        onChange={(e) => onChange({ ...filters, toDate: e.target.value || undefined, page: 1 })}
        className="h-8 text-xs w-36"
      />

      <ActionMultiSelect
        selected={filters.action ?? []}
        onChange={(v) => onChange({ ...filters, action: v.length ? v : undefined, page: 1 })}
      />

      <EntityTypeMultiSelect
        selected={filters.entityType ?? []}
        onChange={(v) => onChange({ ...filters, entityType: v.length ? v : undefined, page: 1 })}
      />

      {activeCount > 0 && (
        <Button
          variant="ghost"
          size="sm"
          onClick={reset}
          className="h-8 text-xs text-muted-foreground gap-1"
        >
          <X className="w-3 h-3" />
          Xóa bộ lọc
          <Badge variant="secondary" className="h-4 min-w-4 px-1 text-[10px]">
            {activeCount}
          </Badge>
        </Button>
      )}
    </div>
  );
}

export function AuditLogFilters({ filters, onChange }: AuditLogFiltersProps) {
  const [mobileOpen, setMobileOpen] = useState(false);

  const activeCount =
    (filters.action?.length ?? 0) +
    (filters.entityType?.length ?? 0) +
    (filters.fromDate ? 1 : 0) +
    (filters.toDate ? 1 : 0);

  return (
    <>
      {/* Desktop */}
      <div className="hidden md:flex items-center gap-2 flex-wrap">
        <FilterBody filters={filters} onChange={onChange} />
      </div>

      {/* Mobile */}
      <div className="flex md:hidden">
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetTrigger asChild>
            <Button variant="outline" size="sm" className="h-8 text-xs gap-1.5">
              <Filter className="w-3.5 h-3.5" />
              Bộ lọc
              {activeCount > 0 && (
                <Badge className="h-4 min-w-4 px-1 text-[10px] bg-primary text-primary-foreground border-0">
                  {activeCount}
                </Badge>
              )}
            </Button>
          </SheetTrigger>
          <SheetContent side="bottom" className="pb-8">
            <SheetHeader className="mb-4">
              <SheetTitle className="text-sm">Bộ lọc</SheetTitle>
            </SheetHeader>
            <div className="flex flex-col gap-3">
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground mb-1 block">Từ ngày</label>
                  <Input
                    type="date"
                    value={filters.fromDate ?? ''}
                    onChange={(e) => onChange({ ...filters, fromDate: e.target.value || undefined, page: 1 })}
                    className="h-9 text-sm w-full"
                  />
                </div>
                <div className="flex-1">
                  <label className="text-xs text-muted-foreground mb-1 block">Đến ngày</label>
                  <Input
                    type="date"
                    value={filters.toDate ?? ''}
                    onChange={(e) => onChange({ ...filters, toDate: e.target.value || undefined, page: 1 })}
                    className="h-9 text-sm w-full"
                  />
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                <ActionMultiSelect
                  selected={filters.action ?? []}
                  onChange={(v) => onChange({ ...filters, action: v.length ? v : undefined, page: 1 })}
                />
                <EntityTypeMultiSelect
                  selected={filters.entityType ?? []}
                  onChange={(v) => onChange({ ...filters, entityType: v.length ? v : undefined, page: 1 })}
                />
              </div>
              {activeCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => { onChange({ page: 1, pageSize: filters.pageSize }); setMobileOpen(false); }}
                  className="text-xs text-muted-foreground gap-1 self-start"
                >
                  <X className="w-3 h-3" />
                  Xóa tất cả bộ lọc
                </Button>
              )}
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </>
  );
}
