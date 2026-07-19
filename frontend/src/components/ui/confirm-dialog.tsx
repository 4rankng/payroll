import { ReactNode, useState } from "react";
import { CircleHelp, Loader2, TriangleAlert } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string | ReactNode;
  confirmText?: string;
  cancelText?: string;
  onConfirm?: () => void | Promise<void>;
  onCancel?: () => void;
  confirmVariant?: "default" | "destructive";
  loading?: boolean;
  disabled?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmText = "Xác nhận",
  cancelText = "Hủy",
  onConfirm,
  onCancel,
  confirmVariant = "default",
  loading = false,
  disabled = false,
}: ConfirmDialogProps) {
  const [isProcessing, setIsProcessing] = useState(false);

  const handleConfirm = async (e: React.MouseEvent) => {
    if (!onConfirm) return;
    // Prevent Radix AlertDialogAction from auto-closing the dialog
    e.preventDefault();
    try {
      setIsProcessing(true);
      const result = onConfirm();

      if (result instanceof Promise) {
        await result;
      }
      onOpenChange(false);
    } catch {
      // Keep dialog open on error so user can retry
    } finally {
      setIsProcessing(false);
    }
  };

  const handleCancel = () => {
    if (onCancel) {
      onCancel();
    }
    onOpenChange(false);
  };

  const isLoading = loading || isProcessing;
  const isDisabled = disabled || isLoading;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-sm">
        <AlertDialogHeader className="pb-5">
          <div className="flex items-start gap-3.5">
            <span
              className={
                confirmVariant === "destructive"
                  ? "grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-error/10 text-error"
                  : "grid h-11 w-11 shrink-0 place-items-center rounded-xl bg-primary/10 text-primary"
              }
            >
              {confirmVariant === "destructive" ? (
                <TriangleAlert className="h-5 w-5" aria-hidden="true" />
              ) : (
                <CircleHelp className="h-5 w-5" aria-hidden="true" />
              )}
            </span>
            <div className="min-w-0 pt-0.5">
              <AlertDialogTitle>{title}</AlertDialogTitle>
              {typeof description === "string" && (
                <AlertDialogDescription>{description}</AlertDialogDescription>
              )}
            </div>
          </div>
        </AlertDialogHeader>

        {typeof description !== "string" && (
          <div className="px-5 pb-5 sm:px-6">{description}</div>
        )}

        <AlertDialogFooter className="w-full gap-2">
          <AlertDialogCancel
            onClick={handleCancel}
            disabled={isLoading}
            className="flex-1 mt-0"
          >
            {cancelText}
          </AlertDialogCancel>
          {onConfirm && confirmText && (
            <AlertDialogAction
              onClick={handleConfirm}
              disabled={isDisabled}
              variant={confirmVariant}
              className="flex-1"
            >
              {isLoading ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                  Đang xử lý...
                </>
              ) : (
                confirmText
              )}
            </AlertDialogAction>
          )}
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
