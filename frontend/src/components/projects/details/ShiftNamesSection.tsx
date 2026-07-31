import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Check, Clock3, X, Moon } from "lucide-react";
import { useUpdateProject } from "@/hooks/api/useProjects";
import { useProjectPayRates } from "@/hooks/api/usePayRates";
import type { Project, ShiftName } from "@/types/api/project.types";
import { dateToString } from "@/utils/dateHelpers";

interface ShiftNamesSectionProps {
  project: Project;
  canEdit: boolean;
  onEditPayrate: (payrateId: number) => void;
}

// Matches the backend regex: ^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$
const SHIFT_RANGE_RE = /^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$/;

/**
 * extractShiftRanges walks the active payrate's rates structure
 * `{position: {dayType: {hourRange: rate}}}` and returns the distinct
 * "HH:MM-HH:MM" hourRange keys across every position/day-type, deduped and
 * sorted by start time. Mirrors the backend `ExtractShiftRanges` so the UI and
 * validation stay in sync.
 */
function extractShiftRanges(rates: unknown): string[] {
  if (!rates || typeof rates !== "object") return [];
  const seen = new Set<string>();
  const ranges: string[] = [];
  for (const position of Object.keys(rates as Record<string, unknown>)) {
    const dayTypes = (rates as Record<string, unknown>)[position];
    if (!dayTypes || typeof dayTypes !== "object") continue;
    for (const dayType of Object.keys(dayTypes as Record<string, unknown>)) {
      const hourRanges = (dayTypes as Record<string, unknown>)[dayType];
      if (!hourRanges || typeof hourRanges !== "object") continue;
      for (const key of Object.keys(hourRanges as Record<string, unknown>)) {
        if (SHIFT_RANGE_RE.test(key) && !seen.has(key)) {
          seen.add(key);
          ranges.push(key);
        }
      }
    }
  }
  return ranges.sort((a, b) => a.slice(0, 5).localeCompare(b.slice(0, 5)));
}

/** Overnight when end time-of-day <= start time-of-day (crosses midnight). */
function isOvernight(range: string): boolean {
  const [start, end] = range.split("-");
  return end <= start;
}

