import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Save, Lock, CornerDownRight, Zap } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
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

// ── Inline field feedback component ──────────────────────────────────────────

function FieldFeedback({ field, onApplySuggestion }: {
  field: FieldResult | undefined;
  onApplySuggestion?: (value: string) => void;
}) {
  if (!field || field.status === 'ok' || !field.message) return null;

  const isError = field.status === 'error';
  return (
    <div className={cn(
      "mt-1.5 rounded-xl px-3 py-2 text-xs space-y-1",
      isError ? "bg-red-50 border border-red-200 text-red-700" : "bg-amber-50 border border-amber-200 text-amber-700"
    )}>
      <p className="font-medium">{field.message}</p>
      {field.hint && (
        <p className="text-xs opacity-80">{field.hint}</p>
      )}
      {field.suggested_value && onApplySuggestion && (
        <button
          type="button"
          onClick={() => onApplySuggestion(field.suggested_value!)}
          className={cn(
            "flex items-center gap-1 text-xs font-semibold mt-1 underline underline-offset-2",
            isError ? "text-red-700 hover:text-red-900" : "text-amber-700 hover:text-amber-900"
          )}
        >
          <CornerDownRight className="h-3 w-3" />
          Dùng giá trị: {field.suggested_value}
        </button>
      )}
    </div>
  );
}

// ── Input border class from field status ─────────────────────────────────────

function fieldBorderClass(field: FieldResult | undefined, highlighted = false): string {
  if (!field || field.status === 'ok') {
    return highlighted ? 'border-amber-400 ring-2 ring-amber-200 bg-amber-50/40' : '';
  }
  if (field.status === 'error') return 'border-red-400 ring-1 ring-red-200 bg-red-50/30';
  if (field.status === 'warning') return 'border-amber-400 ring-1 ring-amber-200 bg-amber-50/30';
  return '';
}

// ── Page ─────────────────────────────────────────────────────────────────────

