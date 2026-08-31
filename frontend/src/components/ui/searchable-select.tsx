import { useState, useCallback, useMemo, useEffect } from 'react';
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

export interface SearchableSelectOption {
  value: string;
  label: string;
  /** Extra text matched by the filter but not shown (e.g. code, alias). */
  searchText?: string;
  disabled?: boolean;
}

interface SearchableSelectProps {
  value: string | undefined;
  onChange: (value: string) => void;
  options: SearchableSelectOption[];
  placeholder?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  disabled?: boolean;
  /** DOM id for the trigger button — lets a <label htmlFor> target it. */
  triggerId?: string;
  /** Accessible name for the trigger when the visible text is not enough. */
  triggerAriaLabel?: string;
  /** Layout/width classes applied to the outline trigger button. */
  triggerClassName?: string;
  contentClassName?: string;
  /** Alignment of the popover relative to the trigger. */
  contentAlign?: 'start' | 'center' | 'end';
  /** Render the popover with modal focus trapping (sheets/dialogs). */
  modal?: boolean;
}

/**
 * Single-select searchable dropdown for long option lists.
 *
 * Complements MultiSearchableDropdown (multi) and AsyncSearchableDropdown
 * (server-side search): sync options, client-side diacritic-insensitive
 * filtering via vietnameseIncludes, cmdk command list inside a popover.
 */
export function SearchableSelect({
  value,
  onChange,
  options,
  placeholder = 'Chọn...',
  searchPlaceholder = 'Tìm kiếm...',
  emptyMessage = 'Không tìm thấy.',
  disabled = false,
  triggerId,
  triggerAriaLabel,
  triggerClassName,
  contentClassName,
  contentAlign = 'start',
  modal = true,
}: SearchableSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');

  useEffect(() => {
    if (!open && query !== '') setQuery('');
  }, [open, query]);

  const filtered = useMemo(() => {
    const q = query.trim();
    if (!q) return options;
    return options.filter(
      (o) =>
        !o.disabled &&
        (vietnameseIncludes(o.label, q) ||
          (o.searchText !== undefined && vietnameseIncludes(o.searchText, q))),
    );
  }, [options, query]);

  const selected = useMemo(
    () => options.find((o) => o.value === value),
    [options, value],
  );

  const handleSelect = useCallback(
    (option: SearchableSelectOption) => {
      onChange(option.value);
      setOpen(false);
    },
    [onChange],
  );

  return (
    <Popover open={open} onOpenChange={setOpen} modal={modal}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          id={triggerId}
          aria-label={triggerAriaLabel}
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn('h-11 w-full justify-between font-normal', triggerClassName)}
        >
          <span className="block truncate">{selected?.label ?? placeholder}</span>
          <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className={cn('w-[--radix-popover-trigger-width] p-0', contentClassName)}
        align={contentAlign}
      >
        <Command shouldFilter={false} className="bg-card">
          <div className="relative">
            <CommandInput
              placeholder={searchPlaceholder}
              value={query}
              onValueChange={setQuery}
            />
          </div>
          <CommandList className="max-h-[260px]">
            {filtered.length === 0 && <CommandEmpty>{emptyMessage}</CommandEmpty>}
            <CommandGroup>
              {filtered.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.value}
                  disabled={option.disabled}
                  onSelect={() => handleSelect(option)}
                  className="items-start"
                >
                  <Check
                    className={cn(
                      'mr-2 h-4 w-4 shrink-0',
                      option.value === value ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  <span className="truncate">{option.label}</span>
                  {option.searchText && (
                    <span className="ml-2 truncate text-xs text-muted-foreground">
                      {option.searchText}
                    </span>
                  )}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
