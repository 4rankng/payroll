import { useState, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useIsMobile } from '@/hooks/useBreakpoint';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { BankSelector } from "@/components/ui/bank-selector";
import { toast } from "@/components/ui/sonner";
import { ErrorState } from "@/components/ui/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { employeeService } from "@/services/api/employee.service";
import { formatVietnameseName } from "@/lib/validation";
import { cn } from "@/lib/utils";
import {
  Eye,
  EyeOff,
  User,
  CreditCard,
  Building2,
  Lock,
  CheckCircle2,
  X,
} from "lucide-react";
import type { Employee, CurrentProject } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";
import { QueryKeys } from "@/lib/queryKeys";

// ─── Props ────────────────────────────────────────────────────────────────────

interface EditAdvPartnerUserSheetProps {
  employeeId: number;
  fullname: string;
  username?: string;
  onClose: () => void;
}

// ─── Form data ────────────────────────────────────────────────────────────────

interface FormData {
  fullname: string;
  email: string;
  cccd: string;
  mobile: string;
  bank_account_number: string;
  bank_account_name: string;
}

// ─── Section wrapper ──────────────────────────────────────────────────────────

function Section({
  icon: Icon,
  title,
  children,
}: {
  icon: React.ElementType;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="h-6 w-6 rounded-md bg-muted flex items-center justify-center shrink-0">
          <Icon className="w-3.5 h-3.5 text-muted-foreground" />
        </div>
        <span className="text-xs font-semibold text-foreground uppercase tracking-wider">
          {title}
        </span>
      </div>
      <div className="rounded-xl border border-border/60 bg-muted/20 p-4 space-y-4">
        {children}
      </div>
    </div>
  );
}

function Field({
  label,
  id,
  error,
  children,
}: {
  label: string;
  id?: string;
  error?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-[12px] font-medium text-muted-foreground">
        {label}
      </Label>
      {children}
      {error && <p className="text-[11px] text-destructive">{error}</p>}
    </div>
  );
}

// ─── Avatar ───────────────────────────────────────────────────────────────────

