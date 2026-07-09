import { useState, useMemo, memo, useRef, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Callout } from '@/components/ui/callout';
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
        className={`flex w-full items-center justify-between h-auto min-h-[46px] rounded-[10px] border-[1.5px] bg-[#fbfcfe] px-[15px] py-[13px] text-[14.5px] font-medium transition-all text-left
          ${hasProject
            ? 'border-[#bfe0cc] bg-white'
            : !value
              ? 'border-[#e4e8ef] text-[#7a8398]'
              : 'border-[#e4e8ef]'
          }
          ${open ? 'border-[#005A2D] shadow-[0_0_0_3px_rgba(0,90,45,.1)] bg-white' : ''}
        `}
      >
        <span className={selectedName ? 'text-[#16223a]' : 'text-[#7a8398]'}>
          {selectedName ?? 'Chọn dự án...'}
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-[#7a8398] transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute z-50 mt-1.5 w-full rounded-[10px] border border-[#e4e8ef] bg-white shadow-[0_8px_24px_-4px_rgba(14,32,56,.14)] overflow-hidden">
          {/* Search input */}
          <div className="flex items-center gap-2 border-b border-[#e4e8ef] px-3 py-2.5">
            <Search className="h-3.5 w-3.5 shrink-0 text-[#7a8398]" />
            <input
              ref={inputRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Tìm dự án..."
              className="flex-1 bg-transparent text-[13.5px] font-medium text-[#16223a] placeholder:text-[#aab2c0] outline-none"
            />
            {search && (
              <button onClick={() => setSearch('')} className="text-[#aab2c0] hover:text-[#475067]">
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-[220px] overflow-y-auto py-1">
            {filtered.length === 0 ? (
              <p className="px-4 py-3 text-[13px] text-[#7a8398]">Không tìm thấy dự án.</p>
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
                    className={`flex w-full items-center justify-between px-4 py-2.5 text-[13.5px] font-medium transition-colors text-left
                      ${isSelected
                        ? 'bg-[#e8f3ec] text-[#005A2D]'
                        : 'text-[#16223a] hover:bg-[#f5f7fa]'
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
    <div className="rounded-xl border border-emerald-200 bg-gradient-to-br from-emerald-50 to-emerald-100/50 p-5">
      <div className="flex items-start gap-3">
        <div className="rounded-full bg-emerald-600 p-1.5 mt-0.5 flex-shrink-0">
          <CheckCircle2 className="h-4 w-4 text-white" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-semibold text-emerald-900">Import thành công!</p>
          <p className="text-sm text-emerald-700 mt-0.5">
            Tháng {formatMonthLabel(result.for_month)} · {result.created_count} bản chấm công đã tạo
          </p>
        </div>
        <PartyPopper className="h-5 w-5 text-emerald-600 flex-shrink-0 mt-0.5" />
      </div>

      {(result.created_count > 0 || result.skipped_count > 0) && (
        <div className="flex flex-wrap gap-2 mt-4">
          {result.created_count > 0 && (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-3 py-1 text-xs font-bold text-emerald-700">
              <CheckCircle2 className="h-3 w-3" />
              {result.created_count} tạo mới
            </span>
          )}
          {result.skipped_count > 0 && (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-bold text-amber-700">
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
      <div className="rounded-xl border border-red-200 bg-gradient-to-br from-red-50 to-red-100/40 p-5">
        <div className="flex items-start gap-3">
          <div className="rounded-full bg-red-600 p-1.5 mt-0.5 flex-shrink-0">
            <XCircle className="h-4 w-4 text-white" />
          </div>
          <div className="min-w-0 flex-1">
            <p className="font-semibold text-red-900">Import thất bại</p>
            <p className="text-sm text-red-700 mt-1 leading-relaxed">{summaryText}</p>
          </div>
        </div>
      </div>

      {(result.created_count > 0 || result.skipped_count > 0 || result.error_count > 0) && (
        <div className="flex flex-wrap gap-2">
          {result.created_count > 0 && (
            <div className="flex items-center gap-2 rounded-lg bg-emerald-50 border border-emerald-200 px-3.5 py-2">
              <CheckCircle2 className="h-4 w-4 text-emerald-600 shrink-0" />
              <span className="font-bold text-emerald-800">{result.created_count}</span>
              <span className="text-emerald-700 text-sm">tạo mới</span>
            </div>
          )}
          {result.skipped_count > 0 && (
            <div className="flex items-center gap-2 rounded-lg bg-amber-50 border border-amber-200 px-3.5 py-2">
              <SkipForward className="h-4 w-4 text-amber-600 shrink-0" />
              <span className="font-bold text-amber-800">{result.skipped_count}</span>
              <span className="text-amber-700 text-sm">bỏ qua</span>
            </div>
          )}
          {result.error_count > 0 && (
            <div className="flex items-center gap-2 rounded-lg bg-red-50 border border-red-200 px-3.5 py-2">
              <AlertTriangle className="h-4 w-4 text-red-600 shrink-0" />
              <span className="font-bold text-red-800">{result.error_count}</span>
              <span className="text-red-700 text-sm">lỗi</span>
            </div>
          )}
        </div>
      )}

      {parsedErrors.length > 0 && (
        <div className="space-y-2">
          <button
            onClick={() => setShowErrors((v) => !v)}
            className="group flex items-center gap-1.5 text-sm font-medium text-red-700 hover:text-red-900 transition-colors"
          >
            <span>{showErrors ? 'Ẩn chi tiết lỗi' : `Xem ${parsedErrors.length} lỗi chi tiết`}</span>
            <ChevronDown
              className={`h-4 w-4 transition-transform duration-200 ${showErrors ? 'rotate-180' : ''}`}
            />
          </button>

          {showErrors && (
            <div className="max-h-52 overflow-y-auto rounded-lg border border-red-200 bg-white divide-y divide-red-100">
              {Array.from(groupedErrors.entries()).map(([employee, errors]) => (
                <div key={employee} className="px-4 py-3">
                  <p className="text-sm font-semibold text-foreground mb-1">{employee}</p>
                  <ul className="space-y-1">
                    {errors.map((e, i) => (
                      <li
                        key={i}
                        className="text-sm text-red-700 leading-relaxed pl-3 border-l-2 border-red-200"
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
        className="sm:max-w-[548px] gap-0 overflow-visible rounded-[14px]"
        contentPadding="none"
        hideCloseButton
      >
        {/* ── Navy Header ─────────────────────────────────────────────────── */}
        <div className="relative flex items-center gap-3.5 px-6 py-5 bg-gradient-to-br from-[#0e2038] to-[#15294a]">
          <div className="flex h-[38px] w-[38px] shrink-0 items-center justify-center rounded-[10px] border border-white/[.14] bg-white/[.08]">
            <FileText className="h-5 w-5 stroke-white" />
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="text-[17px] font-bold tracking-tight text-white leading-tight">
              Tải lên Bảng Chấm Công
            </h2>
            <p className="mt-0.5 text-[12.5px] font-medium text-[#a9bdd9]">
              Nhập dữ liệu chấm công từ file Excel
            </p>
          </div>
          <button
            onClick={handleClose}
            className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-lg border-none bg-white/[.08] text-[#cdd8e8] transition-colors hover:bg-white/[.16] hover:text-white"
            title="Đóng"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* ── Body ────────────────────────────────────────────────────────── */}
        <div className="px-6 pt-[22px] pb-1">
          {!result && (
            <div className="space-y-4">
              {/* Project select */}
              {needsProjectSelect && (
                <div>
                  <div className="mb-[7px] flex items-center gap-1.5 text-[12.5px] font-semibold text-[#475067]">
                    <span>Chọn dự án</span>
                    <span className="text-[#005A2D]">*</span>
                    {hasProject && (
                      <span className="ml-auto inline-flex items-center gap-1 text-[11.5px] font-semibold text-[#005A2D]">
                        <Check className="h-[13px] w-[13px]" strokeWidth={3} />
                        Đã chọn
                      </span>
                    )}
                  </div>
                  <ProjectCombobox
                    projects={projects ?? EMPTY_PROJECTS}
                    value={selectedProjectId}
                    onChange={setSelectedProjectId}
                    hasProject={hasProject}
                  />
                </div>
              )}

              {/* Month select */}
              <div>
                <div className="mb-[7px] flex items-center gap-1.5 text-[12.5px] font-semibold text-[#475067]">
                  <span>Tháng áp dụng</span>
                  <span className="text-[#005A2D]">*</span>
                  <span className="ml-auto inline-flex items-center gap-1 text-[11.5px] font-semibold text-[#005A2D]">
                    <Check className="h-[13px] w-[13px]" strokeWidth={3} />
                    Đã chọn
                  </span>
                </div>
                <Select value={selectedMonth} onValueChange={setSelectedMonth}>
                  <SelectTrigger className="h-auto min-h-[46px] rounded-[10px] border-[1.5px] border-[#bfe0cc] bg-white px-[15px] py-[13px] text-[14.5px] font-medium text-[#16223a] transition-all focus:ring-0 focus:ring-offset-0 focus:border-[#005A2D] focus:shadow-[0_0_0_3px_rgba(0,90,45,.1)]">
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

              {/* Info callout */}
              <Callout variant="info">
                Các ngày trong file Excel sẽ được gán vào tháng đã chọn — ví dụ ngày{' '}
                <code className="rounded-[4px] bg-[rgba(31,78,121,.1)] px-[5px] py-px font-mono text-[11.5px]">
                  22
                </code>{' '}
                sẽ thành{' '}
                <code className="rounded-[4px] bg-[rgba(31,78,121,.1)] px-[5px] py-px font-mono text-[11.5px]">
                  22/{selectedMonth.split('-')[1]}/{selectedMonth.split('-')[0]}
                </code>
                .
              </Callout>

              {/* Upload area */}
              <div>
                <div className="mb-2 flex items-center gap-1.5 text-[12.5px] font-semibold text-[#475067]">
                  <span>Tệp bảng chấm công</span>
                  <span className="text-[#005A2D]">*</span>
                </div>

                {/* Dropzone */}
                {!file && (
                  <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    onClick={() => !isPending && document.getElementById('bcc-file')?.click()}
                    className={`
                      flex flex-col items-center gap-2.5 rounded-xl border-2 border-dashed
                      bg-[#fafbfd] px-5 py-[26px] text-center cursor-pointer transition-all duration-200
                      ${isDragging
                        ? 'border-[#005A2D] bg-[#e8f3ec] scale-[1.005]'
                        : 'border-[#e4e8ef] hover:border-[#b7d8c4] hover:bg-[#e8f3ec]'
                      }
                    `}
                  >
                    <div className="flex h-12 w-12 items-center justify-center rounded-[11px] border border-[#e4e8ef] bg-white shadow-[0_2px_8px_-3px_rgba(14,32,56,.12)]">
                      <Upload className="h-[22px] w-[22px] stroke-[#005A2D]" />
                    </div>
                    <p className="text-[14px] font-bold text-[#16223a]">
                      Kéo thả file <span className="text-[#005A2D]">BCC</span> vào đây
                    </p>
                    <p className="text-[12px] font-medium text-[#7a8398]">
                      hoặc nhấn để chọn file · định dạng .xlsx · tối đa 10 MB
                    </p>
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
                  <div className="flex items-center gap-3 rounded-xl border-[1.5px] border-[#bfe0cc] bg-white p-3.5 animate-in fade-in slide-in-from-bottom-1 duration-200">
                    <div className="flex h-[42px] w-[42px] shrink-0 items-center justify-center rounded-[9px] bg-[#e8f3ec]">
                      <FileSpreadsheet className="h-[21px] w-[21px] stroke-[#005A2D]" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[13.5px] font-bold text-[#16223a]">
                        {file.name}
                      </p>
                      <div className="mt-0.5 flex items-center gap-2">
                        <span className="font-mono text-[11px] font-medium text-[#7a8398]">
                          {(file.size / 1024).toFixed(0)} KB
                        </span>
                      </div>
                    </div>
                    <button
                      onClick={handleRemoveFile}
                      title="Xóa file"
                      className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-lg border border-[#e4e8ef] bg-white text-[#7a8398] transition-colors hover:border-[#f3c8c2] hover:bg-[#fdecea] hover:text-[#c0392b]"
                    >
                      <X className="h-[15px] w-[15px]" />
                    </button>
                  </div>
                )}

                {/* Progress bar */}
                {isPending && (
                  <div className="mt-2.5 animate-in fade-in duration-200">
                    <div className="h-1.5 overflow-hidden rounded-full bg-[#eef1f6]">
                      <div
                        className="h-full animate-pulse rounded-full bg-gradient-to-r from-[#005A2D] to-[#0a7a40]"
                        style={{ width: '60%' }}
                      />
                    </div>
                    <p className="mt-1.5 text-right text-[11.5px] font-semibold text-[#475067]">
                      <Loader2 className="mr-1 inline h-3 w-3 animate-spin" />
                      Đang tải lên…
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Result states */}
          {result &&
            (result.status === 'completed' ? (
              <ResultSuccess result={result} />
            ) : (
              <ResultFailure result={result} />
            ))}
        </div>

        {/* ── Footer ──────────────────────────────────────────────────────── */}
        <div className="flex items-center gap-3 border-t border-[#eef1f6] px-6 pb-5 pt-4 mt-3">
          {!result && (
            <div
              className={`flex items-center gap-1.5 flex-1 min-w-0 ${isReady ? 'text-[#005A2D]' : 'text-[#7a8398]'}`}
            >
              {isReady ? (
                <Check className="h-3.5 w-3.5 shrink-0" strokeWidth={2.5} />
              ) : (
                <svg className="h-3.5 w-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
              )}
              <span className="truncate text-[12px] font-medium">{hintText}</span>
            </div>
          )}
          {result && <div className="flex-1" />}

          <button
            onClick={handleClose}
            disabled={isPending}
            className="inline-flex items-center gap-2 rounded-[10px] border-[1.5px] border-[#e4e8ef] bg-white px-[18px] py-[11px] text-[14px] font-semibold text-[#475067] transition-colors hover:border-[#cfd6e2] hover:bg-[#fafbfd] disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {result ? 'Đóng' : 'Hủy'}
          </button>

          {!result && (
            <button
              onClick={handleUpload}
              disabled={!canUpload || isPending}
              className="inline-flex items-center gap-2 rounded-[10px] border-[1.5px] border-[#005A2D] bg-[#005A2D] px-[18px] py-[11px] text-[14px] font-semibold text-white transition-colors hover:border-[#00481f] hover:bg-[#00481f] disabled:border-[#cdd4de] disabled:bg-[#cdd4de] disabled:text-white disabled:cursor-not-allowed disabled:opacity-85"
            >
              {isPending ? (
                <>
                  <Loader2 className="h-[17px] w-[17px] animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <Upload className="h-[17px] w-[17px] stroke-white" />
                  Tải lên
                </>
              )}
            </button>
          )}

          {result && (
            <button
              onClick={handleReset}
              className="inline-flex items-center gap-2 rounded-[10px] border-[1.5px] border-[#005A2D] bg-[#005A2D] px-[18px] py-[11px] text-[14px] font-semibold text-white transition-colors hover:border-[#00481f] hover:bg-[#00481f]"
            >
              <Upload className="h-[17px] w-[17px] stroke-white" />
              Upload khác
            </button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
});
