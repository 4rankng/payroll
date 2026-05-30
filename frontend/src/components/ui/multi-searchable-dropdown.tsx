import { useState, useCallback, useMemo } from 'react';
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
import { Check, ChevronDown } from 'lucide-react';
import { cn } from '@/lib/utils';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

export interface MultiSearchableDropdownOption {
  value: string;
  label: string;
  searchText?: string;
}

interface MultiSearchableDropdownProps {
  value: string[];
  onChange: (values: string[]) => void;
  options: MultiSearchableDropdownOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  allOption?: {
    value: string;
    label: string;
  };
  disabled?: boolean;
  className?: string;
}

export function MultiSearchableDropdown({
  value,
  onChange,
  options,
  placeholder = 'Chọn...',
  searchPlaceholder = 'Tìm kiếm...',
  emptyMessage = 'Không tìm thấy.',
  allOption,
  disabled = false,
  className,
}: MultiSearchableDropdownProps) {
  const [open, setOpen] = useState(false);

  const valueSet = useMemo(() => new Set(value), [value]);

  const isAllSelected = useMemo(() => {
    if (!allOption) return false;
    return value.length === 0 || value.length === options.length;
  }, [value.length, options.length, allOption]);

  const displayText = useMemo(() => {
    if (!allOption || value.length === 0 || value.length === options.length) {
      return placeholder;
    }
    return `Đã chọn ${value.length}`;
  }, [value.length, options.length, placeholder, allOption]);

  const handleToggleAll = useCallback(() => {
    if (isAllSelected) {
      onChange([]);
    } else {
      onChange(options.map((o) => o.value));
    }
  }, [isAllSelected, options, onChange]);

  const handleToggle = useCallback(
    (optionValue: string) => {
      onChange(
        valueSet.has(optionValue) ? value.filter((v) => v !== optionValue) : [...value, optionValue],
      );
    },
    [value, valueSet, onChange],
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn('w-full justify-between', className)}
          disabled={disabled}
        >
          {displayText}
          <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-full p-0">
        <Command
          shouldFilter
          filter={(itemValue, search) => {
            if (!search) return 1;
            return vietnameseIncludes(itemValue, search) ? 1 : 0;
          }}
        >
          <CommandInput placeholder={searchPlaceholder} />
          <CommandList>
            <CommandEmpty>{emptyMessage}</CommandEmpty>
            <CommandGroup>
              {allOption && (
                <CommandItem onSelect={handleToggleAll} className="cursor-pointer">
                  <Check
                    className={cn('mr-2 h-4 w-4', isAllSelected ? 'opacity-100' : 'opacity-0')}
                  />
                  {allOption.label}
                </CommandItem>
              )}
              {options.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.searchText ?? option.label}
                  onSelect={() => handleToggle(option.value)}
                  className="cursor-pointer"
                >
                  <Check
                    className={cn('mr-2 h-4 w-4', valueSet.has(option.value) ? 'opacity-100' : 'opacity-0')}
                  />
                  {option.label}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
