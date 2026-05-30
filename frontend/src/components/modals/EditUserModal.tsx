import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { User, UserFormData, UpdateUserData } from "@/types/user";
import { validateFullname, formatVietnameseName } from "@/lib/validation";

interface EditUserModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (user: UpdateUserData & { id: number }) => void;
  user: User | null;
  loading?: boolean;
}

export function EditUserModal({ open, onClose, onSubmit, user, loading = false }: EditUserModalProps) {
  const [formData, setFormData] = useState<Omit<UserFormData, 'password'>>({
    email: "",
    username: "",
    fullname: "",
    role: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Update form data when user changes
  useEffect(() => {
    if (user) {
      setFormData({
        email: user.email,
        username: user.username,
        fullname: user.fullname,
        role: user.role,
      });
      setErrors({});
    }
  }, [user]);

  // Form validation
  const validateForm = () => {
    if (!user) return false;

    const newErrors: Record<string, string> = {};

    if (!formData.username.trim()) {
      newErrors.username = "Tên đăng nhập là bắt buộc";
    }

    if (!formData.email.trim()) {
      newErrors.email = "Email là bắt buộc";
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = "Email không hợp lệ";
    }

    const fullnameValidation = validateFullname(formData.fullname);
    if (!fullnameValidation.valid) {
      newErrors.fullname = fullnameValidation.error || 'Họ tên không hợp lệ';
    }

    if (!formData.role) {
      newErrors.role = "Vai trò là bắt buộc";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const resetForm = () => {
    setFormData({ email: "", username: "", fullname: "", role: "" });
    setErrors({});
  };

  const handleSubmit = () => {
    if (!validateForm() || !user) return;

    onSubmit({
      id: user.id,
      email: formData.email,
      username: formData.username,
      fullname: formData.fullname,
      role: formData.role as User["role"],
    });

    onClose();
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Chỉnh sửa người dùng</DialogTitle>
          <DialogDescription>
            Cập nhật thông tin tài khoản
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="edit-fullname">Họ tên *</Label>
            <Input
              id="edit-fullname"
              placeholder="Nguyễn Văn A"
              value={formData.fullname}
              onChange={(e) => setFormData({ ...formData, fullname: e.target.value })}
              onBlur={(e) => {
                const formatted = formatVietnameseName(e.target.value);
                if (formatted !== e.target.value) {
                  setFormData({ ...formData, fullname: formatted });
                }
              }}
              className={errors.fullname ? "border-destructive" : ""}
            />
            {errors.fullname && <p className="typography-body-medium text-destructive">{errors.fullname}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-username">Tên đăng nhập *</Label>
            <Input
              id="edit-username"
              placeholder="username"
              value={formData.username}
              onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              className={errors.username ? "border-destructive" : ""}
            />
            {errors.username && <p className="typography-body-medium text-destructive">{errors.username}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-email">Email *</Label>
            <Input
              id="edit-email"
              type="email"
              placeholder="user@example.com"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              className={errors.email ? "border-destructive" : ""}
            />
            {errors.email && <p className="typography-body-medium text-destructive">{errors.email}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-role">Vai trò *</Label>
            <Select
              value={formData.role}
              onValueChange={(value) => setFormData({ ...formData, role: value as User["role"] })}
            >
              <SelectTrigger className={errors.role ? "border-destructive" : ""}>
                <SelectValue placeholder="Chọn vai trò" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">Quản trị viên</SelectItem>
                <SelectItem value="partner">Quản lý</SelectItem>
              </SelectContent>
            </Select>
            {errors.role && <p className="typography-body-medium text-destructive">{errors.role}</p>}
          </div>
        </div>

        <DialogFooter>
          <button
            onClick={handleClose}
            disabled={loading}
            className="inline-flex items-center gap-1.5 h-8 px-3 rounded border border-border bg-background text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            Đóng
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading}
            className="inline-flex items-center gap-1.5 h-8 px-3 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
          >
            {loading ? "Đang cập nhật..." : "Cập nhật"}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
