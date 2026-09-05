import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SearchableSelect } from "@/components/ui/searchable-select";
import {
  ChevronLeft,
  Save,
  SlidersHorizontal,
  Check,
  AlertTriangle,
  Trash2,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { UserAvatar } from "@/components/ui/user-avatar";
import { useCreateTimesheets } from "@/hooks/api/useTimesheets";
import { useTimesheetProjects } from "@/hooks/api/useProjects";
import { useMultiTimesheetFormState } from "../hooks/useMultiTimesheetFormState";
import { getDefaultDateRange } from "@/utils/date-range.utils";
import { useBottomNav } from "@/contexts";
import { DateRangePicker } from "@/components/ui/date-range-picker";
import type { NewTimesheetEntry } from "@/types/api/timesheet.types";
import type { TimesheetEntry } from "../types/multi-timesheet.types";
import { formatCurrency } from "@/utils/formatters";
import {
  getHourStatus,
  WEEKDAY,
} from "../utils/timesheetHelpers";
import { ErrorBanner } from "../components/ErrorBanner";
import { DeletionStateBanner } from "../components/DeletionStateBanner";
import { HoursWarningBadge } from "../components/HoursWarningBadge";
import { HourInputField } from "../components/HourInputField";
import { DateStrip } from "../components/DateStrip";
import { calculateTimesheetPreviewAmount } from "@/components/timesheet/components/timesheet-pay-unit";

interface MobileTimesheetEntryProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId?: number;
  projectId?: number;
  onSuccess?: () => Promise<void>;
}

