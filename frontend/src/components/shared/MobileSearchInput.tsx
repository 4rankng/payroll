import * as React from 'react';
import { Search, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { useDebouncedInput } from '@/hooks/useDebouncedInput';
import { cn } from '@/lib/utils';

interface MobileSearchInputProps {
  value: string;
  onSearch: (value: string) => void;
  placeholder?: string;
  className?: string;
  debounceMs?: number;
}

/**
 * Full-height (h-11) search input for mobile pages.
 * Owns local state + debounce so the parent only receives committed values.
 */
export const MobileSearchInput = React.memo(function MobileSearchInput({
  value,
  onSearch,
  placeholder = 'Tìm kiếm...',
  className,
  debounceMs = 300,
}: MobileSearchInputProps) {
  const { localValue, handleChange, handleClear } = useDebouncedInput(value, onSearch, debounceMs);

  return (
    <div className={cn('relative flex-1', className)}>
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
      <Input
        placeholder={placeholder}
        value={localValue}
        onChange={handleChange}
        className="h-11 rounded-lg border-border bg-background pl-9 pr-11"
      />
      {localValue && (
        <button
          onClick={handleClear}
          className="absolute right-0 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="Xóa tìm kiếm"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
});
