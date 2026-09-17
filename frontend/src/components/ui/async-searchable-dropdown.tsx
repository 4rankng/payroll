import { useState, useEffect, useCallback } from 'react';
import type { Key, ReactNode, UIEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command';
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface AsyncSearchableDropdownEmptyMessages {
  searchTooShort: string;
  noResults: string;
  empty: string;
}

interface AsyncSearchableDropdownProps<T> {
  options: T[];
  value?: T | null;
  onSelect: (item: T) => void;
  placeholder: string;
  searchValue: string;
  onSearchValueChange: (value: string) => void;
  searchPlaceholder?: string;
  minSearchLength?: number;
  isLoading?: boolean;
  isSearching?: boolean;
  errorMessage?: string;
  loadingMessage?: string;
  loadingMoreMessage?: string;
  emptyMessages: AsyncSearchableDropdownEmptyMessages;
  renderTrigger?: (selected: T | null, placeholderText: string) => ReactNode;
  renderOption: (item: T, isSelected: boolean) => ReactNode;
  getOptionKey: (item: T) => Key;
  getOptionFilterValue: (item: T) => string;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  loadMore?: () => void;
  disabled?: boolean;
  triggerClassName?: string;
  popoverContentClassName?: string;
  modal?: boolean;
}

export function AsyncSearchableDropdown<T>({
  options,
  value,
  onSelect,
  placeholder,
  searchValue,
  onSearchValueChange,
  searchPlaceholder = 'Tìm kiếm...',
  minSearchLength = 3,
  isLoading = false,
  isSearching = false,
  errorMessage,
  loadingMessage = 'Đang tìm kiếm...',
  loadingMoreMessage = 'Đang tải thêm...',
  emptyMessages,
  renderTrigger,
  renderOption,
  getOptionKey,
  getOptionFilterValue,
  hasNextPage,
  isFetchingNextPage,
  loadMore,
  disabled = false,
  triggerClassName,
  popoverContentClassName,
  modal = true
}: AsyncSearchableDropdownProps<T>) {
  const [open, setOpen] = useState(false);

  const handleSelect = useCallback((item: T) => {
    onSelect(item);
    setOpen(false);
  }, [onSelect]);

  useEffect(() => {
    if (!open && searchValue !== '') {
      onSearchValueChange('');
    }
  }, [open, searchValue, onSearchValueChange]);

  const handleScroll = (e: UIEvent<HTMLDivElement>) => {
    const target = e.currentTarget;
    // Load more when scrolled to bottom with some tolerance
    if (target.scrollHeight - target.scrollTop <= target.clientHeight + 20) {
      loadMore?.();
    }
  };

  const triggerContent = renderTrigger ? renderTrigger(value ?? null, placeholder) : (
    <span className="text-left">{value ? getOptionFilterValue(value) : placeholder}</span>
  );

  return (
    <Popover open={open} onOpenChange={setOpen} modal={modal}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-label={value ? `${placeholder}: ${getOptionFilterValue(value)}` : placeholder}
          aria-expanded={open}
          disabled={disabled}
          className={cn("justify-between", triggerClassName)}
        >
          {triggerContent}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className={cn("w-72 p-0", popoverContentClassName)}>
        <Command shouldFilter={false}>
          <div className="relative">
            <CommandInput
              placeholder={searchPlaceholder}
              value={searchValue}
              onValueChange={onSearchValueChange}
            />
            {isSearching && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
              </div>
            )}
          </div>
          <CommandList
            onWheel={(e) => e.stopPropagation()}
            onScroll={loadMore ? handleScroll : undefined}
          >
            {isLoading && !isSearching && (
               <div className="py-6 text-center text-sm text-muted-foreground">
                 {loadingMessage}
               </div>
            )}

            {errorMessage && (
              <div className="py-6 text-center text-sm text-destructive">
                {errorMessage}
              </div>
            )}

            {!isLoading && !errorMessage && options.length === 0 && (
              <CommandEmpty>
                {searchValue.length < minSearchLength
                  ? emptyMessages.searchTooShort
                  : emptyMessages.noResults || emptyMessages.empty}
              </CommandEmpty>
            )}

            <CommandGroup>
              {options.map((option) => {
                const optionKey = getOptionKey(option);
                const commandValue = getOptionFilterValue(option);
                const isSelected = value ? getOptionKey(value) === optionKey : false;

                return (
                  <CommandItem
                    key={optionKey}
                    value={commandValue}
                    onSelect={() => handleSelect(option)}
                    className="items-start"
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        isSelected ? "opacity-100" : "opacity-0"
                      )}
                    />
                    {renderOption(option, isSelected)}
                  </CommandItem>
                );
              })}
              {hasNextPage && isFetchingNextPage && (
                <div className="py-2 text-center text-sm text-muted-foreground">
                  {loadingMoreMessage}
                </div>
              )}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
