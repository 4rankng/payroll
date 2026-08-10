import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useUploadBCCTimesheet } from '@/hooks/timesheet/useUploadBCCTimesheet';
import { timesheetService } from '@/services/api/timesheet.service';
import { QueryKeys } from '@/lib/queryKeys';
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
	allowFlexibleEmployeeImport?: boolean;
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
  if (errors.length === 0) return 'Không thể xử lý tệp. Vui lòng kiểm tra lại định dạng.';

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
    return `Phát hiện ${duplicateErrors.length} bản chấm công trùng lặp. Vui lòng kiểm tra lại dữ liệu trong tệp.`;
  }

  return `Phát hiện ${errors.length} lỗi khi xử lý tệp. Vui lòng kiểm tra chi tiết bên dưới.`;
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

function invalidateTerminalImportQueries(
  queryClient: ReturnType<typeof useQueryClient>,
  result: PartnerImportFile,
) {
  if (result.status !== 'completed' && result.status !== 'failed') {
    return;
  }

  queryClient.invalidateQueries({ queryKey: ['partner-imports'] });
  queryClient.invalidateQueries({
    queryKey: QueryKeys.employees.missingBankDetails(),
  });

  if (result.status === 'completed') {
    queryClient.invalidateQueries({ queryKey: QueryKeys.timesheets.all });
  }
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useBCCUploadModal({
	projectId,
	projects,
	allowFlexibleEmployeeImport = false,
	onClose,
}: UseBCCUploadModalParams) {
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<PartnerImportFile | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string>(
    projectId > 0 ? String(projectId) : '',
  );
	const [isDragging, setIsDragging] = useState(false);
	const [includeFlexibleEmployees, setIncludeFlexibleEmployees] = useState(false);
  const idempotencyKeyRef = useRef<string | null>(null);
  const queryClient = useQueryClient();

  // Sync selectedProjectId when parent changes projectId (e.g. user switches project filter)
  const prevProjectIdRef = useRef(projectId);
  if (prevProjectIdRef.current !== projectId) {
    prevProjectIdRef.current = projectId;
    setSelectedProjectId(projectId > 0 ? String(projectId) : '');
  }

  // Default: current month as "YYYY-MM"
  const [selectedMonth, setSelectedMonth] = useState<string>(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  });

  // ── Derived values ──────────────────────────────────────────────────────

  const effectiveProjectId = useMemo(() => {
    const parsed = parseInt(selectedProjectId, 10);
    return parsed > 0 ? parsed : 0;
  }, [selectedProjectId]);

  // The page filter supplies a default. Users can still redirect the import to
  // any project that is available to their role.
  const needsProjectSelect = (projects?.length ?? 0) > 0;
  const hasProject = effectiveProjectId > 0;
  const isReady = hasProject && !!file;
  const canUpload = !!file && effectiveProjectId > 0;

  const hintText = useMemo(
    () => computeHintText(hasProject, !!file),
    [hasProject, file],
  );

  // ── Mutation ────────────────────────────────────────────────────────────

  const { mutate: upload, isPending } = useUploadBCCTimesheet();
  const isImportActive =
    result?.status === 'pending' || result?.status === 'processing';
  const { data: polledResult } = useQuery({
    queryKey: ['partner-import', result?.id],
    queryFn: () => timesheetService.getPartnerImport(result!.id),
    enabled: !!result?.id && isImportActive,
    refetchInterval: (query) => {
      const current = query.state.data;
      return current?.status === 'pending' || current?.status === 'processing'
        ? 2_000
        : false;
    },
    retry: 3,
  });

  useEffect(() => {
    if (!polledResult) return;
    setResult(polledResult);
    invalidateTerminalImportQueries(queryClient, polledResult);
  }, [polledResult, queryClient]);

  useEffect(() => {
    idempotencyKeyRef.current = null;
  }, [effectiveProjectId, selectedMonth]);

  // ── Handlers ────────────────────────────────────────────────────────────

  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setFile(e.target.files?.[0] ?? null);
      setResult(null);
      idempotencyKeyRef.current = null;
    },
    [],
  );

  const handleUpload = useCallback(() => {
    if (!file || effectiveProjectId === 0) return;
    setResult(null);
    const idempotencyKey =
      idempotencyKeyRef.current ?? crypto.randomUUID();
    idempotencyKeyRef.current = idempotencyKey;
    upload(
      {
        file,
        projectId: effectiveProjectId,
        forMonth: selectedMonth,
        includeFlexibleEmployees,
        idempotencyKey,
      },
      {
        onSuccess: (data) => {
          setResult(data);
          setFile(null);
          invalidateTerminalImportQueries(queryClient, data);
        },
        onError: () => {
          // Error already handled by useUploadBCCTimesheet
        },
      },
    );
  }, [file, effectiveProjectId, includeFlexibleEmployees, queryClient, selectedMonth, upload]);

	const handleIncludeFlexibleEmployeesChange = useCallback((checked: boolean) => {
		setIncludeFlexibleEmployees(checked);
		idempotencyKeyRef.current = null;
	}, []);

  const handleClose = useCallback(() => {
    setFile(null);
    setResult(null);
    onClose();
  }, [onClose]);

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
        idempotencyKeyRef.current = null;
      }
    },
    [isPending],
  );

  const handleReset = useCallback(() => {
    setResult(null);
    idempotencyKeyRef.current = null;
  }, []);

  const handleRemoveFile = useCallback(() => {
    setFile(null);
    idempotencyKeyRef.current = null;
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
    isImportActive,

    // Derived
    effectiveProjectId,
    needsProjectSelect,
    hasProject,
    canUpload,
    hintText,
		isReady,
		includeFlexibleEmployees,
		allowFlexibleEmployeeImport,

    // Setters
    setSelectedProjectId,
		setSelectedMonth,
		handleIncludeFlexibleEmployeesChange,

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
