import * as React from 'react';
import { Search, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SearchBarProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  placeholder?: string;
  showClearButton?: boolean;
  className?: string;
  debounceMs?: number;
}

export const SearchBar = React.memo(function SearchBar({
  searchTerm,
  onSearchChange,
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
        'inline-flex items-center gap-1.5 h-7 px-2.5 py-0 rounded-xl text-xs',
        'border border-border/60 bg-card text-muted-foreground',
        'hover:border-border hover:text-foreground transition-colors duration-100',
        'focus-within:border-ring focus-within:ring-2 focus-within:ring-ring/20',
        className,
      )}
    >
      <Search className="h-3.5 w-3.5 shrink-0 opacity-50" />
      <input
        type="text"
        value={localValue}
        onChange={handleChange}
        placeholder={placeholder}
        className="flex-1 min-w-0 bg-transparent outline-none placeholder:text-muted-foreground/60 text-foreground"
      />
      {showClearButton && localValue && (
        <button
          onClick={handleClear}
          className="shrink-0 opacity-50 hover:opacity-100 transition-opacity"
          aria-label="Xóa tìm kiếm"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  );
});
