import { useState, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2, ScanFace, Search, AlertCircle } from "lucide-react";
import { useProjects } from "@/hooks/api/useProjects";
import { useProjectEmployees, useBulkToggleCheckInEnabled } from "@/hooks/api/useProjectEmployees";
import { projectEmployeesKey } from "@/lib/queryKeys";

interface CheckInBulkDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CheckInBulkDialog({ open, onOpenChange }: CheckInBulkDialogProps) {
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [search, setSearch] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const queryClient = useQueryClient();

  const { data: projectsData } = useProjects(
    { status: ["active"], pageSize: 200 },
    { enabled: open }
  );

  const employeeParams = useMemo(
    () => ({ pageSize: 200, status: "current" as const }),
    []
  );

  const { data: employeesData, isLoading: loadingEmployees } = useProjectEmployees(
    selectedProjectId ?? 0,
    employeeParams,
    open && !!selectedProjectId
  );

  const bulkToggle = useBulkToggleCheckInEnabled();

  const flexibleProjects = useMemo(
    () => (projectsData?.data ?? []).filter((p) => p.is_flexible),
    [projectsData]
  );

  const allEmployees = employeesData?.data ?? [];

  const filteredEmployees = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allEmployees;
    return allEmployees.filter(
      (e) =>
        e.employee_name.toLowerCase().includes(q) ||
        e.employee_cccd.toLowerCase().includes(q)
    );
  }, [allEmployees, search]);

  const handleProjectChange = (value: string) => {
    setSelectedProjectId(Number(value));
    setSelectedIds(new Set());
    setSearch("");
    setErrorMsg(null);
  };

  const toggle = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const allFilteredSelected =
    filteredEmployees.length > 0 &&
    filteredEmployees.every((e) => selectedIds.has(e.employee_id));

  const selectAllFiltered = () =>
    setSelectedIds((prev) => {
      const next = new Set(prev);
      filteredEmployees.forEach((e) => next.add(e.employee_id));
      return next;
    });

  const deselectAllFiltered = () =>
    setSelectedIds((prev) => {
      const next = new Set(prev);
      filteredEmployees.forEach((e) => next.delete(e.employee_id));
      return next;
    });

  const handleApply = (enabled: boolean) => {
    if (!selectedProjectId || selectedIds.size === 0) return;
    setErrorMsg(null);
    bulkToggle.mutate(
      { projectId: selectedProjectId, employeeIds: Array.from(selectedIds), enabled },
      {
        onSuccess: () => {
          setSelectedIds(new Set());
          // Explicitly invalidate the exact query key used by this dialog
          queryClient.invalidateQueries({
            queryKey: projectEmployeesKey(selectedProjectId, employeeParams),
          });
        },
        onError: (err: any) => {
          // Surface backend validation errors (e.g. no payrate) inside the dialog
          const msg =
            err?.response?.data?.message ||
            err?.message ||
            "Có lỗi xảy ra. Vui lòng thử lại.";
          setErrorMsg(msg);
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl w-full">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScanFace className="h-5 w-5" />
            Cấu hình điểm danh
          </DialogTitle>
          <DialogDescription>
            Chọn dự án và nhân viên để bật / tắt tính năng điểm danh.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 pt-1">
          {/* Top row: project selector + search */}
          <div className="flex items-end gap-3">
            <div className="flex-1 space-y-1.5">
              <label className="text-sm font-medium">Dự án linh động</label>
              <Select onValueChange={handleProjectChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Chọn dự án..." />
                </SelectTrigger>
                <SelectContent>
                  {flexibleProjects.length === 0 ? (
                    <SelectItem value="__none" disabled>
                      Không có dự án linh động
                    </SelectItem>
                  ) : (
                    flexibleProjects.map((p) => (
                      <SelectItem key={p.id} value={String(p.id)}>
                        {p.name}
                        {p.code && (
                          <span className="ml-1.5 text-muted-foreground text-xs">
                            ({p.code})
                          </span>
                        )}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
            </div>

            {selectedProjectId && (
              <div className="flex-1 space-y-1.5">
                <label className="text-sm font-medium">Tìm kiếm nhân viên</label>
                <div className="relative">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
                  <Input
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder="Tên hoặc CCCD..."
                    className="pl-9"
                  />
                </div>
              </div>
            )}
          </div>

          {/* Error banner */}
          {errorMsg && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{errorMsg}</AlertDescription>
            </Alert>
          )}

          {/* Employee grid */}
          {selectedProjectId && (
            <>
              {/* Toolbar */}
              {!loadingEmployees && (
                <div className="flex items-center justify-between">
                  <p className="text-sm text-muted-foreground">
                    {search
                      ? `${filteredEmployees.length} / ${allEmployees.length} nhân viên`
                      : `${allEmployees.length} nhân viên`}
                    {selectedIds.size > 0 && (
                      <span className="ml-2 font-medium text-primary">
                        · {selectedIds.size} đã chọn
                      </span>
                    )}
                  </p>
                  {filteredEmployees.length > 0 && (
                    <button
                      onClick={allFilteredSelected ? deselectAllFiltered : selectAllFiltered}
                      className="text-sm text-primary hover:underline"
                    >
                      {allFilteredSelected ? "Bỏ chọn tất cả" : "Chọn tất cả"}
                    </button>
                  )}
                </div>
              )}

              {/* Grid */}
              {loadingEmployees ? (
                <div className="flex justify-center py-16">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : filteredEmployees.length === 0 ? (
                <div className="rounded-lg border border-dashed py-12 text-center text-sm text-muted-foreground">
                  {search ? "Không tìm thấy nhân viên phù hợp" : "Dự án chưa có nhân viên"}
                </div>
              ) : (
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-80 overflow-y-auto pr-1">
                  {filteredEmployees.map((emp) => {
                    const checked = selectedIds.has(emp.employee_id);
                    return (
                      <div
                        key={emp.employee_id}
                        onClick={() => toggle(emp.employee_id)}
                        className={`
                          relative flex items-start gap-2.5 rounded-lg border p-3 cursor-pointer select-none transition-colors
                          ${checked
                            ? "border-primary bg-primary/5"
                            : "border-border hover:bg-muted/40"}
                        `}
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={() => toggle(emp.employee_id)}
                          onClick={(e) => e.stopPropagation()}
                          className="mt-0.5 shrink-0"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium truncate leading-tight">
                            {emp.employee_name}
                          </div>
                          <div className="text-xs text-muted-foreground mt-0.5">
                            {emp.employee_cccd}
                          </div>
                          <Badge
                            variant={emp.check_in_enabled ? "default" : "secondary"}
                            className="text-xs mt-1.5"
                          >
                            {emp.check_in_enabled ? "Đang bật" : "Đang tắt"}
                          </Badge>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}

              {/* Action buttons */}
              <div className="flex gap-2 pt-1 border-t">
                <Button
                  className="flex-1"
                  disabled={selectedIds.size === 0 || bulkToggle.isPending}
                  onClick={() => handleApply(true)}
                >
                  {bulkToggle.isPending && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                  Bật điểm danh
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  disabled={selectedIds.size === 0 || bulkToggle.isPending}
                  onClick={() => handleApply(false)}
                >
                  Tắt điểm danh
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
