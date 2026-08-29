import { useState } from "react";
import { CheckCircle2, Circle, Eye, EyeOff } from "lucide-react";
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
import { useIsMobile } from '@/hooks/useBreakpoint';

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

// The special-character clause must match the BACKEND validator's enumerated
// set (internal/pkg/password/validator.go: "!@#$%^&*()_+-=[]{}|;:,.<>?").
// A broader rule such as [^A-Za-z0-9] lets characters like ~ pass the UI,
// then get rejected by the server after submit.
const SPECIAL_CHAR_REGEX = /[!@#$%^&*()_+\-=[\]{}|;:,.<>?]/;

/**
 * Live requirement checklist for the new-password field. Each rule mirrors
 * one clause of the backend validator so the ticks can never disagree with
 * what submit rejects — same pattern as ForceChangePasswordDialog.
 */
const PASSWORD_RULES: ReadonlyArray<{
  label: string;
  test: (value: string) => boolean;
}> = [
  { label: "8 ký tự trở lên", test: (value) => value.length >= 8 },
  { label: "Chữ hoa", test: (value) => /[A-Z]/.test(value) },
  { label: "Chữ thường", test: (value) => /[a-z]/.test(value) },
  { label: "Chữ số", test: (value) => /[0-9]/.test(value) },
  { label: "Ký tự đặc biệt", test: (value) => SPECIAL_CHAR_REGEX.test(value) },
  { label: "Tối đa 72 ký tự", test: (value) => value.length <= 72 },
];

// Same field/button styling as ForceChangePasswordDialog so the employee
// portal's two password surfaces stay visually identical.
const passwordFieldClass =
  "h-12 w-full rounded-xl border border-[var(--employee-border-strong)] bg-white px-3.5 pr-12 text-[0.9375rem] leading-normal text-[var(--employee-text)] transition-colors placeholder:text-[var(--employee-text-muted)] focus-visible:border-[var(--employee-accent)] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-[var(--employee-accent-ring)] disabled:cursor-not-allowed disabled:opacity-60";

const revealButtonClass =
  "absolute inset-y-1 right-1 inline-flex w-10 items-center justify-center rounded-lg text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-surface-muted)] hover:text-[var(--employee-text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:opacity-60";

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
  // Unmet rules stay neutral while typing; they only turn red after a submit
  // attempt (same pre-submit neutrality as ForceChangePasswordDialog), so the
  // checklist never reads as failure while the user is still composing.
  const [attemptedSubmit, setAttemptedSubmit] = useState(false);

  // Client-side guard so mismatched or incomplete input fails fast with
  // Vietnamese messages instead of a round-trip to the server validator.
  // The empty-confirm case only surfaces the message after a submit attempt —
  // silently doing nothing on tap reads as a broken button.
  const confirmMismatch =
    confirmPassword !== newPassword &&
    (confirmPassword.length > 0 || attemptedSubmit);
  const newPasswordFailedRule = PASSWORD_RULES.find(
    (rule) => !rule.test(newPassword),
  );

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
    setAttemptedSubmit(true);
    if (
      newPasswordFailedRule ||
      confirmPassword !== newPassword ||
      currentPassword.length === 0
    ) {
      return;
    }
    await onSubmit({ currentPassword, newPassword });
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setAttemptedSubmit(false);
  };

  const formContent = (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void handleSubmit();
      }}
      className="space-y-4"
    >
      {fieldValues.map((field) => (
        <div key={field.id}>
          <Label
            htmlFor={field.id}
            className="employee-type-label mb-1.5 block text-[var(--employee-text)]"
          >
            {field.label}
          </Label>
          <div className="relative">
            <Input
              id={field.id}
              type={field.show ? "text" : "password"}
              value={field.value}
              onChange={(e) => field.onChange(e.target.value)}
              autoComplete={
                field.id === "cp" ? "current-password" : "new-password"
              }
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
              placeholder={field.placeholder}
              className={passwordFieldClass}
              disabled={isPending}
            />
            <button
              type="button"
              className={revealButtonClass}
              onClick={field.toggle}
              disabled={isPending}
              aria-label={`${field.show ? "Ẩn" : "Hiện"} ${field.label.toLowerCase()}`}
            >
              {field.show ? (
                <EyeOff className="h-[1.125rem] w-[1.125rem]" />
              ) : (
                <Eye className="h-[1.125rem] w-[1.125rem]" />
              )}
            </button>
          </div>
          {field.id === "np" && (
            <ul className="mt-1.5 grid grid-cols-2 gap-x-3 gap-y-1.5">
              {PASSWORD_RULES.map((rule) => {
                const met = rule.test(newPassword);
                return (
                  <li
                    key={rule.label}
                    className={`employee-type-body-sm flex items-center gap-1.5 ${
                      met
                        ? "text-[var(--employee-accent)]"
                        : attemptedSubmit
                          ? "text-[var(--employee-error)]"
                          : "text-[var(--employee-text-secondary)]"
                    }`}
                  >
                    {met ? (
                      <CheckCircle2
                        className="h-4 w-4 shrink-0"
                        aria-hidden="true"
                      />
                    ) : (
                      <Circle
                        className="h-4 w-4 shrink-0 opacity-60"
                        aria-hidden="true"
                      />
                    )}
                    <span className="min-w-0 truncate">{rule.label}</span>
                  </li>
                );
              })}
            </ul>
          )}
          {field.id === "cfp" && confirmMismatch && (
            <p className="employee-type-body-sm mt-1.5 text-[var(--employee-error)]">
              Xác nhận mật khẩu chưa khớp.
            </p>
          )}
        </div>
      ))}
      <button
        type="submit"
        disabled={isPending}
        className="employee-type-action inline-flex h-12 w-full items-center justify-center rounded-xl bg-[var(--employee-accent)] px-4 text-white shadow-[var(--employee-cta-shadow)] transition-colors hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-60"
      >
        {isPending ? "Đang đổi..." : "Đổi mật khẩu"}
      </button>
    </form>
  );

  if (isMobile) {
    return (
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent
          side="bottom"
          className="h-auto max-h-[85vh] rounded-t-3xl px-5"
        >
          <SheetHeader className="border-b border-[var(--employee-border)] pb-4">
            <SheetTitle className="employee-type-card-title text-[var(--employee-text)]">
              Đổi mật khẩu
            </SheetTitle>
            <p className="employee-type-body-sm text-[var(--employee-text-secondary)]">
              Cập nhật mật khẩu để giữ tài khoản an toàn.
            </p>
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
          <DialogTitle className="employee-type-card-title text-white">
            Đổi mật khẩu
          </DialogTitle>
        </DialogHeader>
        {formContent}
      </DialogContent>
    </Dialog>
  );
}
