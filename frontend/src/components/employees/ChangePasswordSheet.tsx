import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";
import { useIsMobile } from "@/hooks/use-mobile";

interface ChangePasswordSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: {
    currentPassword: string;
    newPassword: string;
  }) => Promise<void>;
  isPending: boolean;
}

const PASSWORD_FIELDS = [
  {
    id: "cp",
    label: "Mật khẩu hiện tại",
    placeholder: "Nhập mật khẩu hiện tại",
  },
  { id: "np", label: "Mật khẩu mới", placeholder: "Tối thiểu 8 ký tự" },
  {
    id: "cfp",
    label: "Xác nhận mật khẩu mới",
    placeholder: "Nhập lại mật khẩu mới",
  },
] as const;

export function ChangePasswordSheet({
  open,
  onOpenChange,
  onSubmit,
  isPending,
}: ChangePasswordSheetProps) {
  const isMobile = useIsMobile();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const fieldValues = [
    {
      ...PASSWORD_FIELDS[0],
      value: currentPassword,
      onChange: setCurrentPassword,
      show: showCurrent,
      toggle: () => setShowCurrent((v) => !v),
    },
    {
      ...PASSWORD_FIELDS[1],
      value: newPassword,
      onChange: setNewPassword,
      show: showNew,
      toggle: () => setShowNew((v) => !v),
    },
    {
      ...PASSWORD_FIELDS[2],
      value: confirmPassword,
      onChange: setConfirmPassword,
      show: showConfirm,
      toggle: () => setShowConfirm((v) => !v),
    },
  ];

  const handleSubmit = async () => {
    await onSubmit({ currentPassword, newPassword });
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
  };

  const formContent = (
    <div className="space-y-4">
      {fieldValues.map((field) => (
        <div key={field.id}>
          <Label
            htmlFor={field.id}
            className="text-sm font-semibold mb-1.5 block text-gray-700"
          >
            {field.label}
          </Label>
          <div className="relative">
            <Input
              id={field.id}
              type={field.show ? "text" : "password"}
              value={field.value}
              onChange={(e) => field.onChange(e.target.value)}
              className="h-11 pr-12 text-sm rounded-xl border-gray-200 focus:border-[#00B14F] focus:ring-[#00B14F]/20"
              placeholder={field.placeholder}
            />
            <button
              type="button"
              className="absolute right-0 top-0 h-11 w-11 flex items-center justify-center text-gray-400 hover:text-gray-600"
              onClick={field.toggle}
              aria-label={field.show ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
            >
              {field.show ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>
      ))}
      <div className="flex justify-end pt-1">
        <button
          onClick={handleSubmit}
          disabled={isPending}
          className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl text-sm font-semibold text-white disabled:opacity-50 transition-all active:scale-[0.97]"
          style={{ background: EMPLOYEE_BRAND_COLOR }}
        >
          {isPending ? "Đang đổi..." : "Đổi mật khẩu"}
        </button>
      </div>
    </div>
  );

  if (isMobile) {
    return (
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent
          side="bottom"
          className="h-auto max-h-[85vh] rounded-t-3xl px-5"
        >
          <SheetHeader className="pb-4 border-b border-gray-100">
            <SheetTitle className="text-base font-bold text-gray-900">
              Đổi mật khẩu
            </SheetTitle>
          </SheetHeader>
          <div className="mt-4 overflow-y-auto" style={{ paddingBottom: "calc(env(safe-area-inset-bottom) + 16px)" }}>
            {formContent}
          </div>
        </SheetContent>
      </Sheet>
    );
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm rounded-2xl">
        <DialogHeader>
          <DialogTitle className="text-base font-bold text-gray-900">
            Đổi mật khẩu
          </DialogTitle>
        </DialogHeader>
        {formContent}
      </DialogContent>
    </Dialog>
  );
}
