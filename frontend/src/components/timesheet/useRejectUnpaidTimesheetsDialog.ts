import { useCallback, useEffect, useMemo, useState } from 'react';
import { useAllProjects } from '@/hooks/api/useProjects';
import { useRejectUnpaidTimesheets } from '@/hooks/api/useTimesheets';
import type { RejectUnpaidTimesheetsData } from '@/types/api/timesheet.types';

type DialogStep = 'form' | 'confirm';

interface FormErrors {
  project?: string;
  dateRange?: string;
  reason?: string;
}

interface UseRejectUnpaidTimesheetsDialogOptions {
  open: boolean;
  initialProjectId?: number;
  onSuccess: () => void;
}

function getDefaultDateRange() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return {
    fromDate: `${year}-${month}-01`,
    toDate: `${year}-${month}-${day}`,
  };
}

export function validateRejectUnpaidTimesheetsForm(
  projectId: string,
  fromDate: string,
  toDate: string,
  rejectionReason: string,
): FormErrors {
  const errors: FormErrors = {};
  if (!projectId) errors.project = 'Vui lòng chọn một dự án.';
  if (!fromDate || !toDate) {
    errors.dateRange = 'Vui lòng chọn đầy đủ khoảng ngày.';
  } else if (fromDate > toDate) {
    errors.dateRange = 'Ngày bắt đầu không được sau ngày kết thúc.';
  }
  if (!rejectionReason.trim()) errors.reason = 'Vui lòng nhập lý do loại.';
  return errors;
}

export function useRejectUnpaidTimesheetsDialog({
  open,
  initialProjectId,
  onSuccess,
}: UseRejectUnpaidTimesheetsDialogOptions) {
  const defaultDates = useMemo(getDefaultDateRange, []);
  const [projectId, setProjectId] = useState('');
  const [fromDate, setFromDate] = useState(defaultDates.fromDate);
  const [toDate, setToDate] = useState(defaultDates.toDate);
  const [rejectionReason, setRejectionReason] = useState('');
  const [errors, setErrors] = useState<FormErrors>({});
  const [step, setStep] = useState<DialogStep>('form');
  const { data: projects = [], isLoading: isProjectsLoading } = useAllProjects({ enabled: open });
  const mutation = useRejectUnpaidTimesheets();

  const projectOptions = useMemo(
    () => projects.map((project) => ({
      value: String(project.id),
      label: project.name,
      subtitle: project.code,
    })),
    [projects],
  );
  const selectedProject = useMemo(
    () => projects.find((project) => String(project.id) === projectId),
    [projectId, projects],
  );

  useEffect(() => {
    if (!open) return;
    setProjectId(initialProjectId ? String(initialProjectId) : '');
    setFromDate(defaultDates.fromDate);
    setToDate(defaultDates.toDate);
    setRejectionReason('');
    setErrors({});
    setStep('form');
  }, [defaultDates.fromDate, defaultDates.toDate, initialProjectId, open]);

  const handleReview = useCallback(() => {
    const nextErrors = validateRejectUnpaidTimesheetsForm(
      projectId,
      fromDate,
      toDate,
      rejectionReason,
    );
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length === 0) setStep('confirm');
  }, [fromDate, projectId, rejectionReason, toDate]);

  const handleSubmit = useCallback(async () => {
    const payload: RejectUnpaidTimesheetsData = {
      project_id: Number(projectId),
      from_date: fromDate,
      to_date: toDate,
      rejection_reason: rejectionReason.trim(),
    };
    try {
      await mutation.mutateAsync(payload);
      onSuccess();
    } catch {
      // The global mutation error handler shows the server message. Keep the
      // confirmation step and all inputs intact so Admin can correct or retry.
    }
  }, [fromDate, mutation, onSuccess, projectId, rejectionReason, toDate]);

  return {
    projectId,
    setProjectId,
    fromDate,
    setFromDate,
    toDate,
    setToDate,
    rejectionReason,
    setRejectionReason,
    errors,
    step,
    setStep,
    projectOptions,
    selectedProject,
    isProjectsLoading,
    isPending: mutation.isPending,
    handleReview,
    handleSubmit,
  };
}
