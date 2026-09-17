import { useState, useCallback, useEffect, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Trash2, Plus, MapPin, Pencil, Check, X } from "lucide-react";
import { useUpdateProject } from "@/hooks/api/useProjects";
import type { Project, GeofenceGate } from "@/types/api/project.types";

interface GeofenceSectionProps {
  project: Project;
}

interface EditingGate {
  name: string;
  lat: string;
  lng: string;
}

export function GeofenceSection({ project }: GeofenceSectionProps) {
  const updateMutation = useUpdateProject();
  const gates = useMemo(() => project.geofence_gates ?? [], [project.geofence_gates]);
  const radius = project.geofence_radius_meters ?? 100;
  const [adding, setAdding] = useState(false);
  const [newGate, setNewGate] = useState<EditingGate>({ name: "", lat: "", lng: "" });
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editValues, setEditValues] = useState<EditingGate>({ name: "", lat: "", lng: "" });
  const [radiusValue, setRadiusValue] = useState(String(radius));

  useEffect(() => {
    setRadiusValue(String(radius));
  }, [radius]);

  const parsedRadius = parseInt(radiusValue, 10);
  const isRadiusValid = !isNaN(parsedRadius) && parsedRadius >= 10 && parsedRadius <= 1000;
  const isRadiusChanged = radiusValue.trim() !== String(radius);
  const canSaveRadius = isRadiusChanged && isRadiusValid && parsedRadius !== radius;

  const saveNewGate = useCallback(() => {
    const lat = parseFloat(newGate.lat);
    const lng = parseFloat(newGate.lng);
    if (!newGate.name.trim() || isNaN(lat) || isNaN(lng)) return;
    const gate: GeofenceGate = { name: newGate.name.trim(), lat, lng };
    const updated = [...gates, gate];
    updateMutation.mutate(
      { id: project.id, data: { geofence_gates: updated } },
      {
        onSuccess: () => {
          setAdding(false);
          setNewGate({ name: "", lat: "", lng: "" });
        },
      }
    );
  }, [newGate, gates, project.id, updateMutation]);

  const startEditing = useCallback((index: number) => {
    const g = gates[index];
    setEditingIndex(index);
    setEditValues({ name: g.name, lat: String(g.lat), lng: String(g.lng) });
    setAdding(false);
  }, [gates]);

  const saveEdit = useCallback(() => {
    if (editingIndex === null) return;
    const lat = parseFloat(editValues.lat);
    const lng = parseFloat(editValues.lng);
    if (!editValues.name.trim() || isNaN(lat) || isNaN(lng)) return;
    const updated = gates.map((g, i) =>
      i === editingIndex ? { name: editValues.name.trim(), lat, lng } : g
    );
    updateMutation.mutate(
      { id: project.id, data: { geofence_gates: updated } },
      { onSuccess: () => setEditingIndex(null) }
    );
  }, [editingIndex, editValues, gates, project.id, updateMutation]);

  const cancelEdit = useCallback(() => {
    setEditingIndex(null);
    setEditValues({ name: "", lat: "", lng: "" });
  }, []);

  const removeGate = useCallback(
    (index: number) => {
      const updated = gates.filter((_, i) => i !== index);
      updateMutation.mutate({ id: project.id, data: { geofence_gates: updated } });
    },
    [gates, project.id, updateMutation]
  );

  const updateRadius = useCallback(
    () => {
      if (!canSaveRadius) return;
      updateMutation.mutate({ id: project.id, data: { geofence_radius_meters: parsedRadius } });
    },
    [canSaveRadius, parsedRadius, project.id, updateMutation]
  );

  const cancelRadiusEdit = useCallback(() => {
    setRadiusValue(String(radius));
  }, [radius]);

  return (
    <div className="border-t pt-2 pb-4">
      <div className="flex items-center justify-between px-4 sm:px-6 py-2">
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
          Vị trí check-in
        </p>
        {!adding && editingIndex === null && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
            onClick={() => setAdding(true)}
          >
            <Plus className="h-3 w-3" />
            Thêm cổng
          </Button>
        )}
      </div>

      <div className="px-4 sm:px-6 space-y-3">
        {/* Radius */}
        <div className="rounded-lg border bg-muted/20 px-3 py-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="min-w-0">
              <p className="text-xs font-medium">Bán kính chấm công</p>
              <p className="text-xs text-muted-foreground">Cho phép từ 10 đến 1000 mét.</p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Input
                type="number"
                min={10}
                max={1000}
                value={radiusValue}
                onChange={(e) => setRadiusValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") updateRadius();
                  if (e.key === "Escape") cancelRadiusEdit();
                }}
                className="h-8 w-24 text-xs tabular-nums"
              />
              <span className="text-xs text-muted-foreground">mét</span>
              {isRadiusChanged && (
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
                    disabled={updateMutation.isPending}
                    onClick={cancelRadiusEdit}
                  >
                    <X className="h-3 w-3" />
                    Hủy
                  </Button>
                  <Button
                    size="sm"
                    className="h-8 px-2 text-xs"
                    disabled={!canSaveRadius || updateMutation.isPending}
                    onClick={updateRadius}
                  >
                    <Check className="h-3 w-3" />
                    {updateMutation.isPending ? "Đang lưu..." : "Lưu"}
                  </Button>
                </div>
              )}
            </div>
          </div>
          {!isRadiusValid && (
            <p className="mt-1 text-xs text-destructive">
              Bán kính phải nằm trong khoảng 10-1000 mét.
            </p>
          )}
        </div>

        {/* Gate list */}
        {gates.length === 0 && !adding ? (
          <div className="flex flex-col items-center justify-center py-6 text-center">
            <MapPin className="h-8 w-8 text-muted-foreground/40 mb-2" />
            <p className="text-xs text-muted-foreground">
              Chưa có cổng check-in. Thêm cổng mới để bắt đầu.
            </p>
          </div>
        ) : (
          <div className="rounded-lg border overflow-hidden">
            {/* Header */}
            <div className="grid grid-cols-[1fr_100px_100px_72px] gap-2 bg-muted/50 px-3 py-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              <span>Tên cổng</span>
              <span>Vĩ độ</span>
              <span>Kinh độ</span>
              <span />
            </div>
            {/* Existing gates */}
            {gates.map((gate, i) =>
              editingIndex === i ? (
                /* ── Edit mode row ── */
                <div
                  key={i}
                  className="grid grid-cols-[1fr_100px_100px_72px] gap-2 px-3 py-2 text-xs border-t items-center bg-primary/5"
                >
                  <Input
                    value={editValues.name}
                    onChange={(e) => setEditValues((v) => ({ ...v, name: e.target.value }))}
                    className="h-6 text-xs"
                    autoFocus
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(); if (e.key === "Escape") cancelEdit(); }}
                  />
                  <Input
                    value={editValues.lat}
                    onChange={(e) => setEditValues((v) => ({ ...v, lat: e.target.value }))}
                    className="h-6 text-xs font-mono"
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(); if (e.key === "Escape") cancelEdit(); }}
                  />
                  <Input
                    value={editValues.lng}
                    onChange={(e) => setEditValues((v) => ({ ...v, lng: e.target.value }))}
                    className="h-6 text-xs font-mono"
                    onKeyDown={(e) => { if (e.key === "Enter") saveEdit(); if (e.key === "Escape") cancelEdit(); }}
                  />
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 shrink-0 text-green-600 hover:text-green-700 hover:bg-green-50"
                      disabled={
                        !editValues.name.trim() ||
                        isNaN(parseFloat(editValues.lat)) ||
                        isNaN(parseFloat(editValues.lng)) ||
                        updateMutation.isPending
                      }
                      onClick={saveEdit}
                    >
                      <Check className="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 shrink-0 text-muted-foreground hover:text-foreground"
                      onClick={cancelEdit}
                    >
                      <X className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              ) : (
                /* ── Read mode row ── */
                <div
                  key={i}
                  className="grid grid-cols-[1fr_100px_100px_72px] gap-2 px-3 py-2 text-xs border-t items-center group"
                >
                  <span className="truncate">{gate.name}</span>
                  <span className="text-muted-foreground font-mono text-xs">{gate.lat.toFixed(6)}</span>
                  <span className="text-muted-foreground font-mono text-xs">{gate.lng.toFixed(6)}</span>
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 shrink-0 text-muted-foreground hover:text-foreground opacity-0 group-hover:opacity-100 transition-opacity"
                      onClick={() => startEditing(i)}
                    >
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 shrink-0 text-muted-foreground hover:text-destructive"
                      onClick={() => removeGate(i)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              )
            )}
            {/* Add row */}
            {adding && (
              <div className="grid grid-cols-[1fr_100px_100px_72px] gap-2 px-3 py-2 text-xs border-t items-center bg-primary/5">
                <Input
                  placeholder="Tên cổng"
                  value={newGate.name}
                  onChange={(e) => setNewGate((g) => ({ ...g, name: e.target.value }))}
                  className="h-6 text-xs"
                  autoFocus
                />
                <Input
                  placeholder="10.762622"
                  value={newGate.lat}
                  onChange={(e) => setNewGate((g) => ({ ...g, lat: e.target.value }))}
                  className="h-6 text-xs font-mono"
                />
                <Input
                  placeholder="106.660172"
                  value={newGate.lng}
                  onChange={(e) => setNewGate((g) => ({ ...g, lng: e.target.value }))}
                  className="h-6 text-xs font-mono"
                />
                <div />
              </div>
            )}
          </div>
        )}

        {/* Add gate action buttons */}
        {adding && (
          <div className="flex justify-end gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs"
              onClick={() => {
                setAdding(false);
                setNewGate({ name: "", lat: "", lng: "" });
              }}
            >
              Hủy
            </Button>
            <Button
              size="sm"
              className="h-7 text-xs"
              disabled={
                !newGate.name.trim() ||
                isNaN(parseFloat(newGate.lat)) ||
                isNaN(parseFloat(newGate.lng)) ||
                updateMutation.isPending
              }
              onClick={saveNewGate}
            >
              {updateMutation.isPending ? "Đang lưu..." : "Lưu"}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
