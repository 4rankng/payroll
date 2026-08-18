import { useCallback, useEffect, useMemo, useState } from 'react';
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useEmployeesInfinite } from '@/hooks/api/useEmployees';
import { useDebounce } from '@/hooks/useDebounce';
import { cn } from '@/lib/utils';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeSingleSelectorProps {
  value: Employee | null;
  onSelect: (employee: Employee) => void;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  ariaLabel?: string;
}

/**
 * A deliberately separate single-select control for server-side employee lookup.
 * It mirrors EmployeeMultiSelector without changing transfer form selection rules.
 */
export function EmployeeSingleSelector({
  value,
  onSelect,
  placeholder = 'Chọn nhân viên',
  disabled = false,
  id,
  ariaLabel = 'Chọn nhân viên',
}: EmployeeSingleSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const debouncedSearch = useDebounce(searchValue, 300);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError } =
    useEmployeesInfinite({
      pageSize: 20,
      search: debouncedSearch.trim() || undefined,
    });

  const employees = useMemo(
    () => data?.pages.flatMap((page) => page.data ?? []) ?? [],
    [data?.pages],
  );
  const isSearching = searchValue !== debouncedSearch;

  useEffect(() => {
    if (!open) setSearchValue('');
  }, [open]);

  const handleScroll = useCallback((event: React.UIEvent<HTMLDivElement>) => {
    const target = event.currentTarget;
    if (
      target.scrollHeight - target.scrollTop - target.clientHeight < 50 &&
      hasNextPage &&
      !isFetchingNextPage
    ) {
      fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  const handleSelect = useCallback((employee: Employee) => {
    onSelect(employee);
    setOpen(false);
  }, [onSelect]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label={ariaLabel}
          className="min-h-11 w-full justify-between gap-2 text-left"
          disabled={disabled}
        >
          <span className={cn('min-w-0 break-words text-left', !value && 'truncate text-muted-foreground')}>
            {value ? value.fullname : placeholder}
          </span>
          <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
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
              placeholder="Tìm theo tên hoặc CCCD..."
              aria-label="Tìm nhân viên"
            />
            {isSearching && (
              <Loader2 className="absolute right-3 top-1/2 size-4 -translate-y-1/2 animate-spin text-muted-foreground" />
            )}
          </div>
          <CommandList onScroll={handleScroll}>
            {isLoading && !isSearching && (
              <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Đang tải nhân viên...
              </div>
            )}
            {isError && (
              <div className="px-3 py-3 text-sm text-destructive">
                Không thể tìm kiếm nhân viên. Vui lòng thử lại.
              </div>
            )}
            {!isLoading && !isError && employees.length === 0 && (
              <CommandEmpty>Không tìm thấy nhân viên nào.</CommandEmpty>
            )}
            <CommandGroup>
              {employees.map((employee) => (
                <CommandItem
                  key={employee.id}
                  value={String(employee.id)}
                  onSelect={() => handleSelect(employee)}
                  className="min-h-12 cursor-pointer py-2.5"
                >
                  <Check className={cn('mr-2 size-4 shrink-0', value?.id === employee.id ? 'opacity-100' : 'opacity-0')} />
                  <div className="min-w-0">
                    <p className="break-words leading-snug">{employee.fullname}</p>
                    <p className="break-words text-xs text-muted-foreground">CCCD: {employee.cccd}</p>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
            {isFetchingNextPage && (
              <div className="flex items-center gap-2 px-3 py-3 text-sm text-muted-foreground">
                <Loader2 className="size-4 animate-spin" />
                Đang tải thêm...
              </div>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
