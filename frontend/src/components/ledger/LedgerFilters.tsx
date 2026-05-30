import { SearchBar } from '@/components/shared/SearchBar';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger, DropdownMenuCheckboxItem } from '@/components/ui/dropdown-menu';
import { RotateCw, Settings, Calendar, Filter, Bookmark, X, ChevronDown } from 'lucide-react';
import { ledgerService } from '@/services/api/ledger.service';
import { dateToString } from '@/utils/dateHelpers';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { useState, useMemo } from 'react';
import type { LedgerFilters as FiltersType, AccountMetadata } from '@/types/api/financial.types';

interface Project { id: number; name: string; }
interface User { id: number; full_name: string; email: string; }

interface LedgerFiltersProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  projects: Project[];
  users?: User[];
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
  accountMetadata?: AccountMetadata[];
  isLoadingAccountMetadata?: boolean;
}

const DATE_PRESETS = [
  { label: 'Hôm nay', value: 'today' },
  { label: 'Tuần này', value: 'this-week' },
  { label: 'Tháng này', value: 'this-month' },
  { label: 'Tháng trước', value: 'last-month' },
  { label: 'Quý này', value: 'this-quarter' },
  { label: '6 tháng', value: 'last-6-months' },
  { label: 'Năm này', value: 'this-year' },
];

function getDatePreset(preset: string) {
  const now = new Date();
  switch (preset) {
    case 'today': return { fromDate: dateToString(now), toDate: dateToString(now) };
    case 'this-week': {
      const s = new Date(now); s.setDate(now.getDate() - now.getDay());
      const e = new Date(s); e.setDate(s.getDate() + 6);
      return { fromDate: dateToString(s), toDate: dateToString(e) };
    }
    case 'this-month': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
    };
    case 'last-month': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 0)),
    };
    case 'this-quarter': {
      const q = Math.floor(now.getMonth() / 3);
      return {
        fromDate: dateToString(new Date(now.getFullYear(), q * 3, 1)),
        toDate: dateToString(new Date(now.getFullYear(), q * 3 + 3, 0)),
      };
    }
    case 'this-year': return {
      fromDate: dateToString(new Date(now.getFullYear(), 0, 1)),
      toDate: dateToString(new Date(now.getFullYear(), 11, 31)),
    };
    case 'last-6-months': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth() - 6, 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
    };
    default: return null;
  }
}