export default function PayrateEditPage() {
  const { projectId, payrateId } = useParams<{ projectId: string; payrateId: string }>();
  const navigate = useNavigate();

  const numProjectId = Number(projectId);
  const isNew = payrateId === 'new';
  const numPayrateId = isNew ? null : Number(payrateId);

  const { data: projectData, isLoading: isLoadingProject } = useProject(numProjectId);
  const { data: allPayRatesData, isLoading: isLoadingRates } = useProjectPayRates(numProjectId);
  const createMutation = useCreateProjectPayRate();
  const updateMutation = useUpdatePayRate();

  // projectData IS the Project object (getProjectById unwraps the ApiResponse)
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
  // Only show client-side validation errors after the user first presses save
  const [hasAttemptedSave, setHasAttemptedSave] = useState(false);

  // Ref to auto-scroll to first error field after validation
  const toDateRef = useRef<HTMLInputElement>(null);
  const fromDateRef = useRef<HTMLInputElement>(null);

  const isFlexible = !!project?.is_flexible;

  // Once we know the project is flexible and this is a new payrate,
  // replace the generic default structure with an empty flexible one.
  // The ref prevents this from re-running on subsequent renders.
  const flexibleInitDone = useRef(false);
  useEffect(() => {
    if (!isNew || !isFlexible || flexibleInitDone.current) return;
    flexibleInitDone.current = true;
    setConfig(prev => ({ ...prev, rates: createFlexiblePayrateStructure() }));
  }, [isFlexible, isNew]);

  useEffect(() => {
    if (targetPayrate) {
      setConfig({
        project_id: numProjectId,
        rates: targetPayrate.rates || {},
        fromDate: targetPayrate.fromDate,
        toDate: targetPayrate.toDate,
      });
    }
  }, [targetPayrate, numProjectId]);

  // Clear server result when user edits a field
  const clearServerField = useCallback((field: 'effective_from' | 'effective_to' | 'rates') => {
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

  const ratesLocked = serverResult?.fields.rates.locked ?? false;
  const fromDateLocked = serverResult?.fields.effective_from.locked ?? false;

  const handleRatesChange = useCallback((rates: PayrateStructure) => {
    if (ratesLocked) return;
    setConfig(prev => ({ ...prev, rates }));
    clearServerField('rates');
  }, [ratesLocked, clearServerField]);

  const basePath = authManager.getUserRole() === 'partner' ? '/partner' : '/admin';
  const goBack = useCallback(() => {
    navigate(`${basePath}/projects?modal=project_details&id=${numProjectId}`);
  }, [navigate, basePath, numProjectId]);

  // Step 1: call validate endpoint
  const handleValidate = useCallback(async () => {
    setHasAttemptedSave(true); // Reveal client-side errors from now on
    if (!clientValidation.valid || !config.fromDate) return;
    setSaveStep('validating');
    setServerResult(null);
    setValidateError(null);
    try {
      const result = await payRateService.dryRunValidate(
        numProjectId,
        { rates: config.rates, effective_from: config.fromDate, effective_to: config.toDate },
        numPayrateId ?? undefined,
      );
      setServerResult(result);

      if (result.valid) {
        setSaveStep('saving');
        // validation passed — save immediately, no confirmation step
        try {
          const startDateChanged =
            editorMode === 'edit' && targetPayrate?.fromDate && targetPayrate.fromDate !== config.fromDate;
          if (editorMode === 'edit' && targetPayrate?.id && !startDateChanged) {
            await updateMutation.mutateAsync({
              payRateId: targetPayrate.id,
              data: { rates: config.rates!, effective_from: config.fromDate!, effective_to: config.toDate || undefined },
            });
          } else {
            await createMutation.mutateAsync({
              projectId: numProjectId,
              data: { rates: config.rates!, effective_from: config.fromDate!, effective_to: config.toDate || undefined },
            });
          }
          goBack();
        } catch {
          setSaveStep('idle');
        }
      } else {
        setSaveStep('idle');
        // Auto-focus the first broken field
        setTimeout(() => {
          if (result.fields.effective_from.status === 'error') {
            fromDateRef.current?.focus();
          } else if (result.fields.effective_to.status === 'error') {
            toDateRef.current?.focus();
          }
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

  // Determine which field is the action point based on server result
  // RATES_LOCKED + suggested_value on rates → user must create NEW config → highlight fromDate
  // RATES_LOCKED without suggested_value change → only toDate can change → highlight toDate
  const ratesLockedNeedNewConfig = ratesLocked && !!sf?.rates.suggested_value;
  const toDateIsActionable = ratesLocked && !ratesLockedNeedNewConfig;
  const fromDateIsActionable = ratesLockedNeedNewConfig;

  return (
    <div className="flex flex-col min-h-screen bg-background">

      {/* ── Top bar ── */}
      <header className="sticky top-0 z-20 bg-background/95 backdrop-blur-sm border-b border-border">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 h-14 flex items-center gap-3">
          <button onClick={goBack} className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors shrink-0">
            <ArrowLeft className="h-4 w-4" />
            Quay lại
          </button>
          <div className="h-4 w-px bg-border shrink-0" />
          <div className="flex-1 min-w-0">
            {isLoadingProject
              ? <Skeleton className="h-4 w-32" />
              : <span className="text-sm font-medium truncate">{project?.name}</span>}
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Button variant="ghost" size="sm" onClick={goBack} disabled={isValidating || isSaving} className="hidden sm:flex">
              Hủy
            </Button>
            <Button size="sm" variant="default" onClick={handleValidate} disabled={isValidating || isSaving || !canProceed} className="min-w-[130px]">
              {isValidating || isSaving ? (
                <span className="flex items-center gap-1.5">
                  <span className="h-3.5 w-3.5 rounded-full border-2 border-current border-t-transparent animate-spin" />
                  {isSaving ? 'Đang lưu...' : 'Đang kiểm tra...'}
                </span>
              ) : (
                <span className="flex items-center gap-1.5">
                  <Save className="h-3.5 w-3.5" />
                  {editorMode === 'create' ? 'Tạo cấu hình' : 'Lưu thay đổi'}
                </span>
              )}
            </Button>
          </div>
        </div>
      </header>

      {/* ── Page title ── */}
      <div className="max-w-4xl mx-auto w-full px-4 sm:px-6 pt-6 pb-2">
        <div className="flex items-center gap-2 flex-wrap">
          <h1 className="text-xl font-semibold">
            {isNew ? 'Tạo cấu hình lương mới' : 'Chỉnh sửa cấu hình lương'}
          </h1>
          {isFlexible && (
            <span className="inline-flex items-center gap-1 text-xs font-medium bg-violet-100 text-violet-700 border border-violet-200 px-2 py-0.5 rounded-full">
              <Zap className="h-3 w-3" />
              Linh hoạt
            </span>
          )}
        </div>
        <p className="text-sm text-muted-foreground mt-0.5">
          {isNew
            ? isFlexible
              ? 'Thiết lập ca làm việc (khung giờ) và mức lương theo vị trí, loại ngày'
              : 'Thiết lập mức lương theo vị trí, loại ngày và khung giờ'
            : 'Cập nhật mức lương cho cấu hình hiện tại'}
        </p>
      </div>

      {/* ── Body ── */}
      <main className="flex-1 max-w-4xl mx-auto w-full px-4 sm:px-6 py-4 space-y-5">
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-20 rounded-xl" />
            <Skeleton className="h-64 rounded-xl" />
          </div>
        ) : (
          <>
            {/* ── Validate error banner ── */}
            {validateError && (
              <div className="flex items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                <span>⚠</span>
                <span>{validateError}</span>
              </div>
            )}

            {/* ── Effective period (hidden for flexible — edits take effect immediately) ── */}
            {!isFlexible && (<section className={cn(
              "rounded-xl border bg-card p-4 transition-all",
              (toDateIsActionable || fromDateIsActionable) ? "border-amber-400 ring-2 ring-amber-200" : "border-border",
            )}>
              <div className="flex items-center justify-between mb-3">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
                  Thời gian hiệu lực
                </p>
                {toDateIsActionable && (
                  <span className="text-xs font-semibold text-amber-700 bg-amber-100 px-2 py-0.5 rounded-full">
                    ← Chỉ có thể sửa ngày kết thúc
                  </span>
                )}
              </div>

              <div className="flex flex-wrap gap-6">
                {/* From date */}
                <div className="flex flex-col gap-0.5">
                  <Label htmlFor="fromDate" className={cn(
                    "text-xs font-medium flex items-center gap-1",
                    fromDateLocked && !fromDateIsActionable ? "text-muted-foreground" : "text-foreground",
                    sf?.effective_from.status === 'error' && !fromDateIsActionable && "text-red-600",
                    fromDateIsActionable && "text-amber-700 font-semibold",
                  )}>
                    Từ ngày <span className="text-destructive">*</span>
                    {fromDateLocked && !fromDateIsActionable && <Lock className="h-3 w-3 text-muted-foreground" />}
                    {fromDateIsActionable && <span className="text-[10px] font-normal text-amber-600">(cập nhật tại đây)</span>}
                  </Label>
                  <Input
                    ref={fromDateRef}
                    id="fromDate"
                    type="date"
                    value={config.fromDate || ''}
                    readOnly={fromDateLocked && !fromDateIsActionable}
                    min={fromDateIsActionable ? sf?.rates.suggested_value : sf?.effective_from.min_value}
                    autoFocus={fromDateIsActionable}
                    onChange={e => {
                      if (fromDateLocked && !fromDateIsActionable) return;
                      setConfig(prev => ({ ...prev, fromDate: e.target.value }));
                      clearServerField('effective_from');
                    }}
                    className={cn(
                      "h-8 text-sm w-40",
                      fromDateLocked && !fromDateIsActionable && "bg-muted/50 cursor-not-allowed opacity-60",
                      fromDateIsActionable && "border-amber-400 ring-2 ring-amber-200 bg-amber-50/40",
                      !fromDateIsActionable && fieldBorderClass(sf?.effective_from),
                    )}
                  />
                  {/* When fromDateIsActionable, show the suggested new start date */}
                  {fromDateIsActionable && sf?.rates.suggested_value && (
                    <div className="mt-1.5 rounded-xl px-3 py-2 text-xs space-y-1 bg-amber-50 border border-amber-200 text-amber-800">
                      <p className="font-medium">Cần tạo cấu hình mới để thay đổi mức lương.</p>
                      <p className="text-xs opacity-80">Ngày bắt đầu sớm nhất có thể: <strong>{sf.rates.suggested_value}</strong> (ngày sau bảng công gần nhất).</p>
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

                {/* To date */}
                <div className="flex flex-col gap-0.5">
                  <Label htmlFor="toDate" className={cn(
                    "text-xs font-medium flex items-center gap-1",
                    toDateIsActionable ? "text-amber-700 font-semibold" : "text-muted-foreground",
                    sf?.effective_to.status === 'error' && "text-red-600",
                  )}>
                    Đến ngày
                    {toDateIsActionable
                      ? <span className="text-[10px] text-amber-600 font-normal">(cập nhật tại đây)</span>
                      : <span className="text-[10px] font-normal">(tuỳ chọn)</span>}
                  </Label>
                  <Input
                    ref={toDateRef}
                    id="toDate"
                    type="date"
                    value={config.toDate || ''}
                    min={sf?.effective_to.min_value}
                    autoFocus={toDateIsActionable}
                    onChange={e => {
                      setConfig(prev => ({ ...prev, toDate: e.target.value || null }));
                      clearServerField('effective_to');
                    }}
                    className={cn(
                      "h-8 text-sm w-40",
                      toDateIsActionable && !sf?.effective_to.status
                        ? "border-amber-400 ring-2 ring-amber-200 bg-amber-50/40"
                        : fieldBorderClass(sf?.effective_to),
                    )}
                  />
                  <FieldFeedback
                    field={sf?.effective_to}
                    onApplySuggestion={v => {
                      setConfig(prev => ({ ...prev, toDate: v }));
                      clearServerField('effective_to');
                    }}
                  />
                </div>
              </div>
            </section>)}

            {/* ── Client-side validation — only shown after first save attempt ── */}
            {hasAttemptedSave && !clientValidation.valid && (
              <div className="flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
                <span className="shrink-0 mt-0.5">⚠</span>
                <ul className="space-y-0.5">
                  {clientValidation.errors?.map((e, i) => <li key={i}>{e}</li>)}
                </ul>
              </div>
            )}

            {/* ── Matrix editor ── */}
            <div className="flex justify-end mb-2">
              <div className="flex bg-muted/50 p-1 rounded-lg">
                <button type="button" onClick={() => setViewMode('matrix')} className={cn("px-3 py-1.5 text-xs font-medium rounded-md transition-colors", viewMode === 'matrix' ? 'bg-white shadow-sm text-slate-900' : 'text-slate-500 hover:text-slate-900')}>Ma trận</button>
                <button type="button" onClick={() => setViewMode('json')} className={cn("px-3 py-1.5 text-xs font-medium rounded-md transition-colors", viewMode === 'json' ? 'bg-white shadow-sm text-slate-900' : 'text-slate-500 hover:text-slate-900')}>JSON (Linh hoạt)</button>
              </div>
            </div>

            <section className={cn(
              "rounded-xl border bg-card overflow-hidden transition-all",
              ratesLocked ? "border-muted" : "border-border",
              sf?.rates?.status === 'error' && "border-red-300 ring-2 ring-red-100",
            )}>

              {/* Rates field feedback — shown inside the section */}
              {sf?.rates.status === 'error' && sf.rates.message && !fromDateIsActionable && (
                <div className="mx-4 mt-3 rounded-xl border border-red-200 bg-red-50 px-3 py-2.5 text-xs space-y-1.5">
                  <p className="font-semibold text-red-800">{sf.rates.message}</p>
                  {sf.rates.hint && <p className="text-red-700">{sf.rates.hint}</p>}
                </div>
              )}

              <div className={cn("p-4", ratesLocked && "opacity-60 pointer-events-none select-none")}>
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
            </section>
          </>
        )}
      </main>

      {/* ── Mobile save bar ── */}
      <footer className="sticky bottom-0 sm:hidden border-t border-border bg-background/95 backdrop-blur-sm px-4 py-3 flex gap-3">
        <Button variant="outline" className="flex-1" onClick={goBack} disabled={isValidating || isSaving}>
          Hủy
        </Button>
        <Button className="flex-1" variant="default" onClick={handleValidate} disabled={isValidating || isSaving || !canProceed}>
          {isValidating || isSaving ? 'Đang xử lý...' : editorMode === 'create' ? 'Tạo cấu hình' : 'Lưu thay đổi'}
        </Button>
      </footer>
    </div>
  );
}