export function MobileTimesheetEntry({
  isOpen,
  onClose,
  employeeId,
  projectId,
  onSuccess,
}: MobileTimesheetEntryProps) {
  const [isClosing, setIsClosing] = useState(false);
  const [dateRange, setDateRange] = useState(() => getDefaultDateRange());
  const [filterOpen, setFilterOpen] = useState(false);
  const [filterEmployeeId, setFilterEmployeeId] = useState<number | null>(null);
  const [activeDates, setActiveDates] = useState<Record<number, string>>({});
  const [rawInputs, setRawInputs] = useState<Record<string, string>>({});
  const autoRef = useRef("");
  const createMutation = useCreateTimesheets();

  const { data: projectsData } = useTimesheetProjects({ enabled: isOpen });
  const projects = useMemo(() => projectsData?.data || [], [projectsData]);

  const {
    formData,
    availableEmployees,
    validation,
    getDayTypesForPosition,
    getHourTypesForEntry,
    hasPayRateForEntry,
    getPayRateForEntry,
    transformEntriesToAPI,
    handleProjectChange,
    handleAddEntry,
    handleRemoveEntry,
    handleEntryChange,
    handleDateRangeChange,
    resetForm,
    triggerPreview,
    isPayRateReady,
    isLoadingPayRate,
    payRateForbidden,
    isPreviewLoading,
    allEntriesValid,
    getValidationErrors,
    getErrorsForEmployee,
    getErrorsForRow,
    fetchAndMergeExistingEntries,
  } = useMultiTimesheetFormState({ employeeId, projectId, isOpen, isClosing });

  const selectedProject = useMemo(
    () => projects.find((p) => p.id === formData.projectId) || null,
    [projects, formData.projectId],
  );
  const isFlexibleProject = selectedProject?.is_flexible === true;
  const dateBounds = useMemo(
    () => ({ min: new Date(Date.now() - 14 * 86400000), max: new Date() }),
    [],
  );

  const dates = useMemo(() => {
    const list: string[] = [];
    const s = new Date(dateRange.startDate),
      e = new Date(dateRange.endDate);
    for (let d = new Date(s); d <= e; d.setDate(d.getDate() + 1))
      list.push(d.toISOString().split("T")[0]);
    return list;
  }, [dateRange]);

  const visibleEmployees = useMemo(() => {
    const base = availableEmployees.filter(
      (e) =>
        !formData.projectId ||
        e.current_projects?.some((p) => p.project_id === formData.projectId),
    );
    return filterEmployeeId
      ? base.filter((e) => e.id === filterEmployeeId)
      : base;
  }, [availableEmployees, formData.projectId, filterEmployeeId]);

  // auto-populate
  useEffect(() => {
    if (
      !isOpen ||
      !formData.projectId ||
      !availableEmployees.length ||
      !isPayRateReady ||
      isClosing
    )
      return;
    const key = `${formData.projectId}-${dateRange.startDate}-${dateRange.endDate}`;
    if (autoRef.current === key) return;
    const emps = availableEmployees.filter((e) =>
      e.current_projects?.some((p) => p.project_id === formData.projectId),
    );
    if (!emps.length) return;
    if (
      formData.entries.some(
        (e) => e.employeeId && e.projectId === formData.projectId,
      )
    ) {
      autoRef.current = key;
      return;
    }
    const start = new Date(dateRange.startDate),
      end = new Date(dateRange.endDate);
    const days = Math.ceil((end.getTime() - start.getTime()) / 86400000) + 1;
    emps.forEach((emp) => {
      for (let i = 0; i < days; i++) {
        const d = new Date(start);
        d.setDate(start.getDate() + i);
        handleAddEntry(emp, d.toISOString().split("T")[0]);
      }
    });
    autoRef.current = key;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    isOpen,
    formData.projectId,
    availableEmployees,
    dateRange,
    isPayRateReady,
    isClosing,
  ]);

  useEffect(() => {
    if (
      isOpen &&
      !isClosing &&
      dateRange.startDate &&
      dateRange.endDate &&
      formData.projectId &&
      availableEmployees.length
    )
      fetchAndMergeExistingEntries(
        formData.projectId,
        dateRange.startDate,
        dateRange.endDate,
      );
  }, [
    isOpen,
    isClosing,
    dateRange.startDate,
    dateRange.endDate,
    formData.projectId,
    availableEmployees.length,
    fetchAndMergeExistingEntries,
  ]);

  useEffect(() => {
    if (!isOpen && !isClosing) {
      resetForm();
      setActiveDates({});
      setRawInputs({});
      setDateRange(getDefaultDateRange());
      autoRef.current = "";
      setIsClosing(false);
      setFilterEmployeeId(null);
      setFilterOpen(false);
    }
  }, [isOpen, isClosing, resetForm]);

  const handleClose = useCallback(() => {
    setIsClosing(true);
    onClose();
  }, [onClose]);

  const handleProjectSelect = useCallback(
    (id: string) => {
      handleProjectChange(id === "all" ? 0 : Number(id));
      autoRef.current = "";
      setFilterEmployeeId(null);
    },
    [handleProjectChange],
  );

  const handleStartDate = useCallback(
    (s: string) => {
      setDateRange((cur) => {
        const e = cur.endDate < s ? s : cur.endDate;
        handleDateRangeChange(s, e);
        return { ...cur, startDate: s, endDate: e };
      });
    },
    [handleDateRangeChange],
  );

  const handleEndDate = useCallback(
    (e: string) => {
      setDateRange((cur) => {
        const s = cur.startDate > e ? e : cur.startDate;
        handleDateRangeChange(s, e);
        return { ...cur, startDate: s, endDate: e };
      });
    },
    [handleDateRangeChange],
  );

  const handleHourChange = useCallback(
    (
      globalIdx: number,
      entry: TimesheetEntry,
      hourType: string,
      raw: string,
    ) => {
      setRawInputs((p) => ({ ...p, [`${entry.id}-${hourType}`]: raw }));
      const num = raw === "" ? 0 : parseFloat(raw) || 0;
      const hours = { ...entry.hours };
      if (num <= 0) {
        if (entry.originalValues) {
          hours[hourType] = 0;
        } else {
          delete hours[hourType];
        }
      } else {
        hours[hourType] = num;
      }
      handleEntryChange(globalIdx, "hours", hours);
    },
    [handleEntryChange],
  );

  const handleDayTypeChange = useCallback(
    (globalIdx: number, value: string) => {
      handleEntryChange(globalIdx, "dayType", value);
      setTimeout(() => triggerPreview?.(), 0);
    },
    [handleEntryChange, triggerPreview],
  );

  const handleDeleteEntry = useCallback(
    (globalIdx: number) => {
      handleRemoveEntry(globalIdx);
      setTimeout(() => triggerPreview?.(), 0);
    },
    [handleRemoveEntry, triggerPreview],
  );

  const handleUndoDelete = useCallback(
    (globalIdx: number, entry: TimesheetEntry) => {
      if (entry.originalValues) {
        handleEntryChange(globalIdx, "hours", { ...entry.originalValues.hours });
        setRawInputs((p) => {
          const next = { ...p };
          Object.entries(entry.originalValues!.hours).forEach(([ht, v]) => {
            next[`${entry.id}-${ht}`] = v > 0 ? String(v) : "";
          });
          return next;
        });
        setTimeout(() => triggerPreview?.(), 0);
      }
    },
    [handleEntryChange, triggerPreview],
  );

  const handleSave = useCallback(async () => {
    if (
      !validation.valid ||
      isClosing ||
      !allEntriesValid ||
      getValidationErrors()
    )
      return;
    try {
      await createMutation.mutateAsync(
        transformEntriesToAPI(formData.entries) as NewTimesheetEntry[],
      );
      resetForm(formData.projectId);
      autoRef.current = "";
      if (formData.projectId)
        await fetchAndMergeExistingEntries(
          formData.projectId,
          dateRange.startDate,
          dateRange.endDate,
        );
      if (onSuccess) await onSuccess();
    } catch {
      /* handled by mutation */
    }
  }, [
    validation,
    isClosing,
    allEntriesValid,
    getValidationErrors,
    transformEntriesToAPI,
    formData,
    createMutation,
    resetForm,
    fetchAndMergeExistingEntries,
    dateRange,
    onSuccess,
  ]);

  const isLoading = createMutation.isPending;
  const canSave =
    !isLoading &&
    validation.valid &&
    allEntriesValid &&
    !isPreviewLoading &&
    !!formData.projectId;
  const activeFilterCount = filterEmployeeId ? 1 : 0;

  const calcEarnings = useCallback(
    (entries: TimesheetEntry[]) =>
      entries.reduce((sum, entry) => {
        if (!entry.hours || !entry.position || !entry.dayType) return sum;
        return (
          sum +
          Object.entries(entry.hours).reduce(
            (s, [ht, h]) =>
              h > 0
                ? s +
                  h * getPayRateForEntry(entry.position!, entry.dayType!, ht)
                : s,
            0,
          )
        );
      }, 0),
    [getPayRateForEntry],
  );

  const handleSaveRef = useRef(handleSave);
  useEffect(() => {
    handleSaveRef.current = handleSave;
  }, [handleSave]);

  const bottomNavContent = useMemo(
    () => (
      <div className="flex items-center justify-between w-full gap-3">
        <div className="flex-1 min-w-0">
          {selectedProject ? (
            <p className="text-xs text-muted-foreground truncate">
              <span className="font-medium text-foreground">
                {selectedProject.name}
              </span>
              <span className="mx-1">·</span>
              {dateRange.startDate.slice(8, 10)}/
              {dateRange.startDate.slice(5, 7)}–{dateRange.endDate.slice(8, 10)}
              /{dateRange.endDate.slice(5, 7)}
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">Chưa chọn dự án</p>
          )}
        </div>
        <button
          onClick={() => handleSaveRef.current()}
          disabled={!canSave}
          className={cn(
            "flex min-h-11 items-center gap-2 rounded-full px-5 text-sm font-semibold transition-all shrink-0",
            canSave
              ? "bg-primary text-primary-foreground shadow-sm active:scale-95"
              : "bg-muted text-muted-foreground cursor-not-allowed",
          )}
        >
          {isLoading || isPreviewLoading ? (
            <div className="animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          {isPreviewLoading ? "Kiểm tra..." : "Lưu"}
        </button>
      </div>
    ),
    [selectedProject, dateRange, canSave, isLoading, isPreviewLoading],
  );

  const { setOverride } = useBottomNav();
  useEffect(() => {
    setOverride({ content: bottomNavContent });
    return () => setOverride(null);
  }, [setOverride, bottomNavContent]);

  // ─── filter drawer ────────────────────────────────────────────────────────

  const filterDrawer = filterOpen && (
    <div className="border-b bg-background shadow-sm animate-in slide-in-from-top-2 duration-200">
      <div className="px-4 pt-3 pb-4 space-y-3">
        <div className="space-y-1">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
            Dự án
          </p>
          <SearchableSelect
            value={formData.projectId ? String(formData.projectId) : "all"}
            onChange={handleProjectSelect}
            placeholder="Chọn dự án"
            searchPlaceholder="Tìm dự án..."
            options={[
              { value: "all", label: "Tất cả dự án" },
              ...projects.map((p) => ({
                value: String(p.id),
                label: `${p.name}${p.code ? ` · ${p.code}` : ""}`,
              })),
            ]}
          />
        </div>

        {availableEmployees.length > 0 && (
          <div className="space-y-1">
            <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
              Nhân viên
            </p>
            <SearchableSelect
              value={filterEmployeeId ? String(filterEmployeeId) : "all"}
              onChange={(v) =>
                setFilterEmployeeId(v === "all" ? null : Number(v))
              }
              placeholder="Tất cả nhân viên"
              searchPlaceholder="Tìm nhân viên..."
              options={[
                { value: "all", label: "Tất cả nhân viên" },
                ...availableEmployees.map((e) => ({
                  value: String(e.id),
                  label: e.fullname,
                })),
              ]}
            />
          </div>
        )}

        <div className="space-y-1">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
            Khoảng thời gian
          </p>
          <DateRangePicker
            variant="mobile"
            startDate={dateRange.startDate}
            endDate={dateRange.endDate}
            onStartDateChange={handleStartDate}
            onEndDateChange={handleEndDate}
            minDate={dateBounds.min}
            maxDate={dateBounds.max}
          />
        </div>

        {formData.projectId > 0 && !isPayRateReady && (
          <p className="text-xs text-amber-600 bg-amber-50 rounded-xl px-3 py-2">
            {isLoadingPayRate
              ? "Đang tải cấu hình lương..."
              : payRateForbidden
                ? "Bạn không có quyền truy cập bảng lương của dự án này."
                : "Dự án chưa có cấu hình lương."}
          </p>
        )}

        <button
          onClick={() => setFilterOpen(false)}
          className="flex min-h-11 w-full items-center justify-center gap-2 rounded-xl bg-primary text-sm font-semibold text-primary-foreground transition-opacity active:opacity-80"
        >
          <Check className="h-4 w-4" />
          Áp dụng
        </button>
      </div>
    </div>
  );

  // ─── main content ─────────────────────────────────────────────────────────

  const content = !formData.projectId ? (
    <div className="flex-1 flex flex-col items-center justify-center gap-4 px-8 text-center">
      <div className="w-16 h-16 rounded-2xl bg-muted/60 flex items-center justify-center">
        <SlidersHorizontal className="h-7 w-7 text-muted-foreground/60" />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">
          Chọn dự án để bắt đầu
        </p>
        <p className="text-xs text-muted-foreground">
          Mở bộ lọc để chọn dự án và khoảng thời gian
        </p>
      </div>
      <button
        onClick={() => setFilterOpen(true)}
        className="min-h-11 rounded-full bg-primary/10 px-4 text-sm font-medium text-primary transition-colors active:bg-primary/20"
      >
        Mở bộ lọc
      </button>
    </div>
  ) : !isPayRateReady ? (
    <div className="flex-1 flex items-center justify-center">
      <p className="text-sm text-muted-foreground">
        {isLoadingPayRate ? "Đang tải..." : payRateForbidden ? "Bạn không có quyền truy cập bảng lương của dự án này." : "Dự án chưa có cấu hình lương."}
      </p>
    </div>
  ) : visibleEmployees.length === 0 ? (
    <div className="flex-1 flex items-center justify-center">
      <p className="text-sm text-muted-foreground">Không có nhân viên nào</p>
    </div>
  ) : (
    <div className="flex-1 overflow-y-auto">
      <div className="divide-y divide-border/60">
        {visibleEmployees.map((emp) => {
          const empEntries = formData.entries.filter(
            (e) => e.employeeId === emp.id,
          );
          const activeDate = activeDates[emp.id] || dates[0];
          const activeEntry = empEntries.find((e) => e.date === activeDate);
          const globalIdx = activeEntry
            ? formData.entries.indexOf(activeEntry)
            : -1;
          const hourTypes =
            activeEntry?.position && activeEntry?.dayType
              ? getHourTypesForEntry(activeEntry.position, activeEntry.dayType)
              : [];
          const dayTypes = activeEntry?.position
            ? getDayTypesForPosition(activeEntry.position)
            : [];
          const earnings = calcEarnings(empEntries);
          const totalHours = empEntries.reduce(
            (s, e) => s + Object.values(e.hours).reduce((a, h) => a + h, 0),
            0,
          );
          const entryValidation = activeEntry
            ? validation.entryValidations[activeEntry.id]
            : undefined;
          const hasError = !!(
            entryValidation &&
            !entryValidation.valid &&
            entryValidation.errors.length > 0
          );

          const activeEntryTotalHours = activeEntry
            ? Object.values(activeEntry.hours).reduce((s, h) => s + h, 0)
            : 0;
          const activeHourStatus = getHourStatus(activeEntryTotalHours);
          const isMarkedForDeletion = !!(
            activeEntry?.originalValues &&
            (!activeEntry.hours ||
              Object.keys(activeEntry.hours).length === 0 ||
              Object.values(activeEntry.hours).every((v) => v === 0))
          );
          const activeRowErrors =
            activeEntry?.date
              ? getErrorsForRow(emp.id, activeEntry.date)
              : null;

          return (
            <div key={emp.id}>
              {/* Employee header */}
              <div className="flex items-center gap-3 px-4 py-2.5 bg-background/95 backdrop-blur-sm sticky top-0 z-[1] border-b border-border/40">
                <UserAvatar
                  name={emp.fullname}
                  size="sm"
                  className="shrink-0"
                />
                <span className="text-sm font-semibold flex-1 truncate">
                  {emp.fullname}
                </span>
                <div className="flex items-center gap-2 shrink-0">
                  {totalHours > 0 && (
                    <span className="text-xs text-muted-foreground tabular-nums">
                      {totalHours}h
                    </span>
                  )}
                  {earnings > 0 && (
                    <span
                      className={cn(
                        "flex items-center gap-1 text-xs font-semibold tabular-nums px-2 py-0.5 rounded-full",
                        earnings > 3500000
                          ? "bg-red-100 text-red-600"
                          : "bg-emerald-100 text-emerald-700",
                      )}
                    >
                      {earnings > 3500000 && (
                        <AlertTriangle className="h-3 w-3 shrink-0" />
                      )}
                      {formatCurrency(earnings)}
                    </span>
                  )}
                </div>
              </div>

              {/* Per-employee preview error banner */}
              {(() => {
                const empErrors = getErrorsForEmployee(emp.id);
                if (!empErrors) return null;
                const globalOnly = empErrors.filter(
                  (msg) =>
                    !empEntries.some(
                      (e) =>
                        e.date &&
                        getErrorsForRow(emp.id, e.date)?.includes(msg),
                    ),
                );
                if (globalOnly.length === 0) return null;
                return (
                  <div className="px-4 py-2 bg-red-50 border-b border-red-100">
                    <ErrorBanner errors={globalOnly} />
                  </div>
                );
              })()}

              {/* Date strip */}
              <DateStrip
                dates={dates}
                activeDate={activeDate}
                entries={empEntries}
                entryValidations={validation.entryValidations}
                getErrorsForRow={getErrorsForRow}
                employeeId={emp.id}
                onDateSelect={(date) =>
                  setActiveDates((p) => ({ ...p, [emp.id]: date }))
                }
              />

              {/* Active date inputs */}
              {activeEntry && (
                <div
                  className={cn(
                    "px-4 pt-3 pb-4 space-y-3",
                    isMarkedForDeletion
                      ? "bg-red-50/40"
                      : hasError
                        ? "bg-destructive/5"
                        : activeHourStatus === "excessive"
                          ? "bg-red-50/40"
                          : activeHourStatus === "exceeded"
                            ? "bg-amber-50/40"
                            : "",
                  )}
                >
                  {isMarkedForDeletion && (
                    <DeletionStateBanner
                      onUndo={() => handleUndoDelete(globalIdx, activeEntry)}
                    />
                  )}

                  {activeRowErrors && activeRowErrors.length > 0 && (
                    <ErrorBanner errors={activeRowErrors} />
                  )}

                  {hasError && entryValidation && (
                    <ErrorBanner errors={entryValidation.errors} />
                  )}

                  {activeHourStatus !== "normal" && !isMarkedForDeletion && (
                    <HoursWarningBadge
                      totalHours={activeEntryTotalHours}
                      status={activeHourStatus}
                    />
                  )}

                  {/* Day type + delete button row */}
                  <div className="flex items-center gap-2">
                    {/* Active date chip */}
                    <span className="text-xs font-semibold text-primary bg-primary/10 px-2 py-0.5 rounded-full shrink-0">
                      {WEEKDAY[new Date(activeDate).getDay()]} {activeDate.slice(8, 10)}/{activeDate.slice(5, 7)}
                    </span>
                    {dayTypes.length > 0 && (
                      <>
                        <Select
                          value={activeEntry.dayType || ""}
                          onValueChange={(v) =>
                            handleDayTypeChange(globalIdx, v)
                          }
                        >
                          <SelectTrigger className="h-11 w-auto min-w-[130px] rounded-lg border-0 bg-muted/40 px-2.5 text-xs shadow-none focus:ring-1">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {activeEntry.originalValues &&
                              activeEntryTotalHours > 0 && (
                                <div className="px-2 py-1.5 mb-1 border-b border-amber-100 bg-amber-50 flex items-start gap-1.5">
                                  <AlertTriangle className="h-3 w-3 text-amber-500 shrink-0 mt-0.5" />
                                  <p className="text-xs text-amber-700 leading-tight">
                                    Xóa giờ công trước khi đổi loại ngày
                                  </p>
                                </div>
                              )}
                            {dayTypes.map((dt) => (
                              <SelectItem
                                key={dt}
                                value={dt}
                                className="text-xs"
                              >
                                {dt}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </>
                    )}
                    <div className="flex-1" />
                    {activeEntry.originalValues && !isMarkedForDeletion && (
                      <button
                        onClick={() => handleDeleteEntry(globalIdx)}
                        className="flex min-h-11 items-center gap-1 rounded-xl px-2.5 text-xs font-medium text-muted-foreground transition-colors hover:text-destructive hover:bg-destructive/10"
                        aria-label="Xóa chấm công ngày này"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        Xóa
                      </button>
                    )}
                  </div>

                  {/* Hour inputs */}
                  {hourTypes.length > 0 && (
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-3">
                        {hourTypes.map((hourType) => {
                          const disabled = !hasPayRateForEntry(
                            activeEntry.position,
                            activeEntry.dayType || "",
                            hourType,
                          );
                          const val = activeEntry.hours?.[hourType] || 0;
                          const rawKey = `${activeEntry.id}-${hourType}`;
                          const displayValue = isMarkedForDeletion
                            ? ""
                            : rawInputs[rawKey] ?? (val > 0 ? String(val) : "");

                          return (
                            <HourInputField
                              key={hourType}
                              hourType={hourType}
                              value={val}
                              displayValue={displayValue}
                              disabled={disabled}
                              originalValue={
                                activeEntry.originalValues?.hours?.[hourType] ?? 0
                              }
                              hasOriginalValues={!!activeEntry.originalValues}
                              rate={0}
                              onFocus={() => {
                                if (val === 0)
                                  setRawInputs((p) => ({
                                    ...p,
                                    [rawKey]: "",
                                  }));
                              }}
                              onChange={(raw) =>
                                handleHourChange(
                                  globalIdx,
                                  activeEntry,
                                  hourType,
                                  raw,
                                )
                              }
                              onBlur={() =>
                                setTimeout(() => triggerPreview?.(), 0)
                              }
                            />
                          );
                        })}
                      </div>
                      {/* Consolidated earnings line */}
                      {(() => {
                        const rowEarnings = hourTypes.reduce((sum, hourType) => {
                          const disabled = !hasPayRateForEntry(activeEntry.position, activeEntry.dayType || "", hourType);
                          if (disabled) return sum;
                          const val = activeEntry.hours?.[hourType] || 0;
                          if (val <= 0) return sum;
                          const rate = getPayRateForEntry(activeEntry.position, activeEntry.dayType || "", hourType);
                          return sum + calculateTimesheetPreviewAmount(
                            rate,
                            val,
                            isFlexibleProject,
                          );
                        }, 0);
                        return rowEarnings > 0 ? (
                          <span className="text-xs text-emerald-600 font-semibold tabular-nums pl-0.5">
                            {formatCurrency(rowEarnings)}
                          </span>
                        ) : null;
                      })()}
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );

  // ─── render ───────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col h-full bg-background">
      {/* Top nav */}
      <div className="flex items-center gap-2 px-4 h-14 border-b bg-background shrink-0">
        <button
          onClick={handleClose}
          className="-ml-2 flex h-11 w-11 items-center justify-center rounded-xl transition-colors hover:bg-muted active:bg-muted/80"
          aria-label="Quay lại"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
        <h1 className="font-semibold text-base flex-1">Thêm chấm công</h1>
        <button
          onClick={() => setFilterOpen((o) => !o)}
          className={cn(
            "relative flex min-h-11 items-center gap-1.5 rounded-full px-3 text-sm font-medium transition-colors",
            filterOpen
              ? "bg-primary text-primary-foreground"
              : "bg-muted/70 text-foreground",
          )}
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          <span className="text-xs">Bộ lọc</span>
          {activeFilterCount > 0 && (
            <span className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-destructive text-white text-[11px] font-bold flex items-center justify-center">
              {activeFilterCount}
            </span>
          )}
        </button>
      </div>

      {/* Filter drawer */}
      {filterDrawer}

      {/* Global validation summary */}
      {getErrorsForEmployee(0) && (
        <div className="px-4 py-2 bg-red-50 border-b border-red-100">
          <ErrorBanner errors={getErrorsForEmployee(0)!} />
        </div>
      )}

      {content}
    </div>
  );
}
