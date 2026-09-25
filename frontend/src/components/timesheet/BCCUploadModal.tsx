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
import { Checkbox } from '@/components/ui/checkbox';
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
import { groupImportErrors, describeGroupedError } from '@/utils/import-errors';
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
	allowFlexibleEmployeeImport?: boolean;
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
        className={`block w-full rounded-lg border px-3 py-2 text-left text-sm leading-6 font-medium transition-colors placeholder:text-slate-500 focus:outline-none focus:ring-2
          ${hasProject
            ? 'border-emerald-500 bg-white text-slate-900'
            : !value
              ? 'border-slate-300 bg-white text-slate-500 hover:border-slate-300'
              : 'border-slate-300 bg-white text-slate-900 hover:border-slate-300'
          }
          ${open ? 'border-emerald-500 ring-2 ring-emerald-500/40' : ''}
        `}
      >
        <span className={selectedName ? 'text-slate-900' : 'text-slate-500'}>
          {selectedName ?? 'Chọn dự án...'}
        </span>
        <ChevronDown
          className={`float-right mt-1 h-4 w-4 text-slate-500 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1.5 w-full overflow-hidden rounded-lg border border-slate-300 bg-white shadow-lg">
          {/* Search input */}
          <div className="flex items-center gap-2 border-b border-slate-200 px-3 py-2.5">
            <Search className="h-3.5 w-3.5 shrink-0 text-slate-500" />
            <input
              ref={inputRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Tìm dự án..."
              className="flex-1 bg-transparent text-sm font-medium text-slate-900 placeholder:text-slate-500 focus:outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-slate-500 hover:text-slate-700">
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

interface ResultCountsProps {
  result: PartnerImportFile;
  showErrorCount?: boolean;
}

const ResultCounts = memo(function ResultCounts({
  result,
  showErrorCount = false,
}: ResultCountsProps) {
  return (
    <dl
      className={`grid gap-2 ${showErrorCount ? 'grid-cols-1 min-[380px]:grid-cols-3' : 'grid-cols-2'}`}
      aria-label="Kết quả nhập bảng chấm công"
    >
      <div className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 dark:border-emerald-700/50 dark:bg-emerald-900/20">
        <dt className="flex items-center gap-1.5 text-xs font-medium text-emerald-700 dark:text-emerald-300">
          <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
          Tạo mới
        </dt>
        <dd className="mt-0.5 text-base font-bold tabular-nums text-emerald-900 dark:text-emerald-100">
          {result.created_count}
        </dd>
      </div>
      <div className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 dark:border-amber-700/50 dark:bg-amber-900/20">
        <dt className="flex items-center gap-1.5 text-xs font-medium text-amber-700 dark:text-amber-300">
          <SkipForward className="h-3.5 w-3.5" aria-hidden="true" />
          Bỏ qua
        </dt>
        <dd className="mt-0.5 text-base font-bold tabular-nums text-amber-900 dark:text-amber-100">
          {result.skipped_count}
        </dd>
      </div>
      {showErrorCount && (
        <div className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 dark:border-rose-700/50 dark:bg-rose-900/20">
          <dt className="flex items-center gap-1.5 text-xs font-medium text-rose-700 dark:text-rose-300">
            <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />
            Lỗi
          </dt>
          <dd className="mt-0.5 text-base font-bold tabular-nums text-rose-900 dark:text-rose-100">
            {result.error_count}
          </dd>
        </div>
      )}
    </dl>
  );
});

interface ResultErrorDetailsProps {
  result: PartnerImportFile;
}

const ResultErrorDetails = memo(function ResultErrorDetails({
  result,
}: ResultErrorDetailsProps) {
  const [showErrors, setShowErrors] = useState(false);
  const parsedErrors = useMemo(
    () => parseResultErrors(result.error_detail),
    [result.error_detail],
  );
  const groupedErrors = useMemo(
    () => groupErrorsByEmployee(parsedErrors),
    [parsedErrors],
  );

  if (parsedErrors.length === 0) return null;

  return (
    <div className="space-y-2">
      <button
        type="button"
        onClick={() => setShowErrors((value) => !value)}
        aria-expanded={showErrors}
        aria-controls="bcc-import-error-details"
        className="flex min-h-11 items-center gap-1.5 rounded-md text-sm font-medium text-rose-700 transition-colors hover:text-rose-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500 dark:text-rose-300 dark:hover:text-rose-200"
      >
        <span>
          {showErrors
            ? 'Ẩn chi tiết cần xử lý'
            : `Xem ${parsedErrors.length} lỗi cần xử lý`}
        </span>
        <ChevronDown
          className={`h-4 w-4 transition-transform duration-200 motion-reduce:transition-none ${showErrors ? 'rotate-180' : ''}`}
          aria-hidden="true"
        />
      </button>

      {showErrors && (
        <div
          id="bcc-import-error-details"
          className="max-h-52 divide-y divide-rose-100 overflow-y-auto overscroll-contain rounded-lg border border-rose-200 bg-white dark:divide-rose-900/30 dark:border-rose-700/50 dark:bg-slate-800"
        >
          {Array.from(groupedErrors.entries()).map(([employee, errors]) => (
            <div key={employee} className="px-4 py-3">
              <p className="mb-1 break-words text-sm font-semibold text-slate-900 dark:text-slate-100">
                {employee}
              </p>
              <ul className="space-y-1">
                {groupImportErrors(errors).map((group) => {
                  const detail = describeGroupedError(group);
                  return (
                    <li
                      key={`${group.employee}-${group.reason}`}
                      className="break-words border-l-2 border-rose-300 pl-3 text-sm leading-relaxed text-rose-700 dark:border-rose-700 dark:text-rose-300/80"
                    >
                      {group.reason}
                      {detail && (
                        <span className="font-normal text-rose-600/90 dark:text-rose-300/70">
                          {detail}
                        </span>
                      )}
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
});

const ResultSuccess = memo(function ResultSuccess({ result }: ResultSuccessProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3 rounded-lg border border-emerald-200 border-l-4 border-l-emerald-500 bg-emerald-50 p-5 dark:border-emerald-700/50 dark:border-l-emerald-500 dark:bg-emerald-900/20">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
          <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <h4 className="text-base font-semibold text-emerald-900 dark:text-emerald-200">
            Nhập dữ liệu thành công
          </h4>
          <p className="mt-0.5 text-sm text-emerald-700 dark:text-emerald-300/80">
            Tháng {formatMonthLabel(result.for_month)} · Tất cả dữ liệu hợp lệ đã được nhập.
          </p>
        </div>
        <PartyPopper className="h-5 w-5 shrink-0 text-emerald-700" aria-hidden="true" />
      </div>
      <ResultCounts result={result} />
    </div>
  );
});

interface ResultFailureProps {
  result: PartnerImportFile;
}

const ResultPartial = memo(function ResultPartial({ result }: ResultFailureProps) {
  return (
    <div className="space-y-4" role="status" aria-live="polite">
      <div className="flex items-start gap-3 rounded-lg border border-amber-200 border-l-4 border-l-amber-500 bg-amber-50 p-5 dark:border-amber-700/50 dark:border-l-amber-500 dark:bg-amber-900/20">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
          <AlertTriangle className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <h4 className="text-base font-semibold text-amber-950 dark:text-amber-100">
            Nhập dữ liệu hoàn tất một phần
          </h4>
          <p className="mt-1 text-sm leading-relaxed text-amber-800 dark:text-amber-200/90">
            Dữ liệu hợp lệ đã được nhập. Kiểm tra các dòng lỗi để hoàn tất phần còn lại.
          </p>
        </div>
      </div>
      <ResultCounts result={result} showErrorCount />
      <ResultErrorDetails result={result} />
    </div>
  );
});

const ResultFailure = memo(function ResultFailure({ result }: ResultFailureProps) {
  const parsedErrors = useMemo(() => parseResultErrors(result.error_detail), [result.error_detail]);
  const summaryText = useMemo(
    () => buildFailureSummary(parsedErrors, result.for_month),
    [parsedErrors, result.for_month],
  );

  return (
    <div className="space-y-4" role="alert">
      <div className="overflow-hidden rounded-lg border border-rose-200 bg-white dark:border-rose-700/50 dark:bg-slate-800">
        <div className="flex items-start gap-3 border-l-4 border-rose-500 bg-rose-50 p-5 dark:bg-rose-900/20">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-300">
            <XCircle className="h-5 w-5" aria-hidden="true" />
          </div>
          <div className="min-w-0 flex-1">
            <h4 className="text-base font-semibold text-rose-900 dark:text-rose-200">
              Nhập dữ liệu thất bại
            </h4>
            <p className="mt-1 text-sm leading-relaxed text-rose-700 dark:text-rose-300/80">{summaryText}</p>
          </div>
        </div>
      </div>
      <ResultErrorDetails result={result} />
    </div>
  );
});

// ─── Main Component (UI only) ─────────────────────────────────────────────────

function ImportProcessingState({ status }: { status: 'pending' | 'processing' }) {
  const isQueued = status === 'pending';
  return (
    <div
      className="space-y-4 rounded-lg border border-emerald-200 bg-emerald-50 p-5 text-center dark:border-emerald-800 dark:bg-emerald-950/30"
      role="status"
      aria-live="polite"
      aria-atomic="true"
      aria-busy="true"
    >
      <Loader2
        className="mx-auto h-8 w-8 animate-spin text-emerald-700 motion-reduce:animate-none dark:text-emerald-300"
        aria-hidden="true"
      />
      <div>
        <p className="font-semibold text-slate-900 dark:text-slate-100">
          {isQueued ? 'Đã nhận tệp. Đang chờ xử lý…' : 'Đang xử lý bảng chấm công…'}
        </p>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          Bạn có thể đóng cửa sổ này. Hệ thống vẫn tiếp tục xử lý.
        </p>
      </div>
      <div
        className="h-1.5 overflow-hidden rounded-full bg-emerald-100 dark:bg-emerald-900"
        role="progressbar"
        aria-label={isQueued ? 'Tệp đang chờ xử lý' : 'Bảng chấm công đang được xử lý'}
        aria-valuetext={isQueued ? 'Đang chờ' : 'Đang xử lý'}
      >
        <div className="h-full w-1/3 animate-pulse rounded-full bg-emerald-600 motion-reduce:animate-none" />
      </div>
    </div>
  );
}

export const BCCUploadModal = memo(function BCCUploadModal({
  open,
  onClose,
	projectId,
	projects,
	allowFlexibleEmployeeImport,
}: BCCUploadModalProps) {
  const {
    file,
    result,
    selectedProjectId,
    selectedMonth,
    isDragging,
    isPending,
    isImportActive,
    needsProjectSelect,
    hasProject,
    canUpload,
    hintText,
		isReady,
		includeFlexibleEmployees,
		allowFlexibleEmployeeImport: canImportFlexibleEmployees,
    setSelectedProjectId,
		setSelectedMonth,
		handleIncludeFlexibleEmployeesChange,
    handleFileChange,
    handleUpload,
    handleClose,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleReset,
    handleRemoveFile,
	} = useBCCUploadModal({ projectId, projects, allowFlexibleEmployeeImport, onClose });

  const monthOptions = useMemo(() => getMonthOptions(), []);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent
        className="flex max-h-[calc(100dvh-2rem)] max-w-lg flex-col gap-0 overflow-hidden rounded-xl p-0 dark:bg-slate-800 dark:text-slate-100"
        contentPadding="none"
        hideCloseButton
      >
        {/* ── Header (Tailkit modal-head pattern) ─────────────────────────── */}
        <div className="flex shrink-0 items-center justify-between border-b border-slate-200 bg-slate-50 px-4 py-3 sm:px-5 sm:py-4 dark:border-slate-700 dark:bg-slate-800/50">
          <DialogTitle asChild className="text-slate-900 dark:text-slate-100">
            <h3 className="flex items-center gap-2.5 font-semibold text-slate-900 dark:text-slate-100">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                <FileText className="h-5 w-5" />
              </span>
              <span className="flex flex-col leading-tight">
                <span className="text-[15px] font-semibold">Tải lên Bảng Chấm Công</span>
                <span className="text-[12px] font-normal text-slate-500 dark:text-slate-400">
                  Nhập dữ liệu chấm công từ tệp Excel
                </span>
              </span>
            </h3>
          </DialogTitle>
          <button
            onClick={handleClose}
            className="inline-flex h-11 w-11 items-center justify-center rounded-lg border border-transparent text-slate-500 transition-colors hover:bg-slate-200/60 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-200"
            title="Đóng"
            aria-label="Đóng"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* ── Body ────────────────────────────────────────────────────────── */}
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4 sm:p-5">
          {!result && (
            <>
              {/* Project select */}
              {needsProjectSelect && (
                <div className="space-y-1.5">
                  <label className="flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-300">
                    Dự án áp dụng
                    <span className="text-emerald-700">*</span>
                    {hasProject && (
                      <span className="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-emerald-700">
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
                  <span className="text-emerald-700">*</span>
                  <span className="ml-auto inline-flex items-center gap-1 text-xs font-semibold text-emerald-700">
                    <Check className="h-3 w-3" strokeWidth={3} />
                    Đã chọn
                  </span>
                </label>
                <Select value={selectedMonth} onValueChange={setSelectedMonth}>
                  <SelectTrigger className="h-auto rounded-lg border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-900 transition-colors hover:border-slate-300 focus:ring-2 focus:ring-emerald-500/40 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100">
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
                <Info className="mt-0.5 h-4 w-4 shrink-0 text-emerald-700 dark:text-emerald-400" />
                <p className="flex-1">
                  Các ngày trong tệp Excel sẽ được gán vào tháng đã chọn — ví dụ ngày{' '}
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

			  {canImportFlexibleEmployees && (
				<label
					htmlFor="bcc-include-flexible-employees"
					className="flex min-h-11 cursor-pointer items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2.5 text-sm text-amber-950 transition-colors hover:border-amber-300 dark:border-amber-700/50 dark:bg-amber-900/20 dark:text-amber-100"
				>
					<Checkbox
						id="bcc-include-flexible-employees"
						checked={includeFlexibleEmployees}
						onCheckedChange={(checked) => handleIncludeFlexibleEmployeesChange(checked === true)}
						className="mt-0.5 size-5 border-amber-600 data-[state=checked]:bg-amber-700 data-[state=checked]:text-white"
					/>
					<span className="min-w-0 leading-relaxed">
						<span className="block font-semibold">Import lương linh hoạt</span>
						<span className="block text-xs text-amber-800 dark:text-amber-200/90">
							Chỉ tạo ngày chưa có.
						</span>
					</span>
				</label>
			  )}

              {/* Upload area */}
              <div className="space-y-1.5">
                <label className="flex items-center gap-1 text-sm font-medium text-slate-700 dark:text-slate-300">
                  Tệp bảng chấm công
                  <span className="text-emerald-700">*</span>
                </label>

                {/* Dropzone */}
                {!file && (
                  <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={() => !isPending && document.getElementById('bcc-file')?.click()}
                    className={`flex cursor-pointer flex-col items-center gap-3 rounded-lg border-2 border-dashed px-5 py-7 text-center transition-all duration-200 sm:py-8
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
                        Kéo thả tệp <span className="text-emerald-700 dark:text-emerald-400">BCC</span> vào đây
                      </p>
                      <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                        hoặc nhấn để chọn tệp · định dạng .xlsx · tối đa 10 MB
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
                  <div className="flex items-center gap-3 rounded-lg border border-slate-300 bg-white p-3 shadow-sm animate-in fade-in slide-in-from-bottom-1 duration-200 dark:border-slate-700 dark:bg-slate-900">
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
                      title="Xóa tệp"
                      aria-label="Xóa tệp"
                      className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-md border border-slate-300 bg-white text-slate-500 transition-colors hover:border-rose-300 hover:bg-rose-50 hover:text-rose-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 dark:border-slate-600 dark:bg-slate-800 dark:hover:border-rose-700 dark:hover:bg-rose-900/30 dark:hover:text-rose-300"
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
              result.error_count > 0 ? (
                <ResultPartial result={result} />
              ) : (
                <ResultSuccess result={result} />
              )
            ) : result.status === 'pending' || result.status === 'processing' ? (
              <ImportProcessingState status={result.status} />
            ) : (
              <ResultFailure result={result} />
            ))}
        </div>

        {/* ── Footer (Tailkit modal footer pattern) ──────────────────────── */}
        <div className="flex shrink-0 items-center gap-2 border-t border-slate-200 bg-slate-50 px-4 py-3 sm:gap-3 sm:px-5 sm:py-4 dark:border-slate-700 dark:bg-slate-800/50">
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
            className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm leading-5 font-semibold text-slate-700 transition-colors hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-300 dark:hover:border-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-200"
          >
            {isImportActive ? 'Đóng cửa sổ' : result ? 'Đóng' : 'Hủy'}
          </button>

          {!result && (
            <button
              onClick={handleUpload}
              disabled={!canUpload || isPending}
              className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-emerald-700 bg-emerald-700 px-4 py-2 text-sm leading-5 font-semibold text-white transition-colors hover:border-emerald-600 hover:bg-emerald-600 disabled:cursor-not-allowed disabled:border-slate-300 disabled:bg-slate-300 disabled:text-white"
            >
              {isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang tải tệp lên…
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" />
                  Tải lên
                </>
              )}
            </button>
          )}

          {result && !isImportActive && (
            <button
              onClick={handleReset}
              className="inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-emerald-700 bg-emerald-700 px-4 py-2 text-sm leading-5 font-semibold text-white transition-colors hover:border-emerald-600 hover:bg-emerald-600"
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