export function ShiftNamesSection({
  project,
  canEdit,
  onEditPayrate,
}: ShiftNamesSectionProps) {
  const updateMutation = useUpdateProject();
  const { data: payRatesData } = useProjectPayRates(project.id);

  // The payrate owns shift ranges. Prefer the active configuration and retain
  // the existing fallback for projects whose next configuration is upcoming.
  const displayedPayrate = useMemo(() => {
    const payrates = payRatesData?.data ?? [];
    if (payrates.length === 0) return null;
    // Active = fromDate <= today AND (no toDate OR toDate >= today).
    const today = dateToString(new Date());
    const active = payrates.find((p) => p.fromDate <= today && (!p.toDate || p.toDate >= today));
    if (active) return active;

    return payrates
      .filter((p) => p.fromDate > today)
      .sort((a, b) => a.fromDate.localeCompare(b.fromDate))[0] ?? null;
  }, [payRatesData]);
  const detectedRanges = useMemo(
    () => extractShiftRanges(displayedPayrate?.rates),
    [displayedPayrate]
  );

  // Local editable map: range -> name. Seeded from the project's saved names.
  const savedNames = useMemo(() => project.shift_names ?? [], [project.shift_names]);
  const [namesByRange, setNamesByRange] = useState<Record<string, string>>({});
  const [editing, setEditing] = useState(false);

  // Seed local state whenever the saved config or detected ranges change.
  useEffect(() => {
    const map: Record<string, string> = {};
    for (const r of detectedRanges) {
      map[r] = savedNames.find((s) => s.range === r)?.name ?? "";
    }
    setNamesByRange(map);
  }, [detectedRanges, savedNames]);

  const editActions = !editing && canEdit ? (
    <div className="flex w-full items-center justify-end gap-1 min-[420px]:w-auto">
      {displayedPayrate && (
        <Button
          variant="ghost"
          size="sm"
          className="min-h-11 px-3 text-xs text-muted-foreground hover:text-foreground gap-1.5"
          onClick={() => onEditPayrate(displayedPayrate.id)}
        >
          <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
          Chỉnh giờ &amp; lương
        </Button>
      )}
      {detectedRanges.length > 0 && (
        <Button
          variant="ghost"
          size="sm"
          className="min-h-11 px-3 text-xs text-muted-foreground hover:text-foreground gap-1.5"
          onClick={() => setEditing(true)}
        >
          Đặt tên ca
        </Button>
      )}
    </div>
  ) : null;

  // Nothing to name when the payrate has no shifts yet.
  if (detectedRanges.length === 0) {
    return (
      <div className="border-t pt-2 pb-4">
        <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2 sm:px-6">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
            Tên ca làm việc
          </p>
          {editActions}
        </div>
        <div className="px-4 sm:px-6">
          <p className="text-xs text-muted-foreground">
            Chưa có ca nào trong bảng lương. Chọn “Chỉnh giờ &amp; lương” để thêm khung giờ và mức lương theo giờ.
          </p>
        </div>
      </div>
    );
  }

  // Has the local draft diverged from the saved config?
  const isDirty = detectedRanges.some(
    (r) => (namesByRange[r] ?? "") !== (savedNames.find((s) => s.range === r)?.name ?? "")
  );

  const handleSave = () => {
    // Only persist ranges the payrate defines; skip rows with empty names so the
    // backend treats them as "not set" and employees see default labels.
    const payload: ShiftName[] = detectedRanges
      .map((range) => ({ range, name: (namesByRange[range] ?? "").trim() }))
      .filter((s) => s.name.length > 0);
    updateMutation.mutate(
      { id: project.id, data: { shift_names: payload } },
      { onSuccess: () => setEditing(false) }
    );
  };

  const handleCancel = () => {
    const map: Record<string, string> = {};
    for (const r of detectedRanges) {
      map[r] = savedNames.find((s) => s.range === r)?.name ?? "";
    }
    setNamesByRange(map);
    setEditing(false);
  };

  return (
    <div className="border-t pt-2 pb-4">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2 sm:px-6">
        <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
          Tên ca làm việc
        </p>
        {editActions}
      </div>

      <div className="px-4 sm:px-6 space-y-2">
        <p className="text-[11px] text-muted-foreground">
          Đặt tên hiển thị cho từng ca. Khung giờ và mức lương theo giờ được thay đổi trong bảng lương.
        </p>

        <div className="overflow-hidden rounded-lg border">
          <div className="hidden min-[420px]:grid min-[420px]:grid-cols-[180px_minmax(0,1fr)] gap-2 bg-muted/50 px-3 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
            <span>Khung giờ</span>
            <span>Tên hiển thị</span>
          </div>
          {detectedRanges.map((range) => {
            const overnight = isOvernight(range);
            const saved = savedNames.find((s) => s.range === range)?.name ?? "";
            return (
              <div
                key={range}
                className="grid grid-cols-1 gap-2 border-t px-3 py-2 text-xs min-[420px]:grid-cols-[180px_minmax(0,1fr)] min-[420px]:items-center"
              >
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className="font-mono text-[11px] text-muted-foreground">{range}</span>
                  {overnight && (
                    <span
                      className="inline-flex items-center gap-0.5 rounded bg-emerald-50 px-1 py-0.5 text-[11px] font-medium text-emerald-600 shrink-0"
                      title="Ca qua đêm"
                    >
                      <Moon className="h-2.5 w-2.5" />
                      Qua đêm
                    </span>
                  )}
                </div>
                {editing ? (
                  <Input
                    value={namesByRange[range] ?? ""}
                    onChange={(e) =>
                      setNamesByRange((m) => ({ ...m, [range]: e.target.value }))
                    }
                    placeholder={saved || "VD: Ca làm, Ca đêm…"}
                    maxLength={50}
                    className="h-11 text-xs"
                    onKeyDown={(e) => {
                      if (e.key === "Enter") handleSave();
                      if (e.key === "Escape") handleCancel();
                    }}
                  />
                ) : (
                  <span className={saved ? "text-foreground" : "text-muted-foreground italic"}>
                    {saved || "Chưa đặt tên"}
                  </span>
                )}
              </div>
            );
          })}
        </div>

        {editing && (
          <div className="flex flex-col-reverse gap-2 min-[380px]:flex-row min-[380px]:justify-end">
            <Button
              variant="ghost"
              size="sm"
              className="min-h-11 text-xs"
              onClick={handleCancel}
              disabled={updateMutation.isPending}
            >
              <X className="h-3 w-3" />
              Hủy
            </Button>
            <Button
              size="sm"
              className="min-h-11 text-xs"
              onClick={handleSave}
              disabled={!isDirty || updateMutation.isPending}
            >
              <Check className="h-3 w-3" />
              {updateMutation.isPending ? "Đang lưu…" : "Lưu"}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
