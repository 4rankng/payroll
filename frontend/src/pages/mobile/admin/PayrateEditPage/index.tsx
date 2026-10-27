import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
// UU PRO restyle (W13c): lucide → verified @untitledui/icons; error/warning
// accents on the UU utility ladders. Validation flow and copy byte-preserved.
import {
  CalendarDate,
  CornerDownRight,
  Lock01,
  Save01,
  Zap,
} from '@untitledui/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import { MobileSubPageHeader } from '@/components/shared/MobileSubPageHeader';
import { PayratePeriodStrip } from '@/components/payrates/components/PayratePeriodStrip';
import { PayrateMatrixEditor } from '@/components/payrates/components/PayrateMatrixEditor';
import { FlexiblePayrateEditor } from '@/components/payrates/components/FlexiblePayrateEditor';
import {
  type PayrateConfig,
  type PayrateStructure,
  type EditorMode,
  validatePayrateStructure,
  validateFlexiblePayrateStructure,
  createDefaultPayrateStructure,
  createFlexiblePayrateStructure,
  backfillCarriedForwardMatrix,
} from '@/components/payrates/types';
import { dateToString } from '@/utils/dateHelpers';
import { useProject } from '@/hooks/api/useProjects';
import { authManager } from '@/lib/auth';
import {
  useProjectPayRates,
  useCreateProjectPayRate,
  useUpdatePayRate,
} from '@/hooks/api/usePayRates';
import { payRateService, type DryRunValidateResponse, type FieldResult } from '@/services/api/payrate.service';
import type { PayRate } from '@/types/api/payrate.types';
import { cn } from '@/lib/utils';

type SaveStep = 'idle' | 'validating' | 'confirm' | 'saving';

/** Per-field feedback rendered as a stacked card with optional apply-suggestion. */
function FieldFeedback({ field, onApplySuggestion }: {
  field: FieldResult | undefined;
  onApplySuggestion?: (value: string) => void;
}) {
  if (!field || field.status === 'ok' || !field.message) return null;
  const isError = field.status === 'error';
  return (
    <div className={cn(
      'mt-1.5 rounded-xl px-3 py-2 text-xs',
      isError
        ? 'border border-utility-error-200 bg-utility-error-50 text-fg-error-primary'
        : 'border border-utility-warning-200 bg-utility-warning-50 text-fg-warning-primary',
    )}>
      <p className="font-medium">{field.message}</p>
      {field.hint && <p className="mt-0.5 opacity-80">{field.hint}</p>}
      {field.suggested_value && onApplySuggestion && (
          <button
            type="button"
            onClick={() => onApplySuggestion(field.suggested_value!)}
            className={cn(
              'mt-1.5 flex min-h-11 items-center gap-1 text-xs font-semibold underline underline-offset-2',
              isError ? 'hover:text-utility-error-700' : 'hover:text-utility-warning-700',
            )}
        >
          <CornerDownRight className="h-3 w-3" />
          Dùng giá trị: {field.suggested_value}
        </button>
      )}
    </div>
  );
}

function fieldBorderClass(field: FieldResult | undefined, highlighted = false): string {
  if (!field || field.status === 'ok') {
    return highlighted ? 'border-utility-warning-400 ring-2 ring-utility-warning-200 bg-utility-warning-50/40' : '';
  }
  if (field.status === 'error') return 'border-utility-error-400 ring-1 ring-utility-error-200 bg-utility-error-50/30';
  if (field.status === 'warning') return 'border-utility-warning-400 ring-1 ring-utility-warning-200 bg-utility-warning-50/30';
  return '';
}

