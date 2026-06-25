import { useState, useEffect, useMemo, useRef, forwardRef } from 'react';
import { Check, Loader2, Search, Plus, X, ChevronsUpDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogClose,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '@/components/ui/drawer';
import { cn } from '@/lib/utils';
import { useAllBanks, useCreateBank } from '@/hooks/api/useBanks';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';
import type { Bank } from '@/types/api/bank.types';

interface BankSelectorProps {
  value?: Bank | null;
  onSelect: (bank: Bank | null) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  canCreateBank?: boolean;
}

// ─── Bank list (client-side filtered) ────────────────────────────────────────

interface BankListProps {
  value?: Bank | null;
  onSelect: (bank: Bank) => void;
  canCreateBank: boolean;
  searchValue: string;
  scrollHeight?: string;
}

function BankList({ value, onSelect, canCreateBank, searchValue, scrollHeight = '100%' }: BankListProps) {
  const [isCreatingCustomBank, setIsCreatingCustomBank] = useState(false);
  const { data: allBanks, isLoading } = useAllBanks();
  const createBankMutation = useCreateBank();

  const banks = useMemo(() => {
    if (!allBanks) return [];
    const q = searchValue.trim();
    if (!q) return allBanks;
    return allBanks.filter(b => vietnameseIncludes(b.branch_name, q));
  }, [allBanks, searchValue]);

  const handleCreateCustomBank = async () => {
    if (!searchValue.trim()) return;
    setIsCreatingCustomBank(true);
    try {
      const newBank = await createBankMutation.mutateAsync({ branch_name: searchValue.trim() });
      onSelect(newBank);
    } catch {
      // handled globally
    } finally {
      setIsCreatingCustomBank(false);
    }
  };

  return (
    <div
      data-vaul-no-drag
      className={cn(
        "overflow-y-auto overscroll-contain p-2",
        scrollHeight === '100%' ? 'flex-1 min-h-0' : ''
      )}
      style={scrollHeight !== '100%' ? { height: scrollHeight, scrollbarWidth: 'thin' } : { scrollbarWidth: 'thin' }}
    >
      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      )}

      {!isLoading && banks.length === 0 && (
        <div className="py-12 text-center text-sm text-muted-foreground">
          Không tìm thấy ngân hàng nào
        </div>
      )}

      {!isLoading && banks.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-1.5">
          {banks.map((bank) => {
            const isSelected = value?.id === bank.id;
            return (
              <button
                key={bank.id}
                onClick={() => onSelect(bank)}
                className={cn(
                  "flex items-start gap-1.5 rounded-md px-2.5 py-2 text-left text-sm transition-colors",
                  "active:scale-[0.98]",
                  isSelected
                    ? "bg-primary/10 text-primary font-medium ring-1 ring-primary/20"
                    : "text-foreground hover:bg-accent"
                )}
              >
                {isSelected && <Check className="h-3.5 w-3.5 shrink-0 text-primary mt-0.5" />}
                <span className="break-words leading-snug min-w-0">{bank.branch_name}</span>
              </button>
            );
          })}
        </div>
      )}

      {/* Create custom bank */}
      {canCreateBank && searchValue.trim() && !isLoading && (
        <div className="mt-2 border-t pt-2">
          <button
            onClick={handleCreateCustomBank}
            disabled={isCreatingCustomBank || createBankMutation.isPending}
            className={cn(
              "flex w-full items-center gap-2 rounded-md border-2 border-dashed border-blue-300 px-3 py-2 text-sm text-blue-700",
              "hover:border-blue-400 hover:bg-blue-50 active:scale-[0.98] transition-all",
              "disabled:opacity-50 disabled:cursor-not-allowed"
            )}
          >
            {isCreatingCustomBank || createBankMutation.isPending
              ? <Loader2 className="h-4 w-4 animate-spin shrink-0" />
              : <Plus className="h-4 w-4 shrink-0" />}
            <span className="truncate font-medium">
              {isCreatingCustomBank || createBankMutation.isPending
                ? 'Đang tạo...'
                : `Tạo "${searchValue}"`}
            </span>
          </button>
        </div>
      )}
    </div>
  );
}

// ─── Search bar ──────────────────────────────────────────────────────────────

interface SearchBarProps {
  value: string;
  onChange: (v: string) => void;
  inputRef?: React.RefObject<HTMLInputElement>;
}

