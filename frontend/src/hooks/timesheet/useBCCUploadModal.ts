import { useState, useCallback, useMemo } from 'react';
import { useUploadBCCTimesheet } from '@/hooks/timesheet/useUploadBCCTimesheet';
import { parseImportErrors } from '@/utils/import-errors';
import type { PartnerImportFile, ImportError } from '@/types/api/timesheet.types';

// ─── Types ────────────────────────────────────────────────────────────────────

interface ProjectOption {
  id: number;
  name: string;
}

interface UseBCCUploadModalParams {
  projectId: number;
  projects?: ProjectOption[];
  onClose: () => void;
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Format "2026-05" → "05/2026" */
export function formatMonthLabel(month: string): string {
  const parts = month.split('-');
  if (parts.length === 2) return `${parts[1]}/${parts[0]}`;
  return month;
}

/** Derive a human-readable failure explanation from error list. */
export function buildFailureSummary(
  errors: ImportError[],
  forMonth?: string,
): string {
  if (errors.length === 0) return 'File không thể xử lý. Vui lòng kiểm tra lại định dạng.';

  const approvedErrors = errors.filter(
    (e) =>
      e.reason.toLowerCase().includes('phê duyệt') ||
      e.reason.toLowerCase().includes('approved') ||
      e.reason.toLowerCase().includes('đã duyệt'),
  );

  if (approvedErrors.length > 0 && forMonth) {
    const monthLabel = formatMonthLabel(forMonth);
    return `Phát hiện ${approvedErrors.length} bản chấm công đã phê duyệt trong tháng ${monthLabel}. Vui lòng chọn tháng chưa phê duyệt hoặc liên hệ quản trị viên.`;
  }

  const duplicateErrors = errors.filter(
    (e) =>
      e.reason.toLowerCase().includes('tồn tại') ||
      e.reason.toLowerCase().includes('duplicate') ||
      e.reason.toLowerCase().includes('đã có'),
  );

  if (duplicateErrors.length > 0) {
    return `Phát hiện ${duplicateErrors.length} bản chấm công trùng lặp. Vui lòng kiểm tra lại dữ liệu trong file.`;
  }

  return `Phát hiện ${errors.length} lỗi khi xử lý file. Vui lòng kiểm tra chi tiết bên dưới.`;
}

/** Group errors by employee name for display. */
export function groupErrorsByEmployee(
  errors: ImportError[],
): Map<string, ImportError[]> {
  const map = new Map<string, ImportError[]>();
  for (const e of errors) {
    const key = e.employee || 'Không rõ nhân viên';
    const list = map.get(key) || [];
    list.push(e);
    map.set(key, list);
  }
  return map;
}

/** Parse error detail string from import result. */
export function parseResultErrors(
  errorDetail?: string | null,
): ImportError[] {
  return parseImportErrors(errorDetail);
}

/** Generate month options (current + 11 previous) as value/label pairs. */
export function getMonthOptions(): Array<{
  value: string;
  label: string;
  isCurrent: boolean;
}> {
  return Array.from({ length: 12 }, (_, i) => {
    const now = new Date();
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    const label = `Tháng ${String(d.getMonth() + 1).padStart(2, '0')}/${d.getFullYear()}`;
    return { value, label, isCurrent: i === 0 };
  });
}

/** Compute dynamic hint text based on current form state. */
export function computeHintText(hasProject: boolean, hasFile: boolean): string {
  if (hasProject && hasFile) return 'Sẵn sàng tải lên';
  if (!hasProject && !hasFile) return 'Cần chọn dự án và tệp để tiếp tục';
  if (!hasProject) return 'Còn thiếu: dự án';
  return 'Còn thiếu: tệp bảng chấm công';
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useBCCUploadModal({
  projectId,
  projects,
  onClose,
}: UseBCCUploadModalParams) {
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<PartnerImportFile | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string>(
    projectId > 0 ? String(projectId) : '',
  );
  const [isDragging, setIsDragging] = useState(false);

  // Default: current month as "YYYY-MM"
  const [selectedMonth, setSelectedMonth] = useState<string>(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  });

  // ── Derived values ──────────────────────────────────────────────────────

  const effectiveProjectId = useMemo(() => {
    if (projectId > 0) return projectId;
    const parsed = parseInt(selectedProjectId, 10);
    return parsed > 0 ? parsed : 0;
  }, [projectId, selectedProjectId]);

  const needsProjectSelect = projectId === 0 && (projects?.length ?? 0) > 0;
  const hasProject = effectiveProjectId > 0;
  const isReady = hasProject && !!file;
  const canUpload = !!file && effectiveProjectId > 0;

  const hintText = useMemo(
    () => computeHintText(hasProject, !!file),
    [hasProject, file],
  );

  // ── Mutation ────────────────────────────────────────────────────────────

  const { mutate: upload, isPending } = useUploadBCCTimesheet();

  // ── Handlers ────────────────────────────────────────────────────────────

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFile(e.target.files?.[0] ?? null);
      setResult(null);
    },
    [],
  );

  const handleUpload = useCallback(() => {
    if (!file || effectiveProjectId === 0) return;
    setResult(null);
    upload(
      { file, projectId: effectiveProjectId, forMonth: selectedMonth },
      {
        onSuccess: (data) => {
          setResult(data);
          setFile(null);
        },
        onError: () => {
          // Error already handled by useUploadBCCTimesheet
        },
      },
    );
  }, [file, effectiveProjectId, selectedMonth, upload]);

  const handleClose = useCallback(() => {
    if (isPending) return;
    setFile(null);
    setResult(null);
    onClose();
  }, [isPending, onClose]);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragging(false);
      if (isPending) return;
      const dropped = e.dataTransfer.files?.[0];
      if (dropped && dropped.name.endsWith('.xlsx')) {
        setFile(dropped);
        setResult(null);
      }
    },
    [isPending],
  );

  const handleReset = useCallback(() => {
    setResult(null);
  }, []);

  const handleRemoveFile = useCallback(() => {
    setFile(null);
  }, []);

  // ── Return ──────────────────────────────────────────────────────────────

  return {
    // State
    file,
    result,
    selectedProjectId,
    selectedMonth,
    isDragging,
    isPending,

    // Derived
    effectiveProjectId,
    needsProjectSelect,
    hasProject,
    canUpload,
    hintText,
    isReady,

    // Setters
    setSelectedProjectId,
    setSelectedMonth,

    // Handlers
    handleFileChange,
    handleUpload,
    handleClose,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleReset,
    handleRemoveFile,
  };
}