export default function PayrateEditPageMobile() {
  const { projectId, payrateId } = useParams<{ projectId: string; payrateId: string }>();
  const navigate = useNavigate();

  const numProjectId = Number(projectId);
  const isNew = payrateId === 'new';
  const numPayrateId = isNew ? null : Number(payrateId);

  const { data: projectData, isLoading: isLoadingProject } = useProject(numProjectId);
  const { data: allPayRatesData, isLoading: isLoadingRates } = useProjectPayRates(numProjectId);
  const createMutation = useCreateProjectPayRate();
  const updateMutation = useUpdatePayRate();

  const project = projectData;

  const targetPayrate = useMemo<PayRate | null>(() => {
    if (isNew || !numPayrateId) return null;
    return allPayRatesData?.data?.find(p => p.id === numPayrateId) ?? null;
  }, [allPayRatesData, isNew, numPayrateId]);

  // Latest config = first row of the from_date DESC list. New versions must
  // start strictly after it, and create pre-fills from it (carry-forward).
  const latestPayrate = useMemo<PayRate | null>(() => {
    return allPayRatesData?.data?.[0] ?? null;
  }, [allPayRatesData]);

  const editorMode: EditorMode = isNew ? 'create' : 'edit';

  const [config, setConfig] = useState<Partial<PayrateConfig>>({
    project_id: numProjectId,
    rates: createDefaultPayrateStructure(),
    fromDate: dateToString(new Date()),
  });

  const [saveStep, setSaveStep] = useState<SaveStep>('idle');
  const [serverResult, setServerResult] = useState<DryRunValidateResponse | null>(null);
  const [validateError, setValidateError] = useState<string | null>(null);
  const [hasAttemptedSave, setHasAttemptedSave] = useState(false);
  // Whether the pending (validated) save changes rates — decides create vs
  // in-place update when the user confirms.
  const [pendingRatesChanged, setPendingRatesChanged] = useState(false);

  const fromDateRef = useRef<HTMLInputElement>(null);

  const isFlexible = !!project?.is_flexible;

  // Carry-forward: a new payrate starts as a copy of the latest config, so
  // untouched positions keep their rates. First-ever config keeps the generic
  // template. The ref prevents re-running on subsequent renders.
  const carryForwardDone = useRef(false);
  useEffect(() => {
    if (!isNew || carryForwardDone.current || isLoadingProject || isLoadingRates) return;
    carryForwardDone.current = true;
    if (latestPayrate) {
      const next = new Date(latestPayrate.fromDate + 'T00:00:00');
      next.setDate(next.getDate() + 1);
      const cloned = JSON.parse(JSON.stringify(latestPayrate.rates)) as PayrateStructure;
      setConfig(prev => ({
        ...prev,
        // Non-flexible carry-forward backfills the full grid (zeros explicit)
        // so the strict client validator accepts the seeded matrix.
        rates: isFlexible ? cloned : backfillCarriedForwardMatrix(cloned),
        fromDate: dateToString(next),
      }));
    } else if (isFlexible) {
      setConfig(prev => ({ ...prev, rates: createFlexiblePayrateStructure() }));
    }
  }, [isNew, isLoadingProject, isLoadingRates, isFlexible, latestPayrate]);

  // Create mode appends after the latest config: the date input floors at
  // latest from_date + 1 (the backend rejects anything on or before it).
  const minFromDate = useMemo(() => {
    if (editorMode === 'edit') {
      return serverResult?.fields.effective_from.min_value || targetPayrate?.earliest_effective_from;
    }
    if (!latestPayrate) return undefined;
    const d = new Date(latestPayrate.fromDate + 'T00:00:00');
    d.setDate(d.getDate() + 1);
    return dateToString(d);
  }, [editorMode, serverResult, targetPayrate, latestPayrate]);

  useEffect(() => {
    if (targetPayrate) {
      // Default the start date to the paid floor (day after the latest paid
      // timesheet work date) whenever the stored start sits before it — the backend
      // rejects anything earlier, so pre-clamp instead of surfacing an error.
      // Superseded configs are exempt: their stored start legitimately predates
      // the floor, and pre-clamping would point at the current config's date.
      const isSupersededTarget = latestPayrate != null && targetPayrate.id !== latestPayrate.id;
      const floor = targetPayrate.earliest_effective_from;
      const fromDate =
        floor && !isSupersededTarget && targetPayrate.fromDate < floor ? floor : targetPayrate.fromDate;
      setConfig({
        project_id: numProjectId,
        rates: targetPayrate.rates || {},
        fromDate,
      });
    }
  }, [targetPayrate, numProjectId, latestPayrate]);

  const clearServerField = useCallback((field: 'effective_from' | 'rates') => {
    setValidateError(null);
    if (!serverResult) return;
    setServerResult(prev => {
      if (!prev) return prev;
      return {
        ...prev,
        fields: {
          ...prev.fields,
          [field]: { ...prev.fields[field], status: 'ok', message: undefined, hint: undefined },
        },
      };
    });
  }, [serverResult]);

  const clientValidation = useMemo(() => {
    if (!config.rates || Object.keys(config.rates).length === 0) {
      return { valid: false, errors: ['Cần có ít nhất một cấu hình mức lương'] };
    }
    return isFlexible
      ? validateFlexiblePayrateStructure(config.rates)
      : validatePayrateStructure(config.rates);
  }, [config.rates, isFlexible]);

  // Diff summary for the confirmation step: what changes relative to the
  // baseline this save versioned from (target for edits, latest for creates).
  const diffSummary = useMemo(() => {
    const base = (editorMode === 'edit' ? targetPayrate?.rates : latestPayrate?.rates) ?? {};
    const next = config.rates ?? {};
    let changedCells = 0;
    const addedPositions: string[] = [];
    const removedPositions: string[] = [];

    const dayKeys = (m: unknown): string[] => (m && typeof m === 'object' ? Object.keys(m) : []);
    const rateAt = (m: unknown, day: string, shift: string): number | undefined => {
      const dayMap = (m as Record<string, Record<string, number>>)[day];
      return dayMap?.[shift];
    };

    for (const pos of Object.keys(base)) {
      if (!(pos in next)) {
        removedPositions.push(pos);
        continue;
      }
      for (const day of dayKeys(base[pos])) {
        for (const shift of dayKeys((base[pos] as Record<string, unknown>)[day])) {
          if (rateAt(base[pos], day, shift) !== rateAt(next[pos], day, shift)) changedCells += 1;
        }
      }
    }
    for (const pos of Object.keys(next)) {
      if (!(pos in base)) addedPositions.push(pos);
    }
    return { changedCells, addedPositions, removedPositions };
  }, [editorMode, targetPayrate, latestPayrate, config.rates]);

  // Superseded configs (a newer config took over) are history: rows under them
  // are priced, and the backend rejects any update. Render everything
  // read-only instead of offering an edit that can only fail.
  const isEnded = editorMode === 'edit' && !!targetPayrate && !!latestPayrate && targetPayrate.id !== latestPayrate.id;
  const ratesLocked = isEnded || (serverResult?.fields.rates.locked ?? false);
  const fromDateLocked = isEnded || (serverResult?.fields.effective_from.locked ?? false);

  const handleRatesChange = useCallback((rates: PayrateStructure) => {
    if (ratesLocked) return;
    setConfig(prev => ({ ...prev, rates }));
    clearServerField('rates');
  }, [ratesLocked, clearServerField]);

  const basePath = authManager.getUserRole() === 'partner' ? '/partner' : '/admin';
  const goBack = useCallback(() => {
    navigate(`${basePath}/projects?modal=project_details&id=${numProjectId}`);
  }, [navigate, basePath, numProjectId]);

  // From-date-only model: rate changes always create a NEW config version;
  // only a start-date correction without rate changes updates in place.
  const performSave = useCallback(async (ratesChanged: boolean) => {
    setSaveStep('saving');
    try {
      if (editorMode === 'edit' && targetPayrate?.id && !ratesChanged) {
        await updateMutation.mutateAsync({
          payRateId: targetPayrate.id,
          data: { rates: config.rates!, effective_from: config.fromDate! },
        });
      } else {
        await createMutation.mutateAsync({
          projectId: numProjectId,
          data: { rates: config.rates!, effective_from: config.fromDate! },
        });
      }
      goBack();
    } catch (err: unknown) {
      // Never swallow failed saves: a silent reset invites retries that
      // duplicate rows.
      const apiErr = err as { response?: { data?: { message?: string } }; message?: string };
      setValidateError(apiErr?.response?.data?.message || apiErr?.message || 'Lưu thất bại. Vui lòng thử lại.');
      setServerResult(null);
      setSaveStep('idle');
    }
  }, [config, numProjectId, editorMode, targetPayrate, createMutation, updateMutation, goBack]);

  const handleValidate = useCallback(async () => {
    setHasAttemptedSave(true);
    if (!clientValidation.valid || !config.fromDate) return;
    setSaveStep('validating');
    setServerResult(null);
    setValidateError(null);
    try {
      const ratesChanged =
        editorMode === 'edit' && targetPayrate &&
        JSON.stringify(targetPayrate.rates ?? {}) !== JSON.stringify(config.rates ?? {});
      const result = await payRateService.dryRunValidate(
        numProjectId,
        { rates: config.rates, effective_from: config.fromDate },
        ratesChanged ? undefined : numPayrateId ?? undefined,
      );
      setServerResult(result);
      setPendingRatesChanged(ratesChanged);

      if (result.valid) {
        if (editorMode === 'edit' && targetPayrate?.id && !ratesChanged) {
          await performSave(ratesChanged);
        } else {
          setSaveStep('confirm');
        }
      } else {
        setSaveStep('idle');
        setTimeout(() => {
          if (result.fields.effective_from.status === 'error') fromDateRef.current?.focus();
        }, 50);
      }
    } catch (err: unknown) {
      const apiErr = err as { message?: string };
      setValidateError(apiErr?.message || 'Không thể kiểm tra. Vui lòng thử lại.');
      setServerResult(null);
      setSaveStep('idle');
    }
  }, [clientValidation.valid, config, numProjectId, numPayrateId, editorMode, targetPayrate, performSave]);

  const handleConfirmSave = useCallback(async () => {
    await performSave(pendingRatesChanged);
  }, [performSave, pendingRatesChanged]);

  const isLoading = isLoadingProject || isLoadingRates;
  const isValidating = saveStep === 'validating';
  const isSaving = saveStep === 'saving';
  const canProceed = clientValidation.valid && !!config.fromDate && !isLoading;

  const sf = serverResult?.fields;

  return (
    <MobilePageShell>
      <MobileSubPageHeader
        title={isNew ? 'Tạo cấu hình lương' : 'Sửa cấu hình lương'}
        subtitle={isLoadingProject ? undefined : project?.name}
        icon={Zap}
        onBack={goBack}
      />

      <div className="space-y-4 py-4">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-20 rounded-xl" />
            <Skeleton className="h-64 rounded-xl" />
          </div>
        ) : (
          <>
            {/* Flexible badge */}
            {isFlexible && (
              <span className="inline-flex items-center gap-1 rounded-full border border-utility-success-200 bg-utility-success-100 px-2 py-0.5 text-xs font-medium text-fg-success-primary">
                <Zap className="h-3 w-3" />
                Linh hoạt
              </span>
            )}

            {/* Validate error banner */}
            {validateError && (
              <div className="flex items-center gap-2 rounded-xl border border-utility-error-200 bg-utility-error-50 px-4 py-3 text-sm text-fg-error-primary">
                <span>⚠</span>
                <span>{validateError}</span>
              </div>
            )}

            {/* Compact editor controls */}
            {!isFlexible && (
              <section className="-mx-4 border-y border-[hsl(var(--surface-border))] bg-card transition-colors">
                <div className="px-4 py-3">
                  <div className="grid gap-3 min-[380px]:grid-cols-[minmax(0,1fr)_9.5rem] min-[380px]:items-end">
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/5 text-primary">
                        <CalendarDate className="h-4 w-4" aria-hidden="true" />
                      </span>
                      <div className="min-w-0">
                        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Hiệu lực cấu hình</p>
                        <p className="text-xs text-muted-foreground">Áp dụng liên tục từ ngày bắt đầu</p>
                      </div>
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label htmlFor="fromDate" className={cn(
                        'flex items-center gap-1 text-xs font-medium',
                        fromDateLocked ? 'text-muted-foreground' : 'text-foreground',
                        sf?.effective_from.status === 'error' && 'text-utility-error-700',
                      )}>
                        Từ ngày <span className="text-destructive">*</span>
                        {fromDateLocked && <Lock01 className="h-3 w-3" aria-hidden="true" />}
                      </Label>
                      <Input
                        ref={fromDateRef}
                        id="fromDate"
                        type="date"
                        value={config.fromDate || ''}
                        readOnly={fromDateLocked}
                        min={minFromDate}
                        onChange={e => {
                          if (fromDateLocked) return;
                          setConfig(prev => ({ ...prev, fromDate: e.target.value }));
                          clearServerField('effective_from');
                        }}
                        className={cn(
                          'h-11 w-full text-sm',
                          fromDateLocked && 'cursor-not-allowed bg-muted/50 opacity-60',
                          fieldBorderClass(sf?.effective_from),
                        )}
                      />
                    </div>
                  </div>
                  <div className="mt-2 space-y-2">
                    <PayratePeriodStrip payrates={allPayRatesData?.data ?? []} selectedDate={config.fromDate} />
                    {isNew && latestPayrate && (
                      <p className="mb-1 text-xs text-muted-foreground">
                        Cấu hình mới áp dụng sau cấu hình hiện hành (từ {minFromDate}). Muốn sửa mức lương đang chạy, hãy chỉnh sửa cấu hình hiện tại.
                      </p>
                    )}
                    {isEnded && (
                      <p className="mb-1 text-xs text-muted-foreground">
                        Cấu hình đã bị thay thế bởi cấu hình mới hơn — chỉ xem lịch sử. Hãy chỉnh sửa cấu hình lương hiện hành.
                      </p>
                    )}
                    <FieldFeedback
                      field={sf?.effective_from}
                      onApplySuggestion={v => {
                        setConfig(prev => ({ ...prev, fromDate: v }));
                        clearServerField('effective_from');
                      }}
                    />
                  </div>
                </div>
              </section>
            )}

            {/* Client-side validation */}
            {hasAttemptedSave && !clientValidation.valid && (
              <div className="flex items-start gap-2 rounded-xl border border-utility-warning-200 bg-utility-warning-50 px-4 py-3 text-sm text-utility-warning-700">
                <span className="mt-0.5 shrink-0">⚠</span>
                <ul className="space-y-0.5">
                  {clientValidation.errors?.map((e, i) => <li key={i}>{e}</li>)}
                </ul>
              </div>
            )}

            {/* Confirmation step: creates and rate changes */}
            {saveStep === 'confirm' && (
              <section className="rounded-xl border border-border bg-card px-4 py-3 text-sm space-y-2">
                <p className="font-semibold">
                  {isNew ? 'Xác nhận tạo cấu hình mới' : 'Xác nhận cập nhật cấu hình'}
                </p>
                <ul className="space-y-0.5 text-xs text-muted-foreground">
                  <li>Hiệu lực từ {config.fromDate}.</li>
                  {diffSummary.changedCells > 0 && <li>{diffSummary.changedCells} ô mức lương thay đổi.</li>}
                  {diffSummary.addedPositions.length > 0 && <li>Thêm vị trí: {diffSummary.addedPositions.join(', ')}.</li>}
                  {diffSummary.removedPositions.length > 0 && (
                    <li className="text-fg-warning-primary">
                      Bỏ vị trí (sẽ không tạo được bảng công mới cho vị trí này): {diffSummary.removedPositions.join(', ')}.
                    </li>
                  )}
                  {!!serverResult?.fields.rates.key_drift_warnings?.length && (
                    <li className="text-fg-warning-primary">
                      Viết khác hiện tại: {serverResult.fields.rates.key_drift_warnings.join('; ')}.
                    </li>
                  )}
                  {!!serverResult?.mutable_timesheets_affected && serverResult.mutable_timesheets_affected > 0 && (
                    <li>Sẽ tính lại {serverResult.mutable_timesheets_affected} bảng công chưa thanh toán và chưa duyệt.</li>
                  )}
                </ul>
                <div className="grid grid-cols-2 gap-2 pt-1">
                  <Button variant="outline" className="h-11 w-full" onClick={() => setSaveStep('idle')} disabled={isSaving}>
                    Quay lại sửa
                  </Button>
                  <Button className="btn-admin-primary !h-11 w-full" onClick={handleConfirmSave} disabled={isSaving}>
                    {isSaving ? 'Đang lưu...' : 'Xác nhận'}
                  </Button>
                </div>
              </section>
            )}

            {/* Matrix editor */}
            <section className={cn(
              '-mx-4 overflow-hidden border-y bg-card transition-all',
              ratesLocked ? 'border-muted' : 'border-[hsl(var(--surface-border))]',
              sf?.rates?.status === 'error' && 'ring-2 ring-utility-error-100',
            )}>
              {sf?.rates.status === 'error' && sf.rates.message && (
                <div className="mx-4 mt-3 space-y-1.5 rounded-xl border border-utility-error-200 bg-utility-error-50 px-3 py-2.5 text-xs">
                  <p className="font-semibold text-fg-error-primary">{sf.rates.message}</p>
                  {sf.rates.hint && <p className="text-utility-error-700">{sf.rates.hint}</p>}
                </div>
              )}

              <div className={cn('min-w-0 py-3', ratesLocked && 'pointer-events-none select-none opacity-60')}>
                <div className="min-w-0">
                  {isFlexible ? (
                    <FlexiblePayrateEditor
                      rates={config.rates || {}}
                      onChange={handleRatesChange}
                      readOnly={ratesLocked}
                    />
                  ) : (
                    <PayrateMatrixEditor
                      rates={config.rates || {}}
                      originalRates={editorMode === 'edit' ? (targetPayrate?.rates ?? undefined) : undefined}
                      onChange={handleRatesChange}
                      validation={hasAttemptedSave ? clientValidation : undefined}
                      readOnly={ratesLocked}
                      isFlexible={false}
                    />
                  )}
                </div>
              </div>
            </section>
          </>
        )}
      </div>

      {/* Actions remain in document flow above the mobile navigation. */}
      <footer className="mt-4 grid grid-cols-1 gap-2 border-t border-border py-4 min-[380px]:grid-cols-2">
        <Button variant="outline" className="h-11 w-full" onClick={goBack} disabled={isValidating || isSaving}>
          Hủy
        </Button>
        <Button
          className="btn-admin-primary !h-11 w-full"
          onClick={handleValidate}
          disabled={isValidating || isSaving || !canProceed || isEnded}
        >
          {isValidating || isSaving ? (
            <span className="flex items-center gap-1.5">
              <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent" />
              {isSaving ? 'Đang lưu...' : 'Đang kiểm tra...'}
            </span>
          ) : (
            <span className="flex items-center gap-1.5">
              <Save01 className="h-3.5 w-3.5" />
              {editorMode === 'create' ? 'Tạo cấu hình' : 'Lưu thay đổi'}
            </span>
          )}
        </Button>
      </footer>
    </MobilePageShell>
  );
}