function SearchBar({ value, onChange, inputRef }: SearchBarProps) {
  return (
    <div className="relative flex items-center flex-1">
      <Search className="absolute left-2.5 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
      <input
        ref={inputRef}
        className="h-9 w-full rounded-md border bg-muted/50 pl-8 pr-8 text-sm outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
        placeholder="Tìm kiếm ngân hàng..."
        value={value}
        onChange={(e) => onChange(e.target.value)}
        autoComplete="off"
      />
      {value ? (
        <button
          onClick={() => onChange('')}
          className="absolute right-2.5 text-muted-foreground hover:text-foreground"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      ) : null}
    </div>
  );
}

// ─── Trigger button ──────────────────────────────────────────────────────────

interface TriggerProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'value'> {
  value?: Bank | null;
  placeholder: string;
  disabled: boolean;
  className?: string;
  onClear: (e: React.MouseEvent) => void;
}

const SelectorTrigger = forwardRef<HTMLButtonElement, TriggerProps>(
  ({ value, placeholder, disabled, className, onClear, ...props }, ref) => (
    <Button
      ref={ref}
      variant="outline"
      role="combobox"
      className={cn("w-full justify-between", className)}
      disabled={disabled}
      {...props}
    >
      <span className="truncate">{value ? value.branch_name : placeholder}</span>
      <div className="ml-2 flex items-center gap-1 shrink-0">
        {value && (
          <span
            role="button"
            tabIndex={0}
            onClick={onClear}
            onKeyDown={(e) => { if (e.key === 'Enter') onClear(e as unknown as React.MouseEvent); }}
            className="rounded-full p-0.5 hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="h-3 w-3" />
          </span>
        )}
        <ChevronsUpDown className="h-4 w-4 opacity-50" />
      </div>
    </Button>
  )
);

// ─── Main component ──────────────────────────────────────────────────────────

export function BankSelector({
  value,
  onSelect,
  placeholder = "Chọn ngân hàng...",
  disabled = false,
  className,
  canCreateBank = true
}: BankSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const searchInputRef = useRef<HTMLInputElement>(null);
  const isMobile = useIsMobile();

  const handleSelect = (bank: Bank) => {
    onSelect(bank);
    setOpen(false);
  };

  const handleClear = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSelect(null);
  };

  const handleOpenChange = (next: boolean) => {
    setOpen(next);
    if (!next) setSearchValue('');
  };

  useEffect(() => {
    if (open) setTimeout(() => searchInputRef.current?.focus(), 80);
  }, [open]);

  // ── Mobile: bottom drawer ──
  if (isMobile) {
    return (
      <Drawer open={open} onOpenChange={handleOpenChange}>
        <DrawerTrigger asChild>
          <SelectorTrigger
            value={value}
            placeholder={placeholder}
            disabled={disabled}
            className={className}
            onClear={handleClear}
          />
        </DrawerTrigger>
        <DrawerContent className="flex flex-col">
          <DrawerHeader className="flex items-center gap-3 py-3 px-4 border-b shrink-0">
            <DrawerTitle className="text-sm font-semibold shrink-0">Chọn ngân hàng</DrawerTitle>
            <SearchBar
              value={searchValue}
              onChange={setSearchValue}
              inputRef={searchInputRef}
            />
          </DrawerHeader>
          <BankList
            value={value}
            onSelect={handleSelect}
            canCreateBank={canCreateBank}
            searchValue={searchValue}
            scrollHeight="55vh"
          />
        </DrawerContent>
      </Drawer>
    );
  }

  // ── Desktop: dialog ──
  return (
    <>
      <SelectorTrigger
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        className={className}
        onClear={handleClear}
        onClick={() => setOpen(true)}
      />
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent
          className="flex flex-col sm:max-w-[520px] max-w-[calc(100vw-2rem)] p-0 gap-0 max-h-[70vh] overflow-hidden"
          title="Chọn ngân hàng"
          hideCloseButton={true}
        >
          {/* Header */}
          <div className="flex items-center gap-3 px-4 py-3 border-b shrink-0">
            <DialogTitle className="text-sm font-semibold shrink-0 text-foreground">Chọn ngân hàng</DialogTitle>
            <SearchBar
              value={searchValue}
              onChange={setSearchValue}
              inputRef={searchInputRef}
            />
            <DialogClose asChild>
              <Button variant="ghost" size="sm" className="shrink-0">
                Đóng
              </Button>
            </DialogClose>
          </div>

          {/* Bank list */}
          <div className="flex-1 min-h-0 flex flex-col">
            <BankList
              value={value}
              onSelect={handleSelect}
              canCreateBank={canCreateBank}
              searchValue={searchValue}
              scrollHeight="100%"
            />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
