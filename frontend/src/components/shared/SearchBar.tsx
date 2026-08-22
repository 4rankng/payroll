import * as React from 'react';
import { Search, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SearchBarProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  inputId?: string;
  placeholder?: string;
  showClearButton?: boolean;
  className?: string;
  debounceMs?: number;
}

export const SearchBar = React.memo(function SearchBar({
  searchTerm,
  onSearchChange,
  inputId,
  placeholder = 'Tìm kiếm...',
  showClearButton = true,
  className,
  debounceMs = 300,
}: SearchBarProps) {
  const [localValue, setLocalValue] = React.useState(searchTerm);
  const timerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  // Track the last value we committed so we don't re-sync from parent echoes
  const lastCommitted = React.useRef(searchTerm);

  // Only sync inward when parent explicitly clears (e.g. "clear filters" button)
  React.useEffect(() => {
    if (searchTerm === '' && lastCommitted.current !== '') {
      lastCommitted.current = '';
      setLocalValue('');
    }
  }, [searchTerm]);

  const handleChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setLocalValue(value);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        lastCommitted.current = value;
        onSearchChange(value);
      }, debounceMs);
    },
    [onSearchChange, debounceMs],
  );

  const handleClear = React.useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    lastCommitted.current = '';
    setLocalValue('');
    onSearchChange('');
  }, [onSearchChange]);

  React.useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  return (
    <div
      className={cn(
        'inline-flex min-h-11 items-center gap-2 rounded-xl px-3 py-0 text-sm sm:min-h-9 sm:px-2.5',
        'border border-border/60 bg-card text-muted-foreground',
        'hover:border-border hover:text-foreground transition-colors duration-100',
        'focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/20',
        className,
      )}
    >
      <Search className="h-4 w-4 shrink-0 opacity-50" />
      <input
        id={inputId}
        type="text"
        value={localValue}
        onChange={handleChange}
        aria-label={inputId ? undefined : placeholder}
        placeholder={placeholder}
        className="flex-1 min-w-0 bg-transparent outline-none placeholder:text-muted-foreground/60 text-foreground"
      />
      {showClearButton && localValue && (
        <button
          onClick={handleClear}
          className="-mr-2 flex h-11 w-11 shrink-0 items-center justify-center rounded-lg opacity-50 transition-all hover:bg-muted hover:opacity-100 sm:-mr-1.5 sm:h-8 sm:w-8"
          aria-label="Xóa tìm kiếm"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
});
