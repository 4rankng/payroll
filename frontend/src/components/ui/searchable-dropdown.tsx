import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command';
import { Check, ChevronsUpDown } from 'lucide-react';
import { cn } from '@/lib/utils';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

interface SearchableDropdownOption {
  value: string;
  label: string;
  subtitle?: string;
}

interface SearchableDropdownProps {
  value: string;
  onValueChange: (value: string) => void;
  options: SearchableDropdownOption[];
  placeholder: string;
  searchPlaceholder: string;
  emptyMessage: string;
  className?: string;
  allOption?: {
    value: string;
    label: string;
    subtitle?: string;
  };
  /** Render as a compact filter pill instead of a full-height button */
  pillStyle?: boolean;
}

export const SearchableDropdown = ({
  value,
  onValueChange,
  options,
  placeholder,
  searchPlaceholder,
  emptyMessage,
  className,
  allOption,
  pillStyle = false,
}: SearchableDropdownProps) => {
  const [open, setOpen] = useState(false);

  const allOptions = allOption ? [allOption, ...options] : options;
  const selectedOption = allOptions.find(option => option.value === value);
  const isActive = pillStyle && value !== (allOption?.value ?? 'all');

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        {pillStyle ? (
          <button
            role="combobox"
            aria-expanded={open}
            className={cn(
              'inline-flex items-center gap-1 h-7 px-2.5 py-0 rounded-md text-xs font-medium whitespace-nowrap',
              'border transition-colors duration-100 select-none outline-none',
              'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
              !isActive && 'border-border/60 bg-background text-muted-foreground hover:border-border hover:text-foreground hover:bg-accent/40',
              isActive && 'border-primary/30 bg-primary/8 text-primary hover:bg-primary/12',
              className,
            )}
          >
            <span>{isActive ? selectedOption?.label : placeholder}</span>
            <ChevronsUpDown className="h-3 w-3 shrink-0 opacity-50" />
          </button>
        ) : (
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className={cn("justify-between h-auto min-h-10 py-2", className)}
          >
            {selectedOption ? (
              <div className="flex flex-col items-start gap-0.5 flex-1 min-w-0">
                <span className="typography-body-medium whitespace-normal break-words text-left">{selectedOption.label}</span>
                {selectedOption.subtitle && (
                  <span className="typography-body-small text-muted-foreground whitespace-normal break-words text-left">{selectedOption.subtitle}</span>
                )}
              </div>
            ) : (
              <span>{placeholder}</span>
            )}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        )}
      </PopoverTrigger>
      <PopoverContent
        className={cn("p-0", pillStyle ? "w-56" : "w-[var(--radix-popover-trigger-width)]")}
        align="start"
      >
        <Command
          filter={(value, search) => {
            if (!search) return 1;
            return vietnameseIncludes(value, search) ? 1 : 0;
          }}
        >
          <CommandInput placeholder={searchPlaceholder} />
          <CommandList>
            <CommandEmpty>{emptyMessage}</CommandEmpty>
            <CommandGroup>
              {allOptions.map((option) => (
                <CommandItem
                  key={option.value}
                  value={`${option.label} ${option.subtitle || ''}`}
                  onSelect={() => {
                    onValueChange(option.value);
                    setOpen(false);
                  }}
                  className="items-center py-1 text-xs"
                >
                  <Check
                    className={cn(
                      "mr-1.5 h-3 w-3 shrink-0",
                      value === option.value ? "opacity-100" : "opacity-0"
                    )}
                  />
                  <span className="truncate text-xs">{option.label}</span>
                  {option.subtitle && (
                    <span className="ml-1.5 text-xs text-muted-foreground truncate shrink-0">{option.subtitle}</span>
                  )}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
};