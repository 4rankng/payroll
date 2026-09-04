import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Save, Lock, CornerDownRight, Zap, CalendarClock } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import { MobileSubPageHeader } from '@/components/shared/MobileSubPageHeader';
import { PayrateMatrixEditor } from '@/components/payrates/components/PayrateMatrixEditor';
import { FlexiblePayrateEditor } from '@/components/payrates/components/FlexiblePayrateEditor';
import { PayrateJsonEditor } from '@/components/payrates/components/PayrateJsonEditor';
import {
  type PayrateConfig,
  type PayrateStructure,
  type EditorMode,
  validatePayrateStructure,
  validateFlexiblePayrateStructure,
  createDefaultPayrateStructure,
  createFlexiblePayrateStructure,
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

type SaveStep = 'idle' | 'validating' | 'confirmed' | 'saving';

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
        ? 'border border-red-200 bg-red-50 text-red-700'
        : 'border border-amber-200 bg-amber-50 text-amber-700',
    )}>
      <p className="font-medium">{field.message}</p>
      {field.hint && <p className="mt-0.5 opacity-80">{field.hint}</p>}
      {field.suggested_value && onApplySuggestion && (
          <button
            type="button"
            onClick={() => onApplySuggestion(field.suggested_value!)}
            className={cn(
              'mt-1.5 flex min-h-11 items-center gap-1 text-xs font-semibold underline underline-offset-2',
              isError ? 'hover:text-red-900' : 'hover:text-amber-900',
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
    return highlighted ? 'border-amber-400 ring-2 ring-amber-200 bg-amber-50/40' : '';
  }
  if (field.status === 'error') return 'border-red-400 ring-1 ring-red-200 bg-red-50/30';
  if (field.status === 'warning') return 'border-amber-400 ring-1 ring-amber-200 bg-amber-50/30';
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

  const editorMode: EditorMode = isNew ? 'create' : 'edit';

  const [config, setConfig] = useState<Partial<PayrateConfig>>({
    project_id: numProjectId,
    rates: createDefaultPayrateStructure(),
    fromDate: dateToString(new Date()),
  });

  const [viewMode, setViewMode] = useState<'matrix' | 'json'>('matrix');
  const [saveStep, setSaveStep] = useState<SaveStep>('idle');
  const [serverResult, setServerResult] = useState<DryRunValidateResponse | null>(null);
  const [validateError, setValidateError] = useState<string | null>(null);
  const [hasAttemptedSave, setHasAttemptedSave] = useState(false);

  const fromDateRef = useRef<HTMLInputElement>(null);

  const isFlexible = !!project?.is_flexible;

  const flexibleInitDone = useRef(false);
  useEffect(() => {
    if (!isNew || !isFlexible || flexibleInitDone.current) return;
    flexibleInitDone.current = true;
    setConfig(prev => ({ ...prev, rates: createFlexiblePayrateStructure() }));
  }, [isFlexible, isNew]);

  useEffect(() => {
    if (targetPayrate) {
      // Default the start date to the paid floor (day after the latest paid
      // timesheet work date) whenever the stored start sits before it — the backend
      // rejects anything earlier, so pre-clamp instead of surfacing an error.
      const floor = targetPayrate.earliest_effective_from;
      const fromDate =
        floor && !targetPayrate.toDate && targetPayrate.fromDate < floor ? floor : targetPayrate.fromDate;
      setConfig({
        project_id: numProjectId,
        rates: targetPayrate.rates || {},
        fromDate,
      });
    }
  }, [targetPayrate, numProjectId]);

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

  // Ended configs (a newer config took over) are history: rows under them are
  // priced, and the backend rejects any update. Render everything read-only
  // instead of offering an edit that can only fail.
  const isEnded = editorMode === 'edit' && !!targetPayrate?.toDate;
  const ratesLocked = isEnded || (serverResult?.fields.rates.locked ?? false);
  const fromDateLocked = isEnded || (serverResult?.fields.effective_from.locked ?? false) || (editorMode === 'edit' && !!targetPayrate?.from_date_locked);

  const handleRatesChange = useCallback((rates: PayrateStructure) => {
    if (ratesLocked) return;
    setConfig(prev => ({ ...prev, rates }));
    clearServerField('rates');
  }, [ratesLocked, clearServerField]);

  const basePath = authManager.getUserRole() === 'partner' ? '/partner' : '/admin';
  const goBack = useCallback(() => {
    navigate(`${basePath}/projects?modal=project_details&id=${numProjectId}`);
  }, [navigate, basePath, numProjectId]);

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

      if (result.valid) {
        setSaveStep('saving');
        try {
          // Rate changes version append-only: create a new config; the backend
          // closes the previous one the day before the new effective_from.
          // Only a start-date correction without rate changes updates in place
          // (re-pricing mutable, i.e. unpaid and unapproved, timesheets).
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
        } catch {
          setSaveStep('idle');
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
  }, [clientValidation.valid, config, numProjectId, numPayrateId, editorMode, targetPayrate, createMutation, updateMutation, goBack]);

  const isLoading = isLoadingProject || isLoadingRates;
  const isValidating = saveStep === 'validating';
  const isSaving = saveStep === 'saving';
  const canProceed = clientValidation.valid && !!config.fromDate && !isLoading;

  const sf = serverResult?.fields;

  // Server locks all fields only when the project is completed/cancelled.
  // Otherwise every field stays editable: moving the start date earlier or
  // changing rates re-prices mutable (unpaid, unapproved) timesheets on save.
  const fromDateIsActionable = false;

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
              <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">
                <Zap className="h-3 w-3" />
                Linh hoạt
              </span>
            )}

            {/* Validate error banner */}
            {validateError && (
              <div className="flex items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                <span>⚠</span>
                <span>{validateError}</span>
              </div>
            )}

            {/* Compact editor controls */}
            <section className={cn(
              '-mx-4 border-y bg-card transition-colors',
              fromDateIsActionable ? 'border-amber-400 bg-amber-50/30' : 'border-[hsl(var(--surface-border))]',
            )}>
              {!isFlexible && (
                <div className="px-4 py-3">
                  <div className="grid gap-3 min-[380px]:grid-cols-[minmax(0,1fr)_9.5rem] min-[380px]:items-end">
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/5 text-primary">
                        <CalendarClock className="h-4 w-4" aria-hidden="true" />
                      </span>
                      <div className="min-w-0">
                        <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Hiệu lực cấu hình</p>
                        <p className="text-xs text-muted-foreground">Áp dụng liên tục từ ngày bắt đầu</p>
                      </div>
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label htmlFor="fromDate" className={cn(
                        'flex items-center gap-1 text-xs font-medium',
                        fromDateLocked && !fromDateIsActionable ? 'text-muted-foreground' : 'text-foreground',
                        sf?.effective_from.status === 'error' && !fromDateIsActionable && 'text-red-600',
                        fromDateIsActionable && 'font-semibold text-amber-700',
                      )}>
                        Từ ngày <span className="text-destructive">*</span>
                        {fromDateLocked && !fromDateIsActionable && <Lock className="h-3 w-3" aria-hidden="true" />}
                      </Label>
                      <Input
                        ref={fromDateRef}
                        id="fromDate"
                        type="date"
                        value={config.fromDate || ''}
                        readOnly={fromDateLocked && !fromDateIsActionable}
                        min={fromDateIsActionable ? sf?.rates.suggested_value : (sf?.effective_from.min_value || targetPayrate?.earliest_effective_from)}
                        autoFocus={fromDateIsActionable}
                        onChange={e => {
                          if (fromDateLocked && !fromDateIsActionable) return;
                          setConfig(prev => ({ ...prev, fromDate: e.target.value }));
                          clearServerField('effective_from');
                        }}
                        className={cn(
                          'h-11 w-full text-sm',
                          fromDateLocked && !fromDateIsActionable && 'cursor-not-allowed bg-muted/50 opacity-60',
                          fromDateIsActionable && 'border-amber-400 bg-amber-50/40 ring-2 ring-amber-200',
                          !fromDateIsActionable && fieldBorderClass(sf?.effective_from),
                        )}
                      />
                    </div>
                  </div>
                  {fromDateIsActionable && sf?.rates.suggested_value && (
                    <div className="mt-3 space-y-1 border-l-2 border-amber-300 pl-3 text-xs text-amber-800">
                      <p className="font-medium">Cần tạo cấu hình mới để thay đổi mức lương.</p>
                      <p>Các bảng công chưa thanh toán và chưa duyệt từ ngày này sẽ được cập nhật.</p>
                      <p>Ngày bắt đầu sớm nhất: <strong>{sf.rates.suggested_value}</strong>.</p>
                    </div>
                  )}
                  {!fromDateIsActionable && (
                    <FieldFeedback
                      field={sf?.effective_from}
                      onApplySuggestion={v => {
                        setConfig(prev => ({ ...prev, fromDate: v }));
                        clearServerField('effective_from');
                      }}
                    />
                  )}
                </div>
              )}

              <div className={cn('grid grid-cols-2 border-t border-border/70', isFlexible && 'border-t-0')} role="tablist" aria-label="Chế độ chỉnh sửa cấu hình lương">
                <button
                  type="button"
                  role="tab"
                  aria-selected={viewMode === 'matrix'}
                  onClick={() => setViewMode('matrix')}
                  className={cn(
                    'min-h-11 border-b-2 px-4 text-xs font-semibold transition-colors',
                    viewMode === 'matrix' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground',
                  )}
                >
                  Ma trận
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={viewMode === 'json'}
                  onClick={() => setViewMode('json')}
                  className={cn(
                    'min-h-11 border-b-2 px-4 text-xs font-semibold transition-colors',
                    viewMode === 'json' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground',
                  )}
                >
                  JSON
                </button>
              </div>
            </section>

            {/* Client-side validation */}
            {hasAttemptedSave && !clientValidation.valid && (
              <div className="flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
                <span className="mt-0.5 shrink-0">⚠</span>
                <ul className="space-y-0.5">
                  {clientValidation.errors?.map((e, i) => <li key={i}>{e}</li>)}
                </ul>
              </div>
            )}

            {/* Matrix / JSON editor */}
            <section className={cn(
              '-mx-4 overflow-hidden border-y bg-card transition-all',
              ratesLocked ? 'border-muted' : 'border-[hsl(var(--surface-border))]',
              sf?.rates?.status === 'error' && 'ring-2 ring-red-100',
            )}>
              {sf?.rates.status === 'error' && sf.rates.message && !fromDateIsActionable && (
                <div className="mx-4 mt-3 space-y-1.5 rounded-xl border border-red-200 bg-red-50 px-3 py-2.5 text-xs">
                  <p className="font-semibold text-red-800">{sf.rates.message}</p>
                  {sf.rates.hint && <p className="text-red-700">{sf.rates.hint}</p>}
                </div>
              )}

              <div className={cn('min-w-0 py-3', ratesLocked && 'pointer-events-none select-none opacity-60')}>
                <div className="min-w-0">
                  {viewMode === 'matrix' ? (
                    isFlexible ? (
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
                    )
                  ) : (
                    <PayrateJsonEditor
                      rates={config.rates || {}}
                      onChange={handleRatesChange}
                      readOnly={ratesLocked}
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
              <Save className="h-3.5 w-3.5" />
              {editorMode === 'create' ? 'Tạo cấu hình' : 'Lưu thay đổi'}
            </span>
          )}
        </Button>
      </footer>
    </MobilePageShell>
  );
}
