import { useState, useMemo, memo, useRef, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Loader2,
  Upload,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  SkipForward,
  FileSpreadsheet,
  ChevronDown,
  PartyPopper,
  Check,
  X,
  FileText,
  Search,
  Info,
} from 'lucide-react';
import {
  useBCCUploadModal,
  formatMonthLabel,
  buildFailureSummary,
  groupErrorsByEmployee,
  parseResultErrors,
  getMonthOptions,
} from '@/hooks/timesheet/useBCCUploadModal';
import type { PartnerImportFile } from '@/types/api/timesheet.types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface ProjectOption {
  id: number;
  name: string;
}

interface BCCUploadModalProps {
  open: boolean;
  onClose: () => void;
  projectId: number;
  projects?: ProjectOption[];
}

// ─── Searchable Project Combobox ──────────────────────────────────────────────

const EMPTY_PROJECTS: ProjectOption[] = [];

interface ProjectComboboxProps {
  projects: ProjectOption[];
  value: string;
  onChange: (value: string) => void;
  hasProject: boolean;
}

const ProjectCombobox = memo(function ProjectCombobox({
  projects,
  value,
  onChange,
  hasProject,
}: ProjectComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const filtered = useMemo(() => {
    if (!search.trim()) return projects;
    const q = search.toLowerCase();
    return projects.filter((p) => p.name.toLowerCase().includes(q));
  }, [projects, search]);

  const selectedName = useMemo(
    () => projects.find((p) => String(p.id) === value)?.name,
    [projects, value],
  );

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  // Focus input when opened
  useEffect(() => {
    if (open) setTimeout(() => inputRef.current?.focus(), 0);
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={`block w-full rounded-lg border px-3 py-2 text-left text-sm leading-6 font-medium transition-colors placeholder:text-slate-400 focus:outline-none focus:ring-2
          ${hasProject
            ? 'border-emerald-500 bg-white text-slate-900'
            : !value
              ? 'border-slate-200 bg-white text-slate-500 hover:border-slate-300'
              : 'border-slate-200 bg-white text-slate-900 hover:border-slate-300'
          }
          ${open ? 'border-emerald-500 ring-2 ring-emerald-500/40' : ''}
        `}
      >
        <span className={selectedName ? 'text-slate-900' : 'text-slate-500'}>
          {selectedName ?? 'Chọn dự án...'}
        </span>
        <ChevronDown
          className={`float-right mt-1 h-4 w-4 text-slate-400 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1.5 w-full overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg">
          {/* Search input */}
          <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2.5">
            <Search className="h-3.5 w-3.5 shrink-0 text-slate-400" />
            <input
              ref={inputRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Tìm dự án..."
              className="flex-1 bg-transparent text-sm font-medium text-slate-900 placeholder:text-slate-400 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-400 hover:text-slate-700">
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-[220px] overflow-y-auto py-1">
            {filtered.length === 0 ? (
              <p className="px-4 py-3 text-sm text-slate-500">Không tìm thấy dự án.</p>
            ) : (
              filtered.map((p) => {
                const isSelected = String(p.id) === value;
                return (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => {
                      onChange(String(p.id));
                      setOpen(false);
                      setSearch('');
                    }}
                    className={`flex w-full items-center justify-between px-3 py-2 text-sm font-medium transition-colors text-left
                      ${isSelected
                        ? 'bg-emerald-50 text-emerald-700'
                        : 'text-slate-800 hover:bg-slate-50'
                      }
                    `}
                  >
                    <span>{p.name}</span>
                    {isSelected && <Check className="h-3.5 w-3.5 shrink-0" strokeWidth={2.5} />}
                  </button>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
});

// ─── Result Sub-components (UI only) ──────────────────────────────────────────

interface ResultSuccessProps {
  result: PartnerImportFile;
}

const ResultSuccess = memo(function ResultSuccess({ result }: ResultSuccessProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-emerald-200 bg-white shadow-sm dark:border-emerald-700/50 dark:bg-slate-800">
      {/* Accent header */}
      <div className="flex items-start gap-3 border-l-4 border-emerald-500 bg-emerald-50 p-5 dark:bg-emerald-900/20">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
          <CheckCircle2 className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <h4 className="text-base font-semibold text-emerald-900 dark:text-emerald-200">
            Import thành công!
          </h4>
          <p className="mt-0.5 text-sm text-emerald-700 dark:text-emerald-300/80">
            Tháng {formatMonthLabel(result.for_month)} · {result.created_count} bản chấm công đã tạo
          </p>
        </div>
        <PartyPopper className="h-5 w-5 shrink-0 text-emerald-500" />
      </div>

      {(result.created_count > 0 || result.skipped_count > 0) && (
        <div className="flex flex-wrap gap-2 p-4">
          {result.created_count > 0 && (
            <span className="inline-flex items-center gap-1.5 rounded-md border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:border-emerald-700/50 dark:bg-emerald-900/20 dark:text-emerald-300">
              <CheckCircle2 className="h-3 w-3" />
              {result.created_count} tạo mới
            </span>
          )}
          {result.skipped_count > 0 && (
            <span className="inline-flex items-center gap-1.5 rounded-md border border-amber-200 bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:border-amber-700/50 dark:bg-amber-900/20 dark:text-amber-300">
              <SkipForward className="h-3 w-3" />
              {result.skipped_count} bỏ qua
            </span>
          )}
        </div>
      )}
    </div>
  );
});

interface ResultFailureProps {
  result: PartnerImportFile;
}

const ResultFailure = memo(function ResultFailure({ result }: ResultFailureProps) {
  const [showErrors, setShowErrors] = useState(false);
  const parsedErrors = useMemo(() => parseResultErrors(result.error_detail), [result.error_detail]);
  const summaryText = useMemo(
    () => buildFailureSummary(parsedErrors, result.for_month),
    [parsedErrors, result.for_month],
  );
  const groupedErrors = useMemo(() => groupErrorsByEmployee(parsedErrors), [parsedErrors]);

  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-lg border border-rose-200 bg-white shadow-sm dark:border-rose-700/50 dark:bg-slate-800">
        <div className="flex items-start gap-3 border-l-4 border-rose-500 bg-rose-50 p-5 dark:bg-rose-900/20">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300">
            <XCircle className="h-5 w-5" />
          </div>
          <div className="min-w-0 flex-1">
            <h4 className="text-base font-semibold text-rose-900 dark:text-rose-200">
              Import thất bại
            </h4>
            <p className="mt-1 text-sm leading-relaxed text-rose-700 dark:text-rose-300/80">{summaryText}</p>
          </div>
        </div>
      </div>

      {(result.created_count > 0 || result.skipped_count > 0 || result.error_count > 0) && (
        <div className="flex flex-wrap gap-2">
          {result.created_count > 0 && (
            <div className="flex items-center gap-2 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-1.5 dark:border-emerald-700/50 dark:bg-emerald-900/20">
              <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600" />
              <span className="font-bold text-emerald-800 dark:text-emerald-200">{result.created_count}</span>
              <span className="text-sm text-emerald-700 dark:text-emerald-300/80">tạo mới</span>
            </div>
          )}
          {result.skipped_count > 0 && (
            <div className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-1.5 dark:border-amber-700/50 dark:bg-amber-900/20">
              <SkipForward className="h-4 w-4 shrink-0 text-amber-600" />
              <span className="font-bold text-amber-800 dark:text-amber-200">{result.skipped_count}</span>
              <span className="text-sm text-amber-700 dark:text-amber-300/80">bỏ qua</span>
            </div>
          )}
          {result.error_count > 0 && (
            <div className="flex items-center gap-2 rounded-md border border-rose-200 bg-rose-50 px-3 py-1.5 dark:border-rose-700/50 dark:bg-rose-900/20">
              <AlertTriangle className="h-4 w-4 shrink-0 text-rose-600" />
              <span className="font-bold text-rose-800 dark:text-rose-200">{result.error_count}</span>
              <span className="text-sm text-rose-700 dark:text-rose-300/80">lỗi</span>
            </div>
          )}
        </div>
      )}

      {parsedErrors.length > 0 && (
        <div className="space-y-2">
          <button
            onClick={() => setShowErrors((v) => !v)}
            className="flex items-center gap-1.5 text-sm font-medium text-rose-700 transition-colors hover:text-rose-900 dark:text-rose-300 dark:hover:text-rose-200"
          >
            <span>{showErrors ? 'Ẩn chi tiết lỗi' : `Xem ${parsedErrors.length} lỗi chi tiết`}</span>
            <ChevronDown
              className={`h-4 w-4 transition-transform duration-200 ${showErrors ? 'rotate-180' : ''}`}
            />
          </button>

          {showErrors && (
            <div className="max-h-52 overflow-y-auto rounded-lg border border-rose-200 bg-white divide-y divide-rose-100 dark:border-rose-700/50 dark:bg-slate-800 dark:divide-rose-900/30">
              {Array.from(groupedErrors.entries()).map(([employee, errors]) => (
                <div key={employee} className="px-4 py-3">
                  <p className="mb-1 text-sm font-semibold text-slate-900 dark:text-slate-100">{employee}</p>
                  <ul className="space-y-1">
                    {errors.map((e, i) => (
                      <li
                        key={i}
                        className="border-l-2 border-rose-300 pl-3 text-sm leading-relaxed text-rose-700 dark:border-rose-700 dark:text-rose-300/80"
                      >
                        {e.reason}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
});

// ─── Main Component (UI only) ─────────────────────────────────────────────────

export const BCCUploadModal = memo(function BCCUploadModal({
  open,
  onClose,
  projectId,
  projects,
}: BCCUploadModalProps) {
  const {
    file,
    result,
    selectedProjectId,
    selectedMonth,
    isDragging,
    isPending,
    needsProjectSelect,
    hasProject,
    canUpload,
    hintText,
    isReady,
    setSelectedProjectId,
    setSelectedMonth,
    handleFileChange,
    handleUpload,
    handleClose,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleReset,
    handleRemoveFile,
  } = useBCCUploadModal({ projectId, projects, onClose });

  const monthOptions = useMemo(() => getMonthOptions(), []);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent
        className="max-w-lg overflow-hidden rounded-lg p-0 dark:bg-slate-800 dark:text-slate-100"
        contentPadding="none"
        hideCloseButton
      >
        {/* ── Header (Tailkit modal-head pattern) ─────────────────────────── */}
        <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50 px-5 py-4 dark:border-slate-700 dark:bg-slate-800/50">
          <DialogTitle asChild>
            <h3 className="flex items-center gap-2.5 font-semibold text-slate-900 dark:text-slate-100">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                <FileText className="h-5 w-5" />
              </span>
              <span className="flex flex-col leading-tight">
                <span className="text-[15px] font-semibold">Tải lên Bảng Chấm Công</span>
                <span className="text-[12px] font-normal text-slate-500 dark:text-slate-400">
                  Nhập dữ liệu chấm công từ file Excel
                </span>
              </span>
            </h3>
          </DialogTitle>
          <button
            onClick={handleClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-transparent text-slate-500 transition-colors hover:bg-slate-200/60 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
            title="Đóng"
            aria-label="Đóng"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* ── Body ────────────────────────────────────────────────────────── */}
        <div className="space-y-5 p-5">
          {!result && (
            <>
              {/* Project select */}
              {needsProjectSelect && (
                <div className="space-y-1.5">
                  <label className="flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-300">
                    Chọn dự án
                    <span className="text-emerald-600">*</span>
                    {hasProject && (
                      <span className="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                        <Check className="h-3 w-3" strokeWidth={3} />
                        Đã chọn
                      </span>
                    )}
                  </label>
                  <ProjectCombobox
                    projects={projects ?? EMPTY_PROJECTS}
                    value={selectedProjectId}
                    onChange={setSelectedProjectId}
                    hasProject={hasProject}
                  />
                </div>
              )}

              {/* Month select */}
              <div className="space-y-1.5">
                <label className="flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-300">
                  Tháng áp dụng
                  <span className="text-emerald-600">*</span>
                  <span className="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
                    <Check className="h-3 w-3" strokeWidth={3} />
                    Đã chọn
                  </span>
                </label>
                <Select value={selectedMonth} onValueChange={setSelectedMonth}>
                  <SelectTrigger className="h-auto rounded-lg border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-900 transition-colors hover:border-slate-300 focus:ring-2 focus:ring-emerald-500/40 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {monthOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                        {opt.isCurrent ? ' (tháng này)' : ''}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* Info callout (Tailkit alert pattern) */}
              <div className="flex items-start gap-2.5 rounded-lg border border-emerald-200 bg-emerald-50 p-3.5 text-sm leading-relaxed text-emerald-800 dark:border-emerald-700/50 dark:bg-emerald-900/20 dark:text-emerald-200/90">
                <Info className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
                <p className="flex-1">
                  Các ngày trong file Excel sẽ được gán vào tháng đã chọn — ví dụ ngày{' '}
                  <code className="rounded bg-emerald-600/10 px-1 py-px font-mono text-[12px] font-semibold">
                    22
                  </code>{' '}
                  sẽ thành{' '}
                  <code className="rounded bg-emerald-600/10 px-1 py-px font-mono text-[12px] font-semibold">
                    22/{selectedMonth.split('-')[1]}/{selectedMonth.split('-')[0]}
                  </code>
                  .
                </p>
              </div>

              {/* Upload area */}
              <div className="space-y-1.5">
                <label className="flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-300">
                  Tệp bảng chấm công
                  <span className="text-emerald-600">*</span>
                </label>

                {/* Dropzone */}
                {!file && (
                  <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={() => !isPending && document.getElementById('bcc-file')?.click()}
                    className={`flex cursor-pointer flex-col items-center gap-3 rounded-lg border-2 border-dashed px-5 py-9 text-center transition-all duration-200
                      ${isDragging
                        ? 'border-emerald-500 bg-emerald-50 scale-[1.005]'
                        : 'border-slate-300 bg-slate-50/50 hover:border-emerald-400 hover:bg-emerald-50/40'
                      }
                    `}
                  >
                    <span className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                      <Upload className="h-5 w-5" />
                    </span>
                    <div>
                      <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                        Kéo thả file <span className="text-emerald-700 dark:text-emerald-400">BCC</span> vào đây
                      </p>
                      <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                        hoặc nhấn để chọn file · định dạng .xlsx · tối đa 10 MB
                      </p>
                    </div>
                    <input
                      id="bcc-file"
                      type="file"
                      accept=".xlsx"
                      onChange={handleFileChange}
                      disabled={isPending}
                      className="hidden"
                    />
                  </div>
                )}

                {/* File card */}
                {file && (
                  <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white p-3 shadow-sm animate-in fade-in slide-in-from-bottom-1 duration-200 dark:border-slate-700 dark:bg-slate-900">
                    <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                      <FileSpreadsheet className="h-5 w-5" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">
                        {file.name}
                      </p>
                      <p className="mt-0.5 font-mono text-xs text-slate-500 dark:text-slate-400">
                        {(file.size / 1024).toFixed(0)} KB
                      </p>
                    </div>
                    <button
                      onClick={handleRemoveFile}
                      title="Xóa file"
                      aria-label="Xóa file"
                      className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-white text-slate-500 transition-colors hover:border-rose-300 hover:bg-rose-50 hover:text-rose-700 dark:border-slate-600 dark:bg-slate-800 dark:hover:border-rose-700 dark:hover:bg-rose-900/30 dark:hover:text-rose-300"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                )}

                {/* Progress bar */}
                {isPending && (
                  <div className="mt-2.5 animate-in fade-in duration-200">
                    <div className="h-1.5 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700">
                      <div
                        className="h-full animate-pulse rounded-full bg-gradient-to-r from-emerald-600 to-emerald-500"
                        style={{ width: '60%' }}
                      />
                    </div>
                    <p className="mt-1.5 flex items-center justify-end gap-1 text-xs font-semibold text-slate-600 dark:text-slate-400">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Đang tải lên…
                    </p>
                  </div>
                )}
              </div>
            </>
          )}

          {/* Result states */}
          {result &&
            (result.status === 'completed' ? (
              <ResultSuccess result={result} />
            ) : (
              <ResultFailure result={result} />
            ))}
        </div>

        {/* ── Footer (Tailkit modal footer pattern) ──────────────────────── */}
        <div className="flex items-center gap-3 border-t border-slate-100 bg-slate-50 px-5 py-4 dark:border-slate-700 dark:bg-slate-800/50">
          {!result && (
            <div
              className={`flex min-w-0 flex-1 items-center gap-1.5 ${isReady ? 'text-emerald-700 dark:text-emerald-400' : 'text-slate-500 dark:text-slate-400'}`}
            >
              {isReady ? (
                <Check className="h-3.5 w-3.5 shrink-0" strokeWidth={2.5} />
              ) : (
                <svg className="h-3.5 w-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
              )}
              <span className="truncate text-xs font-medium">{hintText}</span>
            </div>
          )}
          {result && <div className="flex-1" />}

          <button
            onClick={handleClose}
            disabled={isPending}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm leading-5 font-semibold text-slate-700 transition-colors hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          >
            {result ? 'Đóng' : 'Hủy'}
          </button>

          {!result && (
            <button
              onClick={handleUpload}
              disabled={!canUpload || isPending}
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-700 bg-emerald-700 px-4 py-2 text-sm leading-5 font-semibold text-white transition-colors hover:border-emerald-600 hover:bg-emerald-600 disabled:cursor-not-allowed disabled:border-slate-300 disabled:bg-slate-300 disabled:text-white"
            >
              {isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" />
                  Tải lên
                </>
              )}
            </button>
          )}

          {result && (
            <button
              onClick={handleReset}
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-700 bg-emerald-700 px-4 py-2 text-sm leading-5 font-semibold text-white transition-colors hover:border-emerald-600 hover:bg-emerald-600"
            >
              <Upload className="h-4 w-4" />
              Upload khác
            </button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
});