function Avatar({ name, username }: { name: string; username?: string }) {
  const initials = name
    .split(" ")
    .filter(Boolean)
    .slice(-2)
    .map((w) => w[0])
    .join("")
    .toUpperCase();
  const colors = [
    "from-teal-500 to-cyan-600",
    "from-sky-500 to-blue-600",
    "from-emerald-500 to-green-600",
    "from-amber-500 to-orange-600",
    "from-rose-500 to-red-600",
    "from-lime-500 to-emerald-600",
  ];
  const gradient = colors[name.charCodeAt(0) % colors.length];
  return (
    <div className="flex items-center gap-3">
      <div
        className={cn(
          "w-11 h-11 rounded-2xl flex items-center justify-center text-white text-[15px] font-bold shrink-0 bg-gradient-to-br",
          gradient
        )}
      >
        {initials || "?"}
      </div>
      <div className="min-w-0">
        <div className="break-words font-bold text-[15px] leading-tight">{name}</div>
        {username && (
          <div className="break-all font-mono text-[12px] text-muted-foreground">
            @{username}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Project pill ─────────────────────────────────────────────────────────────

function ProjectPill({ project }: { project: CurrentProject }) {
  return (
    <div className="inline-flex min-w-0 flex-wrap items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-[12px]">
      <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
      <span className="font-medium text-emerald-700 dark:text-emerald-400">
        {project.name}
      </span>
      {project.code && (
        <span className="text-emerald-600/60 dark:text-emerald-500/60 font-mono">
          {project.code}
        </span>
      )}
    </div>
  );
}

// ─── Sheet ────────────────────────────────────────────────────────────────────

export default function EditAdvPartnerUserSheet({
  employeeId,
  fullname: initialName,
  username: initialUsername,
  onClose,
}: EditAdvPartnerUserSheetProps) {
  const isMobile = useIsMobile();

  const [form, setForm] = useState<FormData>({
    fullname: initialName || "",
    email: "",
    cccd: "",
    mobile: "",
    bank_account_number: "",
    bank_account_name: "",
  });
  const [selectedBank, setSelectedBank] = useState<Bank | null>(null);
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [isSaving, setIsSaving] = useState(false);
  const [savedInfo, setSavedInfo] = useState(false);
  const [savedPassword, setSavedPassword] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [loadAttempt, setLoadAttempt] = useState(0);
  const [projects, setProjects] = useState<CurrentProject[]>([]);
  const [currentUsername, setCurrentUsername] = useState(initialUsername);
  const queryClient = useQueryClient();

  // Fetch full employee data when sheet opens
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setLoadError(false);
    employeeService.getEmployeeById(employeeId).then((res) => {
      if (cancelled) return;
      const emp = res;
      setForm({
        fullname: emp.fullname || "",
        email: (emp.email as string) || "",
        cccd: emp.cccd || "",
        mobile: emp.mobile || "",
        bank_account_number: emp.bank_account_number || "",
        bank_account_name: emp.bank_account_name || "",
      });
      setSelectedBank(emp.bank ?? null);
      setProjects(emp.current_projects ?? []);
      setCurrentUsername(emp.username || initialUsername);
      setLoading(false);
    }).catch(() => {
      if (!cancelled) {
        setLoadError(true);
        setLoading(false);
      }
    });
    return () => { cancelled = true; };
  }, [employeeId, initialUsername, loadAttempt]);

  const set = (field: keyof FormData, value: string) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[field];
        return next;
      });
    }
  };

  const validateInfo = () => {
    const errs: Record<string, string> = {};
    if (!form.fullname.trim()) errs.fullname = "Họ tên là bắt buộc";
    if (form.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email))
      errs.email = "Email không hợp lệ";
    if (!form.cccd.trim()) errs.cccd = "CCCD là bắt buộc";
    setErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const handleSaveInfo = async () => {
    if (loading || loadError || !validateInfo()) return;
    setIsSaving(true);
    try {
      await employeeService.updateAdvPartnerUser(employeeId, {
        fullname: form.fullname,
        email: form.email || undefined,
        cccd: form.cccd,
        mobile: form.mobile || undefined,
        bank_id: selectedBank?.id,
        bank_account_number: form.bank_account_number || undefined,
        bank_account_name: form.bank_account_name || undefined,
      });
      setSavedInfo(true);
      setTimeout(() => setSavedInfo(false), 2500);
      toast({ title: "Đã lưu thông tin nhân viên" });
      queryClient.invalidateQueries({ queryKey: ['admin', 'advance-payments', 'flex-pay-employees'] });
    } catch {
      toast({
        title: "Lỗi",
        description: "Không thể lưu thông tin",
        variant: "destructive",
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleChangePassword = async () => {
    if (loading || loadError) return;
    if (!password) {
      setErrors((prev) => ({
        ...prev,
        password: "Vui lòng nhập mật khẩu mới",
      }));
      return;
    }
    if (password.length < 8) {
      setErrors((prev) => ({
        ...prev,
        password: "Mật khẩu phải có ít nhất 8 ký tự",
      }));
      return;
    }
    setIsSaving(true);
    try {
      await employeeService.updateAdvPartnerUser(employeeId, {
        password,
      });
      setPassword("");
      setSavedPassword(true);
      setTimeout(() => setSavedPassword(false), 2500);
      toast({ title: "Đã đổi mật khẩu thành công" });
    } catch {
      toast({
        title: "Lỗi",
        description: "Không thể đổi mật khẩu",
        variant: "destructive",
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        className="w-full sm:w-[480px] md:w-[520px] p-0 flex flex-col h-full"
      >
        {/* Header */}
        <SheetHeader className="px-5 py-4 border-b border-border/40 shrink-0 space-y-0">
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <SheetTitle className="text-left text-base font-bold leading-tight">
                Chỉnh sửa nhân viên
              </SheetTitle>
              <div className="mt-2.5">
                <Avatar
                  name={form.fullname || initialName}
                  username={currentUsername}
                />
              </div>
            </div>
            <button
              onClick={onClose}
              aria-label="Đóng chỉnh sửa nhân viên"
              className="w-11 h-11 rounded-lg flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted transition-colors shrink-0 mt-0.5"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </SheetHeader>

        {/* Body */}
        <div className="flex-1 overflow-y-auto">
          <div className="p-5 space-y-6">

            {loading ? (
              <div className="space-y-4">
                <Skeleton className="h-40 w-full rounded-xl" />
                <Skeleton className="h-32 w-full rounded-xl" />
                <Skeleton className="h-10 w-full rounded-lg" />
              </div>
            ) : loadError ? (
              <div role="alert">
                <ErrorState message="Không thể tải thông tin nhân viên. Vui lòng thử lại trước khi chỉnh sửa." onRetry={() => setLoadAttempt((attempt) => attempt + 1)} />
              </div>
            ) : (
              <>
                {/* ── Personal info ──────────────────────────────────────── */}
                <Section icon={User} title="Thông tin cá nhân">
                  <div className="grid grid-cols-1 min-[380px]:grid-cols-2 gap-3">
                    <div className="min-[380px]:col-span-2">
                      <Field id="adv-partner-fullname" label="Họ và tên *" error={errors.fullname}>
                        <Input
                          id="adv-partner-fullname"
                          value={form.fullname}
                          onChange={(e) => set("fullname", e.target.value)}
                          onBlur={(e) => {
                            const formatted = formatVietnameseName(e.target.value);
                            if (formatted !== e.target.value) set("fullname", formatted);
                          }}
                          className={cn(
                            "h-11 text-sm",
                            errors.fullname && "border-destructive focus-visible:ring-destructive/20"
                          )}
                        />
                      </Field>
                    </div>

                    <Field id="adv-partner-cccd" label="CCCD *" error={errors.cccd}>
                      <Input
                        id="adv-partner-cccd"
                        value={form.cccd}
                        onChange={(e) => set("cccd", e.target.value)}
                        className={cn(
                          "h-11 text-sm font-mono tracking-wide",
                          errors.cccd && "border-destructive focus-visible:ring-destructive/20"
                        )}
                        placeholder="012345678901"
                      />
                    </Field>

                    <Field id="adv-partner-mobile" label="Số điện thoại">
                      <Input
                        id="adv-partner-mobile"
                        value={form.mobile}
                        onChange={(e) => set("mobile", e.target.value)}
                        className="h-11 text-sm"
                        placeholder="0901234567"
                      />
                    </Field>

                    <div className="min-[380px]:col-span-2">
                      <Field id="adv-partner-email" label="Email" error={errors.email}>
                        <Input
                          type="email"
                          id="adv-partner-email"
                          value={form.email}
                          onChange={(e) => set("email", e.target.value)}
                          className={cn(
                            "h-11 text-sm",
                            errors.email && "border-destructive focus-visible:ring-destructive/20"
                          )}
                          placeholder="email@example.com"
                        />
                      </Field>
                    </div>
                  </div>

                  {/* Username — read-only */}
                  {currentUsername && (
                    <Field label="Tên đăng nhập">
                      <div className="h-9 px-3 flex items-center rounded-md border border-border/50 bg-muted/50 font-mono text-[13px] text-muted-foreground">
                        @{currentUsername}
                      </div>
                    </Field>
                  )}
                </Section>

                {/* ── Bank info ──────────────────────────────────────────── */}
                <Section icon={CreditCard} title="Thông tin ngân hàng">
                  <Field label="Ngân hàng">
                    <BankSelector
                      value={selectedBank}
                      onSelect={setSelectedBank}
                      placeholder="Chọn ngân hàng..."
                      canCreateBank={false}
                    />
                  </Field>

                  <div className="grid grid-cols-1 min-[380px]:grid-cols-2 gap-3">
                    <Field id="adv-partner-bank_account_number" label="Số tài khoản">
                      <Input
                        id="adv-partner-bank_account_number"
                        value={form.bank_account_number}
                        onChange={(e) => set("bank_account_number", e.target.value)}
                        className="h-11 text-sm font-mono tracking-wide"
                        placeholder="0123456789"
                      />
                    </Field>
                    <Field id="adv-partner-bank_account_name" label="Tên chủ tài khoản">
                      <Input
                        id="adv-partner-bank_account_name"
                        value={form.bank_account_name}
                        onChange={(e) => {
                          const upper = e.target.value.toUpperCase();
                          set("bank_account_name", upper);
                        }}
                        className="h-11 text-sm uppercase"
                        placeholder="NGUYEN VAN A"
                      />
                    </Field>
                  </div>
                </Section>

                {/* ── Projects — read-only ───────────────────────────────── */}
                {projects.length > 0 && (
                  <Section icon={Building2} title="Dự án hiện tại">
                    <div className="flex flex-wrap gap-2">
                      {projects.map((p) => (
                        <ProjectPill key={p.project_id} project={p} />
                      ))}
                    </div>
                  </Section>
                )}

                {/* ── Save info button ───────────────────────────────────── */}
                <Button
                  className="w-full h-11"
                  onClick={handleSaveInfo}
                  disabled={isSaving}
                >
                  {savedInfo ? (
                    <span className="inline-flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4" /> Đã lưu
                    </span>
                  ) : isSaving ? (
                    <span className="inline-flex items-center gap-2">
                      <span className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                      Đang lưu...
                    </span>
                  ) : (
                    "Lưu thông tin"
                  )}
                </Button>

                {/* Divider */}
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-border/40" />
                  </div>
                  <div className="relative flex justify-center">
                    <span className="px-3 bg-background text-[11px] text-muted-foreground uppercase tracking-wider font-medium">
                      Bảo mật tài khoản
                    </span>
                  </div>
                </div>

                {/* ── Password change ────────────────────────────────────── */}
                <Section icon={Lock} title="Đổi mật khẩu">
                  <Field
                    id="adv-partner-password"
                    label="Mật khẩu mới"
                    error={errors.password}
                  >
                    <div className="relative">
                      <Input
                        id="adv-partner-password"
                        type={showPassword ? "text" : "password"}
                        value={password}
                        onChange={(e) => {
                          setPassword(e.target.value);
                          if (errors.password)
                            setErrors((prev) => {
                              const next = { ...prev };
                              delete next.password;
                              return next;
                            });
                        }}
                        className={cn(
                          "h-11 text-sm pr-10",
                          errors.password && "border-destructive focus-visible:ring-destructive/20"
                        )}
                        placeholder="Tối thiểu 8 ký tự"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        aria-label={showPassword ? "Ẩn mật khẩu" : "Hiện mật khẩu"}
                        className="absolute right-0 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {showPassword ? (
                          <EyeOff className="w-4 h-4" />
                        ) : (
                          <Eye className="w-4 h-4" />
                        )}
                      </button>
                    </div>
                  </Field>

                  <Button
                    variant="outline"
                    className="w-full h-11 text-[13px]"
                    onClick={handleChangePassword}
                    disabled={isSaving || !password}
                  >
                    {savedPassword ? (
                      <span className="inline-flex items-center gap-2 text-emerald-600">
                        <CheckCircle2 className="w-4 h-4" /> Đã đổi mật khẩu
                      </span>
                    ) : isSaving ? (
                      <span className="inline-flex items-center gap-2">
                        <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-foreground border-t-transparent" />
                        Đang đổi...
                      </span>
                    ) : (
                      "Đổi mật khẩu"
                    )}
                  </Button>
                </Section>
              </>
            )}

            {/* bottom padding */}
            <div className="h-4" />
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
