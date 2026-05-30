import { useState, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { Download, FolderOpen, Check } from "lucide-react";

interface ProjectOption {
  id: number;
  name: string;
  code?: string;
}

interface ExportEmployeesModalProps {
  open: boolean;
  onClose: () => void;
  onExport: (projectIds: number[]) => void;
  projects: ProjectOption[];
  isExporting?: boolean;
}

export function ExportEmployeesModal({
  open,
  onClose,
  onExport,
  projects,
  isExporting = false,
}: ExportEmployeesModalProps) {
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());

  const toggle = useCallback((id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const selectAll = useCallback(() => {
    setSelectedIds(new Set(projects.map((p) => p.id)));
  }, [projects]);

  const deselectAll = useCallback(() => setSelectedIds(new Set()), []);

  const isAllSelected = projects.length > 0 && selectedIds.size === projects.length;
  const hasSelection = selectedIds.size > 0;

  const handleClose = useCallback(() => {
    setSelectedIds(new Set());
    onClose();
  }, [onClose]);

  const handleExport = useCallback(() => {
    onExport(Array.from(selectedIds));
    handleClose();
  }, [onExport, selectedIds, handleClose]);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md gap-0">
        <DialogHeader className="pb-1">
          <DialogTitle className="text-sm font-semibold">Xuất Excel nhân viên</DialogTitle>
          <DialogDescription className="text-xs">
            Chọn một hoặc nhiều dự án để xuất
          </DialogDescription>
        </DialogHeader>

        {/* Select all / clear */}
        <div className="flex items-center justify-between px-1 py-1.5">
          <span className="text-[10px] text-muted-foreground">
            Đã chọn {selectedIds.size} dự án
          </span>
          <button
            onClick={isAllSelected ? deselectAll : selectAll}
            disabled={isExporting}
            className="text-[10px] font-medium text-primary hover:underline"
          >
            {isAllSelected ? "Bỏ chọn tất cả" : "Chọn tất cả"}
          </button>
        </div>

        {/* Project grid */}
        <div className="max-h-[260px] overflow-y-auto">
          <div className="grid grid-cols-2 gap-1.5 p-1">
            {projects.map((project) => {
              const isSelected = selectedIds.has(project.id);
              return (
                <button
                  key={project.id}
                  onClick={() => toggle(project.id)}
                  disabled={isExporting}
                  className={cn(
                    "relative flex flex-col gap-0.5 px-2.5 py-2 rounded-lg border transition-colors text-left",
                    isSelected
                      ? "border-primary/30 bg-primary/5"
                      : "border-border hover:bg-muted/50"
                  )}
                >
                  {isSelected && (
                    <Check className="absolute top-1.5 right-1.5 h-3 w-3 text-primary" />
                  )}
                  <div className={cn(
                    "flex items-center gap-1.5",
                    isSelected ? "text-primary" : "text-foreground"
                  )}>
                    <FolderOpen className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <span className="text-xs font-medium truncate">{project.name}</span>
                  </div>
                  {project.code && (
                    <span className="text-[10px] text-muted-foreground truncate pl-5">
                      {project.code}
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-3">
          <button
            onClick={handleClose}
            disabled={isExporting}
            className="h-8 px-3 rounded-md text-xs font-medium border border-border bg-background text-foreground hover:bg-muted transition-colors"
          >
            Đóng
          </button>
          <button
            onClick={handleExport}
            disabled={isExporting || !hasSelection}
            className={cn(
              "h-8 px-3 rounded-md text-xs font-medium flex items-center gap-1.5 transition-colors",
              isExporting || !hasSelection
                ? "bg-primary/50 text-primary-foreground pointer-events-none"
                : "bg-primary text-primary-foreground hover:bg-primary/90"
            )}
          >
            <Download className="h-3.5 w-3.5" />
            {isExporting ? "Đang xuất..." : "Xuất Excel"}
          </button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
