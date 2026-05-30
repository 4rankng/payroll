import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "@/components/ui/sonner";
import { Users, X, Eye, EyeOff } from "lucide-react";
import { User, CreateUserData } from "@/types/user";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { UserAvatar } from "@/components/ui/user-avatar";
import { validateFullname, formatVietnameseName } from "@/lib/validation";
import type { ModalConfig } from "@/types/modal-config.types";

interface AddUserSheetProps {
  isOpen: boolean;
  onClose: () => void;
  onAddUser?: (data: CreateUserData) => Promise<void>;
  existingUsers?: User[];
  loading?: boolean;
}

interface UserFormData {
  email: string;
  username: string;
  fullname: string;
  password: string;
  role: string;
}

function AddUserSheet({
  isOpen,
  onClose,
  onAddUser,
  existingUsers = [],
  loading = false
}: AddUserSheetProps) {
  const [isSaving, setIsSaving] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  const [formData, setFormData] = useState<UserFormData>({
    email: "",
    username: "",
    fullname: "",
    password: "",
    role: "",
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (isOpen) {
      // Reset form when sheet opens
      setFormData({
        email: "",
        username: "",
        fullname: "",
        password: "",
        role: "",
      });
      setErrors({});
      setShowPassword(false);
    }
  }, [isOpen]);

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.username.trim()) {
      newErrors.username = "Tên đăng nhập là bắt buộc";
    }

    if (formData.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = "Email không hợp lệ";
    }

    const fullnameValidation = validateFullname(formData.fullname);
    if (!fullnameValidation.valid) {
      newErrors.fullname = fullnameValidation.error || 'Họ tên không hợp lệ';
    }

    if (!formData.password.trim()) {
      newErrors.password = "Mật khẩu là bắt buộc";
    } else {
      const password = formData.password;
      const passwordErrors = [];

      if (password.length < 8) {
        passwordErrors.push("ít nhất 8 ký tự");
      }
      if (!/(?=.*[a-z])/.test(password)) {
        passwordErrors.push("ít nhất 1 chữ thường");
      }
      if (!/(?=.*[A-Z])/.test(password)) {
        passwordErrors.push("ít nhất 1 chữ hoa");
      }
      if (!/(?=.*\d)/.test(password)) {
        passwordErrors.push("ít nhất 1 số");
      }
      if (!/(?=.*[!@#$%^&*(),.?":{}|<>])/.test(password)) {
        passwordErrors.push("ít nhất 1 ký tự đặc biệt");
      }

      if (passwordErrors.length > 0) {
        newErrors.password = `Mật khẩu phải có ${passwordErrors.join(", ")}`;
      }
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

    // Check if email already exists (only if email is provided)
    if (formData.email.trim()) {
      const existingEmail = existingUsers.find(u =>
        u.email.toLowerCase() === formData.email.toLowerCase()
      );
      if (existingEmail) {
        newErrors.email = "Email đã được sử dụng";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      toast({
        title: "Lỗi xác thực",
        description: "Vui lòng kiểm tra và điền đầy đủ thông tin.",
        variant: "destructive"
      });
      return;
    }

    setIsSaving(true);

    try {
      if (onAddUser) {
        await onAddUser({
          ...(formData.email.trim() && { email: formData.email.trim() }),
          username: formData.username.trim(),
          fullname: formData.fullname.trim(),
          password: formData.password.trim(),
          role: formData.role as User["role"],
        });

        onClose();
      }
    } catch (error) {
      // Error handling is now done in the mutation hook
      console.error('User creation failed:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleInputChange = (field: keyof UserFormData, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };


  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
            {formData.email || formData.fullname ? (
          <UserAvatar
            name={formData.fullname}
            email={formData.email}
            size="md"
          />
        ) : (
          <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
            <Users className="h-4 w-4 text-primary" />
              </div>
            )}
            <div className="space-y-0.5 flex-1 min-w-0">
              <h2 className="typography-headline-medium text-sm sm:text-base font-semibold">
               Thêm người dùng mới
              </h2>
              <p className="typography-body-medium text-muted-foreground text-xs sm:text-sm">
                Điền thông tin để tạo tài khoản mới
              </p>
            </div>
          </div>
        )
      }}
      footer={
        <div className="grid grid-cols-2 gap-2">
          <Button variant="outline" onClick={onClose} className="w-full" disabled={isSaving || loading}>
            <X />
            Đóng
          </Button>
          <Button onClick={handleSubmit} disabled={isSaving || loading} className="w-full" variant="default">
            {isSaving ? 'Đang thêm...' : 'Thêm'}
          </Button>
        </div>
      }
    >
            <div className="space-y-3 -mx-2 px-2">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="fullname" className="text-xs font-medium">Họ và tên *</Label>
                    <Input
                      id="fullname"
                      value={formData.fullname}
                      onChange={(e) => handleInputChange("fullname", e.target.value)}
                      onBlur={(e) => {
                        const formatted = formatVietnameseName(e.target.value);
                        if (formatted !== e.target.value) {
                          handleInputChange("fullname", formatted);
                        }
                      }}
                      placeholder="Nguyễn Văn A"
                      className={errors.fullname ? 'border-red-500 h-9' : 'h-9'}
                    />
                    {errors.fullname && <p className="text-xs text-red-500 mt-0.5">{errors.fullname}</p>}
                  </div>

                  <div className="space-y-1">
                    <Label htmlFor="role" className="text-xs font-medium">Vai trò *</Label>
                    <Select
                      value={formData.role}
                      onValueChange={(value) => handleInputChange("role", value)}
                    >
                      <SelectTrigger className={errors.role ? 'border-red-500 h-9' : 'h-9'}>
                        <SelectValue placeholder="Chọn vai trò" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="admin">Quản trị viên</SelectItem>
                        <SelectItem value="partner">Quản lý</SelectItem>
                        <SelectItem value="employee">Nhân viên</SelectItem>
                        <SelectItem value="adv_partner">Quản lý ứng lương</SelectItem>
                      </SelectContent>
                    </Select>
                    {errors.role && <p className="text-xs text-red-500 mt-0.5">{errors.role}</p>}
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="username" className="text-xs font-medium">Tên đăng nhập *</Label>
                    <Input
                      id="username"
                      value={formData.username}
                      onChange={(e) => handleInputChange("username", e.target.value)}
                      placeholder="admin123"
                      className={errors.username ? 'border-red-500 h-9' : 'h-9'}
                    />
                    {errors.username && <p className="text-xs text-red-500 mt-0.5">{errors.username}</p>}
                  </div>

                  <div className="space-y-1">
                    <Label htmlFor="email" className="text-xs font-medium">Email</Label>
                    <Input
                      id="email"
                      type="email"
                      value={formData.email}
                      onChange={(e) => handleInputChange("email", e.target.value)}
                      placeholder="admin@example.com"
                      className={errors.email ? 'border-red-500 h-9' : 'h-9'}
                    />
                    {errors.email && <p className="text-xs text-red-500 mt-0.5">{errors.email}</p>}
                  </div>
                </div>

                <div className="space-y-1">
                  <Label htmlFor="password" className="text-xs font-medium">Mật khẩu *</Label>
                  <div className="relative">
                    <Input
                      id="password"
                      type={showPassword ? "text" : "password"}
                      value={formData.password}
                      onChange={(e) => handleInputChange("password", e.target.value)}
                      placeholder="Nhập mật khẩu"
                      className={errors.password ? 'border-red-500 h-9 pr-10' : 'h-9 pr-10'}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="absolute right-0 top-0 h-9 w-9 px-0 hover:bg-transparent"
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Eye className="h-4 w-4 text-muted-foreground" />
                      )}
                    </Button>
                  </div>
                  {errors.password && <p className="text-xs text-red-500 mt-0.5">{errors.password}</p>}
                </div>

                {/* Compact Password Requirements */}
                <div className="bg-muted/30 rounded-xl p-2.5 space-y-1.5">
                  <h4 className="text-xs font-medium text-muted-foreground">Yêu cầu mật khẩu:</h4>
                  <div className="grid grid-cols-1 xs:grid-cols-2 gap-1 text-xs text-muted-foreground">
                    <div className="flex items-center gap-1.5 min-h-[20px]">
                      <div className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${formData.password.length >= 8 ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>8+ ký tự</span>
                    </div>
                    <div className="flex items-center gap-1.5 min-h-[20px]">
                      <div className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${/(?=.*[a-z])/.test(formData.password) ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>Chữ thường</span>
                    </div>
                    <div className="flex items-center gap-1.5 min-h-[20px]">
                      <div className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${/(?=.*[A-Z])/.test(formData.password) ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>Chữ hoa</span>
                    </div>
                    <div className="flex items-center gap-1.5 min-h-[20px]">
                      <div className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${/(?=.*\d)/.test(formData.password) ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>Số</span>
                    </div>
                    <div className="flex items-center gap-1.5 min-h-[20px] xs:col-span-2">
                      <div className={`h-1.5 w-1.5 rounded-full flex-shrink-0 ${/(?=.*[!@#$%^&*(),.?":{}|<>])/.test(formData.password) ? 'bg-green-500' : 'bg-gray-300'}`} />
                      <span>Ký tự đặc biệt</span>
                    </div>
                  </div>
                </div>
              </div>
    </SlideSheetTemplate>
  );
}

export const modalConfig: ModalConfig = {
  id: 'add_user_sheet',
  name: 'Thêm người dùng mới',
  description: 'Điền thông tin để tạo tài khoản người dùng mới trong hệ thống.',
  category: 'user',
  permissions: {
    action: 'create',
    subject: 'User',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: false,
  },
  requiresAuth: true,
  encryptData: false,
};

export default AddUserSheet;
