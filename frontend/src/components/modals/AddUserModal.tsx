import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "@/components/ui/sonner";
import { User, UserFormData, CreateUserData } from "@/types/user";
import { validateFullname, formatVietnameseName } from "@/lib/validation";

interface AddUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAddUser: (user: CreateUserData) => void;
  existingUsers: User[];
}

export function AddUserModal({ isOpen, onClose, onAddUser, existingUsers }: AddUserModalProps) {
  const [formData, setFormData] = useState<UserFormData>({
    username: "",
    email: "",
    password: "",
    fullname: "",
    role: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Form validation
  const validateForm = () => {
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

    if (!formData.password.trim()) {
      newErrors.password = "Mật khẩu là bắt buộc";
    } else if (formData.password.length < 6) {
      newErrors.password = "Mật khẩu phải có ít nhất 6 ký tự";
    }

    if (!formData.role) {
      newErrors.role = "Vai trò là bắt buộc";
    }

    // Check if username already exists
    const existingUser = existingUsers.find(u =>
      u.username.toLowerCase() === formData.username.toLowerCase()
    );
    if (existingUser) {
      newErrors.username = "Tên đăng nhập đã tồn tại";
    }

    // Check if email already exists
    const existingEmail = existingUsers.find(u =>
      u.email.toLowerCase() === formData.email.toLowerCase()
    );
    if (existingEmail) {
      newErrors.email = "Email đã được sử dụng";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const resetForm = () => {
    setFormData({ username: "", email: "", password: "", fullname: "", role: "" });
    setErrors({});
  };

  const handleSubmit = () => {
    if (!validateForm()) return;

    onAddUser({
      username: formData.username,
      email: formData.email,
      fullname: formData.fullname,
      password: formData.password,
      role: formData.role as User["role"],
    });

    resetForm();
    onClose();

    toast({
      title: "Thành công",
      description: "Đã thêm người dùng mới",
    });
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Thêm người dùng mới</DialogTitle>
          <DialogDescription>
            Tạo tài khoản mới cho hệ thống
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="add-fullname">Họ và tên *</Label>
            <Input
              id="add-fullname"
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
            <Label htmlFor="add-username">Tên đăng nhập *</Label>
            <Input
              id="add-username"
              placeholder="username"
              value={formData.username}
              onChange={(e) => setFormData({ ...formData, username: e.target.value })}
              className={errors.username ? "border-destructive" : ""}
            />
            {errors.username && <p className="typography-body-medium text-destructive">{errors.username}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="add-email">Email *</Label>
            <Input
              id="add-email"
              type="email"
              placeholder="user@example.com"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
              className={errors.email ? "border-destructive" : ""}
            />
            {errors.email && <p className="typography-body-medium text-destructive">{errors.email}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="add-password">Mật khẩu *</Label>
            <Input
              id="add-password"
              type="password"
              placeholder="••••••••"
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              className={errors.password ? "border-destructive" : ""}
            />
            {errors.password && <p className="typography-body-medium text-destructive">{errors.password}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="add-role">Vai trò *</Label>
            <Select
              value={formData.role}
              onValueChange={(value) => setFormData({ ...formData, role: value as User["role"] })}
            >
              <SelectTrigger className={errors.role ? "border-destructive" : ""}>
                <SelectValue placeholder="Chọn vai trò" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Admin">Quản trị viên</SelectItem>
                <SelectItem value="Partner">Quản lý</SelectItem>
                <SelectItem value="Manager">Manager</SelectItem>
              </SelectContent>
            </Select>
            {errors.role && <p className="typography-body-medium text-destructive">{errors.role}</p>}
          </div>
        </div>

        <DialogFooter>
          <button
            onClick={handleClose}
            className="inline-flex items-center gap-1.5 h-11 px-3 sm:h-8 rounded border border-border bg-card text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors"
          >
            Đóng
          </button>
          <button
            onClick={handleSubmit}
            className="inline-flex items-center gap-1.5 h-11 px-3 sm:h-8 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors"
          >
            Tạo tài khoản
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
