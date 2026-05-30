import { ReactNode } from "react";
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
import { AlertTriangle, Trash2, AlertCircle, Loader2 } from "lucide-react";

interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm?: () => void;
  title: string;
  description: string | ReactNode;
  confirmText?: string;
  cancelText?: string;
  variant?: "default" | "destructive" | "warning";
  isLoading?: boolean;
  icon?: ReactNode;
}

export function ConfirmationModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmText = "Xác nhận",
  cancelText = "Hủy bỏ",
  variant = "default",
  isLoading = false,
  icon
}: ConfirmationModalProps) {
  const defaultIcons = {
    default: <AlertCircle className="h-6 w-6 text-blue-500" />,
    destructive: <Trash2 className="h-6 w-6 text-red-500" />,
    warning: <AlertTriangle className="h-6 w-6 text-orange-500" />
  };

  const buttonVariants = {
    default: "",
    destructive: "bg-red-600 hover:bg-red-700 focus:ring-red-600",
    warning: "bg-orange-600 hover:bg-orange-700 focus:ring-orange-600"
  };

  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent className="max-w-md">
        <AlertDialogHeader>
          <div className="flex items-center gap-3">
            {icon || defaultIcons[variant]}
            <AlertDialogTitle className="text-lg font-semibold">
              {title}
            </AlertDialogTitle>
          </div>
        </AlertDialogHeader>
        <div className="px-6 pt-4 pb-2">
          <AlertDialogDescription className="text-sm leading-relaxed">
            {description}
          </AlertDialogDescription>
        </div>
        <AlertDialogFooter className="!flex-row w-full gap-2">
          <AlertDialogCancel
            onClick={onClose}
            disabled={isLoading}
            className="flex-1 mt-0 h-8 px-3 text-sm"
          >
            {cancelText}
          </AlertDialogCancel>
          {onConfirm && confirmText && (
            <AlertDialogAction
              onClick={onConfirm}
              disabled={isLoading}
              className={`flex-1 h-8 px-3 text-sm ${buttonVariants[variant]}`}
            >
              {isLoading ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  Đang xử lý...
                </div>
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

export default ConfirmationModal;