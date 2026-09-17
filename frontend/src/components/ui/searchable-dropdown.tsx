import { useEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command';
import { ArrowLeft, Check, ChevronsUpDown } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useIsMobile } from '@/hooks/useBreakpoint';
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
  /** Accessible heading for the mobile picker. Defaults to the trigger placeholder. */
  mobileTitle?: string;
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
  mobileTitle,
}: SearchableDropdownProps) => {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const mobileBackButtonRef = useRef<HTMLButtonElement>(null);
  const mobilePickerRef = useRef<HTMLElement>(null);
  const isMobile = useIsMobile();

  useEffect(() => {
    if (isMobile && open) {
      mobileBackButtonRef.current?.focus();
    }
  }, [isMobile, open]);

  const closePicker = () => {
    setOpen(false);
    if (isMobile) {
      triggerRef.current?.focus();
    }
  };

  const handleMobilePickerKeyDown = (event: ReactKeyboardEvent<HTMLElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      closePicker();
      return;
    }

    if (event.key !== 'Tab') return;

    const focusableElements = Array.from(
      mobilePickerRef.current?.querySelectorAll<HTMLElement>(
        'button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])',
      ) ?? [],
    );
    const firstElement = focusableElements[0];
    const lastElement = focusableElements.at(-1);

    if (event.shiftKey && document.activeElement === firstElement) {
      event.preventDefault();
      lastElement?.focus();
    } else if (!event.shiftKey && document.activeElement === lastElement) {
      event.preventDefault();
      firstElement?.focus();
    }
  };

  const allOptions = allOption ? [allOption, ...options] : options;
  const selectedOption = allOptions.find(option => option.value === value);
  const isActive = pillStyle && value !== (allOption?.value ?? 'all');

  const trigger = pillStyle ? (
    <button
      ref={triggerRef}
      role="combobox"
      aria-label={selectedOption?.label ? `${placeholder}: ${selectedOption.label}` : placeholder}
      aria-expanded={open}
      onClick={isMobile ? () => setOpen(true) : undefined}
      className={cn(
        'inline-flex items-center gap-1 whitespace-nowrap rounded-md border px-2.5 py-0 text-xs font-medium outline-none transition-colors duration-100 select-none',
        isMobile ? 'h-11 min-h-11' : 'h-7',
        'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
        !isActive && 'border-border/60 bg-background text-muted-foreground hover:border-border hover:bg-accent/40 hover:text-foreground',
        isActive && 'border-primary/30 bg-primary/8 text-primary hover:bg-primary/12',
        className,
      )}
    >
      <span>{isActive ? (selectedOption?.label ?? placeholder) : placeholder}</span>
      <ChevronsUpDown className="size-3 shrink-0 opacity-50" />
    </button>
  ) : (
    <Button
      ref={triggerRef}
      variant="outline"
      role="combobox"
      aria-label={selectedOption?.label ? `${placeholder}: ${selectedOption.label}` : placeholder}
      aria-expanded={open}
      onClick={isMobile ? () => setOpen(true) : undefined}
      className={cn('h-auto justify-between py-2', isMobile ? 'min-h-11' : 'min-h-10', className)}
    >
      {selectedOption ? (
        <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5">
          <span className="typography-body-medium whitespace-normal break-words text-left">{selectedOption.label}</span>
          {selectedOption.subtitle && (
            <span className="typography-body-small whitespace-normal break-words text-left text-muted-foreground">{selectedOption.subtitle}</span>
          )}
        </div>
      ) : (
        <span>{placeholder}</span>
      )}
      <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
    </Button>
  );

  const renderCommand = (mobile: boolean) => (
    <Command
      className={cn(mobile && 'min-h-0 flex-1 rounded-none')}
      filter={(optionValue, search) => {
        if (!search) return 1;
        return vietnameseIncludes(optionValue, search) ? 1 : 0;
      }}
    >
      <CommandInput placeholder={searchPlaceholder} />
      <CommandList
        className={cn(
          mobile &&
            '!max-h-none min-h-0 flex-1 touch-pan-y overscroll-contain pb-[max(0.5rem,env(safe-area-inset-bottom))] [-webkit-overflow-scrolling:touch]',
        )}
      >
        <CommandEmpty>{emptyMessage}</CommandEmpty>
        <CommandGroup>
          {allOptions.map((option) => (
            <CommandItem
              key={option.value}
              value={`${option.label} ${option.subtitle || ''}`}
              onSelect={() => {
                onValueChange(option.value);
                closePicker();
              }}
              className="items-center py-1 text-xs"
            >
              <Check
                className={cn(
                  'mr-1.5 size-3 shrink-0',
                  value === option.value ? 'opacity-100' : 'opacity-0',
                )}
              />
              <span className="truncate text-xs">{option.label}</span>
              {option.subtitle && (
                <span className="ml-1.5 shrink-0 truncate text-xs text-muted-foreground">{option.subtitle}</span>
              )}
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </Command>
  );

  if (isMobile) {
    return (
      <>
        {trigger}
        {open && (
          <section
            ref={mobilePickerRef}
            role="region"
            aria-label={mobileTitle ?? placeholder}
            onKeyDown={handleMobilePickerKeyDown}
            className="fixed inset-x-0 bottom-0 z-[60] flex h-[85dvh] max-h-[42rem] min-h-0 flex-col overflow-hidden rounded-t-2xl border bg-background"
          >
            <div className="flex h-14 shrink-0 items-center gap-2 border-b px-2">
              <button
                ref={mobileBackButtonRef}
                type="button"
                onClick={closePicker}
                aria-label="Quay lại bộ lọc"
                className="inline-flex size-11 shrink-0 items-center justify-center rounded-xl text-muted-foreground hover:bg-accent hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <ArrowLeft className="size-5" />
              </button>
              <h2 className="text-base font-semibold">{mobileTitle ?? placeholder}</h2>
            </div>
            {renderCommand(true)}
          </section>
        )}
      </>
    );
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{trigger}</PopoverTrigger>
      <PopoverContent
        className={cn('p-0', pillStyle ? 'w-56' : 'w-[var(--radix-popover-trigger-width)]')}
        align="start"
      >
        {renderCommand(false)}
      </PopoverContent>
    </Popover>
  );
};
