import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { Eye, EyeOff } from "lucide-react";
import { User } from "@/types/user";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { validateFullname, formatVietnameseName } from "@/lib/validation";
import { userService } from "@/services/api/user.service";
import { cn } from "@/lib/utils";
import type { ModalConfig } from "@/types/modal-config.types";

interface EditUserSheetProps {
  isOpen: boolean;
  onClose: () => void;
  id?: string;
  loading?: boolean;
}

interface FormData {
  email: string;
  mobile: string;
  username: string;
  fullname: string;
  password: string;
}

function EditUserSheet({
  isOpen,
  onClose,
  id,
  loading = false,
}: EditUserSheetProps) {
  const [user, setUser] = useState<User | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoadingUser, setIsLoadingUser] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const [formData, setFormData] = useState<FormData>({
    email: "",
    mobile: "",
    username: "",
    fullname: "",
    password: "",
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (isOpen && id) {
      setIsLoadingUser(true);
      userService
        .getUserById(Number(id))
        .then((data) => {
          setUser(data);
          setFormData({
            email: data.email || "",
            mobile: data.mobile || "",
            username: data.username || "",
            fullname: data.fullname || "",
            password: "",
          });
          setErrors({});
        })
        .catch(() => {
          toast({ title: "Lỗi", description: "Không thể tải thông tin người dùng", variant: "destructive" });
        })
        .finally(() => setIsLoadingUser(false));
    }
  }, [isOpen, id]);

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
      newErrors.fullname = fullnameValidation.error || "Họ tên không hợp lệ";
    }

    if (formData.password && formData.password.length < 6) {
      newErrors.password = "Mật khẩu phải có ít nhất 6 ký tự";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm() || !user) return;

    setIsSaving(true);
    try {
      await userService.updateUser(user.id, {
        email: formData.email,
        ...((user.role === "admin" || user.role === "partner") && { mobile: formData.mobile }),
        username: formData.username,
        fullname: formData.fullname,
      });

      if (formData.password) {
        await userService.resetPassword(user.id, { password: formData.password });
      }

      toast({ title: "Thành công", description: "Đã cập nhật thông tin người dùng" });
      onClose();
    } catch {
      toast({ title: "Lỗi", description: "Không thể cập nhật người dùng", variant: "destructive" });
    } finally {
      setIsSaving(false);
    }
  };

  const handleClose = () => {
    setFormData({ email: "", mobile: "", username: "", fullname: "", password: "" });
    setErrors({});
    onClose();
  };

  const handleInputChange = (field: keyof FormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => {
        const next = { ...prev };
        delete next[field];
        return next;
      });
    }
  };

  if (!isOpen) return null;

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={handleClose}
      title="Chỉnh sửa người dùng"
      subtitle={user ? `@${user.username}` : ""}
    >
      {isLoadingUser ? (
        <div className="flex items-center justify-center py-12">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-foreground border-t-transparent" />
        </div>
      ) : (
        <div className="flex flex-col gap-5 p-6">
          <div className="space-y-2">
            <Label htmlFor="edit-fullname">Họ tên *</Label>
            <Input
              id="edit-fullname"
              placeholder="Nguyễn Văn A"
              value={formData.fullname}
              onChange={(e) => handleInputChange("fullname", e.target.value)}
              onBlur={(e) => {
                const formatted = formatVietnameseName(e.target.value);
                if (formatted !== e.target.value) {
                  setFormData((prev) => ({ ...prev, fullname: formatted }));
                }
              }}
              className={errors.fullname ? "border-destructive" : ""}
            />
            {errors.fullname && <p className="text-sm text-destructive">{errors.fullname}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-username">Tên đăng nhập *</Label>
            <Input
              id="edit-username"
              placeholder="username"
              value={formData.username}
              onChange={(e) => handleInputChange("username", e.target.value)}
              className={errors.username ? "border-destructive" : ""}
            />
            {errors.username && <p className="text-sm text-destructive">{errors.username}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-email">Email *</Label>
            <Input
              id="edit-email"
              type="email"
              placeholder="user@example.com"
              value={formData.email}
              onChange={(e) => handleInputChange("email", e.target.value)}
              className={errors.email ? "border-destructive" : ""}
            />
            {errors.email && <p className="text-sm text-destructive">{errors.email}</p>}
          </div>

          {(user?.role === "admin" || user?.role === "partner") && (
            <div className="space-y-2">
              <Label htmlFor="edit-mobile">Số điện thoại</Label>
              <Input
                id="edit-mobile"
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                placeholder="090 123 4567"
                value={formData.mobile}
                onChange={(e) => handleInputChange("mobile", e.target.value)}
              />
              <p className="text-xs text-muted-foreground">Dùng để đặt lại mật khẩu qua Zalo OTP.</p>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="edit-password">Mật khẩu mới</Label>
            <div className="relative">
              <Input
                id="edit-password"
                type={showPassword ? "text" : "password"}
                placeholder="Để trống nếu không đổi"
                value={formData.password}
                onChange={(e) => handleInputChange("password", e.target.value)}
                className={cn("pr-10", errors.password ? "border-destructive" : "")}
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
            {errors.password && <p className="text-sm text-destructive">{errors.password}</p>}
          </div>

          <div className="flex justify-end gap-2 pt-4 border-t">
            <Button variant="outline" onClick={handleClose} disabled={isSaving}>
              Đóng
            </Button>
            <Button onClick={handleSubmit} disabled={isSaving || loading}>
              {isSaving ? "Đang lưu..." : "Cập nhật"}
            </Button>
          </div>
        </div>
      )}
    </SlideSheetTemplate>
  );
}

export const modalConfig: ModalConfig = {
  id: "edit_user",
  name: "Chỉnh sửa người dùng",
  description: "Chỉnh sửa thông tin và mật khẩu người dùng.",
  category: "user",
  permissions: {
    action: "update",
    subject: "User",
    roles: ["admin", "partner", "adv_partner"],
  },
  deeplink: {
    enabled: true,
    params: ["id"],
    example: "?modal=edit_user&id=1",
  },
  requiresAuth: true,
  encryptData: false,
};

export default EditUserSheet;
