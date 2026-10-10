import { useCallback, useRef, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { toast } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";
import {
  FileSpreadsheet,
  Loader2,
  Upload,
  X,
} from "lucide-react";
import { useBackfillEmployeeMobiles } from "@/hooks/api/useEmployees";
import type { MobileBackfillResult } from "@/types/api/employee.types";

const MAX_FILE_SIZE = 10 * 1024 * 1024; // 10MB
const MAX_VISIBLE_ERRORS = 20;

interface ImportMobilesModalProps {
  open: boolean;
  onClose: () => void;
  onCompleted?: () => void;
}

/**
 * Admin upload that backfills missing employee mobile numbers from the
 * "NV chưa có SĐT" export with the Mobile column filled in. Employees are
 * matched by CCCD; only blank mobile fields are written.
 */
export function ImportMobilesModal({
  open,
  onClose,
  onCompleted,
}: ImportMobilesModalProps) {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<MobileBackfillResult | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { mutateAsync: backfillMobiles, isPending } =
    useBackfillEmployeeMobiles();

  const resetState = useCallback(() => {
    setFile(null);
    setIsDragging(false);
    setProgress(0);
    setResult(null);
  }, []);

  const handleClose = useCallback(() => {
    if (isPending) return;
    resetState();
    onClose();
  }, [isPending, resetState, onClose]);

  const acceptFile = useCallback((candidate: File | undefined | null) => {
    if (!candidate) return;
    if (!candidate.name.toLowerCase().endsWith(".xlsx")) {
      toast({
        title: "Sai định dạng tệp",
        description: "Vui lòng tải lên tệp Excel (.xlsx).",
        variant: "destructive",
      });
      return;
    }
    if (candidate.size > MAX_FILE_SIZE) {
      toast({
        title: "Tệp quá lớn",
        description: "Kích thước tệp không được vượt quá 10MB.",
        variant: "destructive",
      });
      return;
    }
    setFile(candidate);
    setProgress(0);
  }, []);

  const handleDragOver = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (!isPending) setIsDragging(true);
    },
    [isPending]
  );

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragging(false);
      if (isPending) return;
      acceptFile(e.dataTransfer.files?.[0]);
    },
    [isPending, acceptFile]
  );

  const handleUpload = useCallback(async () => {
    if (!file || isPending) return;
    try {
      const data = await backfillMobiles({ file, onProgress: setProgress });
      setResult(data);
    } catch {
      // The global React Query mutation handler shows the error toast.
      setProgress(0);
    }
  }, [file, isPending, backfillMobiles]);

  const handleComplete = useCallback(() => {
    onCompleted?.();
    resetState();
    onClose();
  }, [onCompleted, resetState, onClose]);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md gap-0">
        <DialogHeader className="pb-1">
          <DialogTitle className="text-sm font-semibold">
            Nhập số điện thoại
          </DialogTitle>
          <DialogDescription className="text-xs">
            Tải lên tệp &ldquo;NV chưa có SĐT&rdquo; đã điền cột Mobile. Hệ
            thống khớp theo CCCD và chỉ điền ô còn trống.
          </DialogDescription>
        </DialogHeader>

        {!result ? (
          <>
            {/* Dropzone / file card */}
            {!file ? (
              <div
                role="button"
                tabIndex={isPending ? -1 : 0}
                aria-label="Chọn tệp Excel"
                onClick={() => !isPending && fileInputRef.current?.click()}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    if (!isPending) fileInputRef.current?.click();
                  }
                }}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                className={cn(
                  "flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-5 py-7 text-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                  isDragging
                    ? "border-primary bg-primary/5"
                    : "border-border hover:border-primary/40 hover:bg-muted/40"
                )}
              >
                <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <Upload className="h-5 w-5" />
                </span>
                <p className="text-sm font-medium text-foreground">
                  Kéo thả tệp vào đây
                </p>
                <p className="text-xs text-muted-foreground">
                  hoặc nhấn để chọn tệp · .xlsx · tối đa 10MB
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".xlsx"
                  onChange={(e) => {
                    acceptFile(e.target.files?.[0]);
                    e.target.value = "";
                  }}
                  disabled={isPending}
                  className="hidden"
                />
              </div>
            ) : (
              <div className="flex items-center gap-3 rounded-lg border border-border bg-card p-3">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <FileSpreadsheet className="h-4 w-4" />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground">
                    {file.name}
                  </p>
                  <p className="mt-0.5 text-xs tabular-nums text-muted-foreground">
                    {(file.size / 1024).toFixed(0)} KB
                  </p>
                </div>
                {!isPending && (
                  <button
                    onClick={() => setFile(null)}
                    title="Xóa tệp"
                    aria-label="Xóa tệp"
                    className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  >
                    <X className="h-4 w-4" />
                  </button>
                )}
              </div>
            )}

            {/* Upload progress */}
            {isPending && (
              <div className="mt-3">
                <Progress value={progress} className="h-1.5" />
                <p className="mt-1.5 flex items-center justify-end gap-1 text-xs font-medium text-muted-foreground">
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Đang tải lên…
                </p>
              </div>
            )}

            <div className="flex justify-end gap-2 mt-3">
              <button
                onClick={handleClose}
                disabled={isPending}
                className="h-8 px-3 rounded-md text-xs font-medium border border-border bg-card text-foreground hover:bg-muted transition-colors disabled:opacity-50"
              >
                Đóng
              </button>
              <button
                onClick={handleUpload}
                disabled={!file || isPending}
                className={cn(
                  "h-8 px-3 rounded-md text-xs font-medium flex items-center gap-1.5 transition-colors",
                  !file || isPending
                    ? "bg-primary/50 text-primary-foreground pointer-events-none"
                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                )}
              >
                {isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Upload className="h-3.5 w-3.5" />
                )}
                {isPending ? "Đang tải lên..." : "Tải lên"}
              </button>
            </div>
          </>
        ) : (
          <>
            <p className="px-1 pt-1 text-xs text-muted-foreground">
              Đã xử lý {result.total_rows} dòng
            </p>

            {/* Count tiles */}
            <div className="grid grid-cols-2 gap-1.5">
              <div className="rounded-lg border border-primary/30 bg-primary/5 px-2.5 py-2">
                <p className="text-xs font-medium text-primary">Đã cập nhật</p>
                <p className="mt-0.5 text-base font-bold tabular-nums text-foreground">
                  {result.updated}
                </p>
              </div>
              <div className="rounded-lg border border-border bg-muted/50 px-2.5 py-2">
                <p className="text-xs font-medium text-muted-foreground">
                  Đã có SĐT
                </p>
                <p className="mt-0.5 text-base font-bold tabular-nums text-foreground">
                  {result.skipped_existing}
                </p>
              </div>
              <div className="rounded-lg border border-warning/20 bg-warning/10 px-2.5 py-2">
                <p className="text-xs font-medium text-warning">
                  Không tìm thấy
                </p>
                <p className="mt-0.5 text-base font-bold tabular-nums text-foreground">
                  {result.not_found}
                </p>
              </div>
              <div className="rounded-lg border border-error/20 bg-error/10 px-2.5 py-2">
                <p className="text-xs font-medium text-error">Lỗi</p>
                <p className="mt-0.5 text-base font-bold tabular-nums text-foreground">
                  {result.error_count}
                </p>
              </div>
            </div>

            {/* Error rows */}
            {result.errors.length > 0 && (
              <div className="mt-2">
                <div className="max-h-44 overflow-y-auto rounded-lg border border-error/20 bg-error/5">
                  <ul className="divide-y divide-error/10">
                    {result.errors.slice(0, MAX_VISIBLE_ERRORS).map((error, index) => (
                      <li key={index} className="px-3 py-2 text-xs leading-relaxed">
                        <span className="font-medium text-error">
                          Dòng {error.row_number}
                          {error.cccd ? ` · ${error.cccd}` : ""}
                        </span>
                        <span className="text-muted-foreground">
                          {" "}
                          — {error.message}
                        </span>
                      </li>
                    ))}
                    {result.errors.length > MAX_VISIBLE_ERRORS && (
                      <li className="px-3 py-2 text-xs text-muted-foreground">
                        …và {result.errors.length - MAX_VISIBLE_ERRORS} lỗi khác
                      </li>
                    )}
                  </ul>
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2 mt-3">
              <button
                onClick={handleComplete}
                className="h-8 px-3 rounded-md text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                Hoàn tất
              </button>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
