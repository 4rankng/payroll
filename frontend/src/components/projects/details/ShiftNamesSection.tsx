import { useEffect, useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Check, X, Moon } from "lucide-react";
import { useUpdateProject } from "@/hooks/api/useProjects";
import { useProjectPayRates } from "@/hooks/api/usePayRates";
import type { Project, ShiftName } from "@/types/api/project.types";

interface ShiftNamesSectionProps {
  project: Project;
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

export function ShiftNamesSection({ project }: ShiftNamesSectionProps) {
  const updateMutation = useUpdateProject();
  const { data: payRatesData } = useProjectPayRates(project.id);

  // Detect configured shift ranges from the active payrate.
  const detectedRanges = useMemo(() => {
    const payrates = payRatesData?.data ?? [];
    if (payrates.length === 0) return [];
    // Active = fromDate <= today AND (no toDate OR toDate >= today).
    const today = new Date().toISOString().slice(0, 10);
    const active =
      payrates.find((p) => p.fromDate <= today && (!p.toDate || p.toDate >= today)) ?? payrates[0];
    return extractShiftRanges(active?.rates);
  }, [payRatesData]);

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

  // Nothing to name when the payrate has no shifts yet.
  if (detectedRanges.length === 0) {
    return (
      <div className="border-t pt-2 pb-4">
        <div className="px-4 sm:px-6 py-2">
          <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
            Tên ca làm việc
          </p>
        </div>
        <div className="px-4 sm:px-6">
          <p className="text-xs text-muted-foreground">
            Chưa có ca nào trong bảng lương. Thêm khung giờ vào bảng lương để đặt tên ca.
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
      <div className="flex items-center justify-between px-4 sm:px-6 py-2">
        <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-widest">
          Tên ca làm việc
        </p>
        {!editing && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
            onClick={() => setEditing(true)}
          >
            Đặt tên ca
          </Button>
        )}
      </div>

      <div className="px-4 sm:px-6 space-y-2">
        <p className="text-[11px] text-muted-foreground">
          Đặt tên hiển thị cho từng ca trong bảng lương. Tên này sẽ hiện trên thẻ chấm công của nhân viên.
        </p>

        <div className="rounded-lg border overflow-hidden">
          <div className="grid grid-cols-[120px_1fr] gap-2 bg-muted/50 px-3 py-1.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
            <span>Khung giờ</span>
            <span>Tên hiển thị</span>
          </div>
          {detectedRanges.map((range) => {
            const overnight = isOvernight(range);
            const saved = savedNames.find((s) => s.range === range)?.name ?? "";
            return (
              <div
                key={range}
                className="grid grid-cols-[120px_1fr] gap-2 px-3 py-2 text-xs border-t items-center"
              >
                <div className="flex items-center gap-1.5 min-w-0">
                  <span className="font-mono text-[11px] text-muted-foreground truncate">{range}</span>
                  {overnight && (
                    <span
                      className="inline-flex items-center gap-0.5 rounded bg-indigo-50 px-1 py-0.5 text-[11px] font-medium text-indigo-600 shrink-0"
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
                    className="h-7 text-xs"
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
          <div className="flex justify-end gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              onClick={handleCancel}
              disabled={updateMutation.isPending}
            >
              <X className="h-3 w-3" />
              Hủy
            </Button>
            <Button
              size="sm"
              className="h-7 text-xs"
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
