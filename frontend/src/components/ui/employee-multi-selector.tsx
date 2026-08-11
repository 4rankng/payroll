import { useCallback, useEffect, useMemo, useState } from 'react';
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useEmployeesInfinite } from '@/hooks/api/useEmployees';
import { useDebounce } from '@/hooks/useDebounce';
import { cn } from '@/lib/utils';

interface EmployeeMultiSelectorProps {
  value: number[];
  onChange: (employeeIds: number[]) => void;
  placeholder?: string;
  disabled?: boolean;
}

export function EmployeeMultiSelector({
  value,
  onChange,
  placeholder = 'Tất cả nhân viên',
  disabled = false,
}: EmployeeMultiSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const debouncedSearch = useDebounce(searchValue, 300);
  const selectedIDs = useMemo(() => new Set(value), [value]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError } = useEmployeesInfinite({
    pageSize: 20,
    search: debouncedSearch.trim() || undefined,
  });

  const employees = useMemo(
    () => data?.pages.flatMap((page) => page.data ?? []) ?? [],
    [data?.pages],
  );

  const isSearching = searchValue !== debouncedSearch;
  const triggerLabel = value.length === 0 ? placeholder : `Đã chọn ${value.length} nhân viên`;

  useEffect(() => {
    if (!open) setSearchValue('');
  }, [open]);

  const handleToggle = useCallback((employeeID: number) => {
    if (selectedIDs.has(employeeID)) {
      onChange(value.filter((id) => id !== employeeID));
      return;
    }
    onChange([...value, employeeID]);
  }, [onChange, selectedIDs, value]);

  const handleScroll = useCallback((event: React.UIEvent<HTMLDivElement>) => {
    const target = event.currentTarget;
    if (target.scrollHeight - target.scrollTop - target.clientHeight < 50 && hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Chọn nhân viên"
          className="min-h-11 w-full justify-between gap-2 text-left"
          disabled={disabled}
        >
          <span className="min-w-0 truncate">{triggerLabel}</span>
          <ChevronsUpDown className="h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-[calc(100vw-2rem)] max-w-[420px] p-0 sm:w-[--radix-popover-trigger-width]"
      >
        <Command shouldFilter={false}>
          <div className="relative">
            <CommandInput
              value={searchValue}
              onValueChange={setSearchValue}
              placeholder="Tìm kiếm tên hoặc CCCD..."
            />
            {isSearching && (
              <Loader2 className="absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-muted-foreground" />
            )}
          </div>
          <CommandList onScroll={handleScroll}>
            {isLoading && !isSearching && (
              <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Đang tải nhân viên...
              </div>
            )}
            {isError && (
              <div className="px-3 py-3 text-sm text-destructive">Không thể tìm kiếm nhân viên.</div>
            )}
            {!isLoading && !isError && employees.length === 0 && (
              <CommandEmpty>Không tìm thấy nhân viên nào.</CommandEmpty>
            )}
            <CommandGroup>
              <CommandItem
                onSelect={() => onChange([])}
                className="min-h-12 cursor-pointer py-2.5"
              >
                <Check className={cn('mr-2 h-4 w-4', value.length === 0 ? 'opacity-100' : 'opacity-0')} />
                <span className="break-words leading-snug">{placeholder}</span>
              </CommandItem>
              {employees.map((employee) => (
                <CommandItem
                  key={employee.id}
                  value={String(employee.id)}
                  onSelect={() => handleToggle(employee.id)}
                  className="min-h-12 cursor-pointer py-2.5"
                >
                  <Check
                    className={cn('mr-2 h-4 w-4', selectedIDs.has(employee.id) ? 'opacity-100' : 'opacity-0')}
                  />
                  <div className="min-w-0">
                    <p className="break-words leading-snug">{employee.fullname}</p>
                    <p className="break-words text-xs text-muted-foreground">{employee.cccd}</p>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
            {isFetchingNextPage && (
              <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Đang tải thêm...
              </div>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