export function LedgerFilters({
  filters,
  onFiltersChange,
  projects,
  users = [],
  onClearFilters,
  hasFilters,
  accountMetadata = [],
  isLoadingAccountMetadata = false,
}: LedgerFiltersProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [presetName, setPresetName] = useState('');
  const [savedPresets, setSavedPresets] = useState<Record<string, FiltersType>>(() => {
    try { return JSON.parse(localStorage.getItem('ledger-filter-presets') || '{}'); }
    catch { return {}; }
  });

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  const accountOptions = useMemo(() =>
    accountMetadata.length > 0
      ? ledgerService.getAccountOptionsFromMetadata(accountMetadata)
      : ledgerService.getAccountOptions(),
    [accountMetadata]
  );

  const selectedAccounts = useMemo(() =>
    filters.account ? filters.account.split(',') : [],
    [filters.account]
  );

  const handleAccountToggle = (val: string) => {
    const updated = selectedAccounts.includes(val)
      ? selectedAccounts.filter(a => a !== val)
      : [...selectedAccounts, val];
    set('account', updated.length > 0 ? updated.join(',') : undefined);
  };

  const activeFilterCount = useMemo(() => {
    return [filters.account, filters.project_id, filters.party, filters.created_by, filters.has_evidence !== undefined]
      .filter(Boolean).length;
  }, [filters]);

  const handleSavePreset = () => {
    if (!presetName.trim()) return;
    const next = { ...savedPresets, [presetName]: filters };
    setSavedPresets(next);
    localStorage.setItem('ledger-filter-presets', JSON.stringify(next));
    setPresetName('');
  };

  const handleDeletePreset = (key: string) => {
    const next = { ...savedPresets };
    delete next[key];
    setSavedPresets(next);
    localStorage.setItem('ledger-filter-presets', JSON.stringify(next));
  };

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      {/* Search */}
      <SearchBar
        searchTerm={filters.party || ''}
        onSearchChange={(v) => set('party', v || undefined)}
        placeholder="Tìm diễn giải, đối tượng..."
        className="w-52"
      />

      {/* Date preset */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button className="flex items-center gap-1.5 h-8 px-2.5 text-xs rounded-lg border border-input bg-background hover:bg-accent transition-colors">
            <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Thời gian</span>
            <ChevronDown className="h-3 w-3 text-muted-foreground" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-40 p-1">
          {DATE_PRESETS.map((p) => (
            <button
              key={p.value}
              className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-accent transition-colors"
              onClick={() => {
                const range = getDatePreset(p.value);
                if (range) onFiltersChange({ ...filters, ...range, page: 1 });
              }}
            >
              {p.label}
            </button>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Date range inputs */}
      <DateRangePicker
        startDate={filters.fromDate}
        endDate={filters.toDate}
        onStartDateChange={(date) => set('fromDate', date)}
        onEndDateChange={(date) => set('toDate', date)}
      />

      {/* Advanced filters popover */}
      <Popover open={showAdvanced} onOpenChange={setShowAdvanced}>
        <PopoverTrigger asChild>
          <button className="relative flex items-center gap-1.5 h-8 px-2.5 text-xs rounded-lg border border-input bg-background hover:bg-accent transition-colors">
            <Filter className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Bộ lọc</span>
            {activeFilterCount > 0 && (
              <Badge className="absolute -top-1.5 -right-1.5 h-4 w-4 p-0 text-[10px] flex items-center justify-center rounded-full">
                {activeFilterCount}
              </Badge>
            )}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-80" align="start">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">Bộ lọc nâng cao</p>
              {Object.keys(savedPresets).length > 0 && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <button className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                      <Bookmark className="h-3 w-3" />
                      Đã lưu
                    </button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-44">
                    {Object.entries(savedPresets).map(([key, preset]) => (
                      <div key={key} className="flex items-center">
                        <button
                          className="flex-1 text-left px-2 py-1.5 text-xs hover:bg-accent rounded"
                          onClick={() => { onFiltersChange({ ...preset, page: 1 }); setShowAdvanced(false); }}
                        >
                          {key}
                        </button>
                        <button
                          className="p-1 text-destructive hover:text-destructive/80"
                          onClick={() => handleDeletePreset(key)}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </div>

            {/* Account type */}
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Loại tài khoản</Label>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="w-full justify-between h-8 text-xs" disabled={isLoadingAccountMetadata}>
                    <span className="truncate">
                      {isLoadingAccountMetadata ? 'Đang tải...' : selectedAccounts.length === 0 ? 'Tất cả' : `${selectedAccounts.length} loại`}
                    </span>
                    <Settings className="h-3.5 w-3.5 shrink-0" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-64">
                  {accountOptions.map((o) => (
                    <DropdownMenuCheckboxItem
                      key={o.value}
                      checked={selectedAccounts.includes(o.value)}
                      onCheckedChange={() => handleAccountToggle(o.value)}
                    >
                      {o.label}
                    </DropdownMenuCheckboxItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            {/* Project */}
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Dự án</Label>
              <Select
                value={filters.project_id?.toString() || 'all'}
                onValueChange={(v) => set('project_id', v === 'all' ? undefined : parseInt(v))}
              >
                <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="Tất cả" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả</SelectItem>
                  {projects.map((p) => <SelectItem key={p.id} value={p.id.toString()}>{p.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>

            {/* Created by */}
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Người tạo</Label>
              <Select
                value={filters.created_by?.toString() || 'all'}
                onValueChange={(v) => set('created_by', v === 'all' ? undefined : parseInt(v))}
              >
                <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="Tất cả" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả</SelectItem>
                  {users.map((u) => <SelectItem key={u.id} value={u.id.toString()}>{u.full_name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>

            {/* Evidence */}
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Chứng từ</Label>
              <Select
                value={filters.has_evidence === undefined ? 'all' : String(filters.has_evidence)}
                onValueChange={(v) => set('has_evidence', v === 'all' ? undefined : v === 'true')}
              >
                <SelectTrigger className="h-8 text-xs"><SelectValue placeholder="Tất cả" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả</SelectItem>
                  <SelectItem value="true">Có chứng từ</SelectItem>
                  <SelectItem value="false">Không có chứng từ</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Save preset */}
            <div className="flex gap-1.5 border-t pt-3">
              <Input
                placeholder="Tên bộ lọc..."
                value={presetName}
                onChange={(e) => setPresetName(e.target.value)}
                className="flex-1 h-8 text-xs"
              />
              <Button size="sm" onClick={handleSavePreset} disabled={!presetName.trim()} className="h-8 text-xs px-2">
                <Bookmark className="h-3 w-3" />
              </Button>
            </div>

            {hasFilters && (
              <button
                onClick={() => { onClearFilters(); setShowAdvanced(false); }}
                className="flex items-center gap-1.5 w-full justify-center h-8 text-xs text-muted-foreground hover:text-foreground border rounded-xl transition-colors"
              >
                <RotateCw className="h-3 w-3" />
                Xóa tất cả bộ lọc
              </button>
            )}
          </div>
        </PopoverContent>
      </Popover>

      {hasFilters && (
        <button
          onClick={onClearFilters}
          className="flex items-center gap-1 h-8 px-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <X className="h-3 w-3" />
          Xóa lọc
        </button>
      )}
    </div>
  );
}
