import React, { useState, useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { User2, Mail, Shield, Calendar, Clock, Edit2, Save, X, Key, CreditCard, Phone } from 'lucide-react';
import {
  Sheet,
  SheetContent,
} from '@/components/ui/sheet';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/contexts';
import { useProfile } from '@/hooks/api/useProfile';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { MODAL_IDS } from '@/constants/modalRegistry';
import { format, formatDistanceToNow } from 'date-fns';
import { vi } from 'date-fns/locale';
import type { ModalConfig } from "@/types/modal-config.types";

const profileUpdateSchema = z.object({
  name: z.string().min(1, 'Tên không được để trống'),
  email: z.string().optional().refine((val) => {
    if (!val || val.trim() === '') return true; // Allow empty email
    return z.string().email().safeParse(val).success;
  }, { message: 'Email không hợp lệ' }),
  cccd: z.string().optional().refine((val) => {
    if (!val || val.trim() === '') return true;
    return /^\d{6,15}$/.test(val);
  }, { message: 'Số CCCD không hợp lệ' }),
  mobile: z.string().optional().refine((val) => {
    if (!val || val.trim() === '') return true;
    return /^[0-9+\-\s]{6,20}$/.test(val);
  }, { message: 'Số điện thoại không hợp lệ' }),
});

type ProfileUpdateFormData = z.infer<typeof profileUpdateSchema>;

interface InfoRowProps {
  icon: React.ReactNode;
  label: string;
  value: string;
}

const InfoRow = ({ icon, label, value }: InfoRowProps) => (
  <div className="flex items-start gap-3 rounded-lg px-2 py-2.5 -mx-2 hover:bg-muted/40 transition-colors">
    <span className="text-muted-foreground flex-shrink-0">{icon}</span>
    <div className="flex min-w-0 flex-1 flex-col gap-1 min-[380px]:flex-row min-[380px]:items-baseline min-[380px]:justify-between min-[380px]:gap-3">
      <span className="text-xs text-muted-foreground flex-shrink-0">{label}</span>
      <span className="text-sm font-medium break-all min-[380px]:text-right">{value}</span>
    </div>
  </div>
);

interface UserProfileSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export const UserProfileSheet = ({ isOpen, onClose }: UserProfileSheetProps) => {
  const { user, updateUser } = useAuth();
  const { completeUser, isLoadingCompleteUser, updateProfileMutation } = useProfile();
  const { openModal } = useModalNavigation();
  const [isEditing, setIsEditing] = useState(false);

  // Use complete profile data if available, fallback to auth context user
  const displayUser = useMemo(() => {
    return completeUser ? {
      id: String(completeUser.id),
      name: completeUser.fullname,
      email: completeUser.email,
      role: completeUser.role,
      created_at: completeUser.created_at,
      updated_at: completeUser.updated_at,
      last_login: completeUser.last_login,
    } : user;
  }, [completeUser, user]);

  const form = useForm<ProfileUpdateFormData>({
    resolver: zodResolver(profileUpdateSchema),
    values: {
      name: displayUser?.name || '',
      email: displayUser?.email || '',
      cccd: completeUser?.cccd || '',
      mobile: completeUser?.mobile || '',
    },
  });

  const handleEditToggle = () => {
    if (isEditing) {
      // Reset form when canceling edit
      form.reset({
        name: displayUser?.name || '',
        email: displayUser?.email || '',
        cccd: completeUser?.cccd || '',
        mobile: completeUser?.mobile || '',
      });
    }
    setIsEditing(!isEditing);
  };

  const handleSave = async (data: ProfileUpdateFormData) => {
    try {
      const payload: { fullname?: string; email?: string; cccd?: string; mobile?: string } = {
        fullname: data.name,
      };

      // Only include email if it was provided (avoid sending empty string)
      if (data.email !== undefined) {
        payload.email = data.email || '';
      }

      // Only include cccd/mobile if they have a value or were explicitly cleared
      if (data.cccd !== undefined) {
        payload.cccd = data.cccd || '';
      }
      if (data.mobile !== undefined) {
        payload.mobile = data.mobile || '';
      }

      await updateProfileMutation.mutateAsync(payload);

      updateUser({
        name: data.name,
        email: data.email || '',
      });

      setIsEditing(false);
    } catch {
      // Error is handled globally; mutation state resets automatically after error
    }
  };

  const handleChangePassword = () => {
    openModal(MODAL_IDS.CHANGE_PASSWORD);
  };

  if (!displayUser) return null;

  // Show loading state if fetching complete user data
  if (isLoadingCompleteUser && !completeUser) {
    return (
      <Sheet open={isOpen} onOpenChange={onClose}>
        <SheetContent className="w-full sm:max-w-md flex flex-col h-full p-0 gap-0">
          <div className="flex-1 flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
          </div>
        </SheetContent>
      </Sheet>
    );
  }

  const getRoleText = (role: 'admin' | 'partner' | 'employee' | 'adv_partner'): string => {
    if (role === 'admin') return 'Quản trị viên';
    if (role === 'partner' || role === 'adv_partner') return 'Quản lý';
    return 'Nhân viên';
  };

  const getRoleBadgeVariant = (role: 'admin' | 'partner' | 'employee' | 'adv_partner') => {
    return role === 'admin' ? 'default' : 'secondary';
  };

  const formatLastLogin = (lastLogin?: string) => {
    if (!lastLogin) return 'Chưa có thông tin';

    try {
      const loginDate = new Date(lastLogin);
      return formatDistanceToNow(loginDate, {
        addSuffix: true,
        locale: vi
      });
    } catch {
      return 'Không xác định';
    }
  };

  const formatCreatedAt = (createdAt?: string) => {
    if (!createdAt) return 'Chưa có thông tin';

    try {
      const date = new Date(createdAt);
      return format(date, 'dd/MM/yyyy');
    } catch {
      return 'Không xác định';
    }
  };

  return (
    <Sheet open={isOpen} onOpenChange={onClose}>
      <SheetContent className="w-full sm:max-w-md flex flex-col h-full p-0 gap-0">
        {/* Hero header */}
        <div className="relative bg-gradient-to-br from-primary/10 via-primary/5 to-background px-6 pt-10 pb-6 border-b">
          <div className="flex items-end gap-4">
            <UserAvatar
              email={displayUser.email}
              name={displayUser.name}
              size="xl"
              className="h-16 w-16 flex-shrink-0 ring-4 ring-background shadow-md"
            />
            <div className="flex-1 min-w-0 pb-1">
              <h2 className="text-lg font-semibold leading-tight break-words">{displayUser.name}</h2>
              <div className="mt-1.5">
                <Badge variant={getRoleBadgeVariant(displayUser.role)} className="text-xs">
                  {getRoleText(displayUser.role)}
                </Badge>
              </div>
            </div>
            {!isEditing && (
              <Button
                size="icon"
                variant="ghost"
                onClick={handleEditToggle}
                className="h-11 w-11 flex-shrink-0 text-muted-foreground hover:text-foreground"
                title="Chỉnh sửa"
              >
                <Edit2 className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {isEditing ? (
            <Form {...form}>
              <form className="p-6 space-y-4">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-2">Chỉnh sửa thông tin</p>
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs text-muted-foreground">Họ và tên</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder="Nhập họ và tên"
                          disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
                          className="h-11"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs text-muted-foreground">Email</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type="email"
                          placeholder="Nhập email"
                          disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
                          className="h-11"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="cccd"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs text-muted-foreground">Số CCCD</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder="Nhập số CCCD"
                          disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
                          className="h-11"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="mobile"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs text-muted-foreground">Số điện thoại</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder="Nhập số điện thoại"
                          disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
                          className="h-11"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </form>
            </Form>
          ) : (
            <div className="divide-y">
              {/* Contact section */}
              <div className="px-6 pt-5 pb-2">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">Liên hệ</p>
                <div className="space-y-1">
                  <InfoRow icon={<User2 className="h-4 w-4" />} label="Họ và tên" value={displayUser.name} />
                  <InfoRow icon={<Mail className="h-4 w-4" />} label="Email" value={displayUser.email} />
                  {completeUser?.cccd && (
                    <InfoRow icon={<CreditCard className="h-4 w-4" />} label="Số CCCD" value={completeUser.cccd} />
                  )}
                  {completeUser?.mobile && (
                    <InfoRow icon={<Phone className="h-4 w-4" />} label="Số điện thoại" value={completeUser.mobile} />
                  )}
                  <InfoRow icon={<Shield className="h-4 w-4" />} label="Vai trò" value={getRoleText(displayUser.role)} />
                </div>
              </div>

              {/* Account section */}
              <div className="px-6 pt-5 pb-2">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">Tài khoản</p>
                <div className="space-y-1">
                  <InfoRow icon={<Calendar className="h-4 w-4" />} label="Ngày tạo" value={formatCreatedAt(displayUser.created_at)} />
                  <InfoRow icon={<Clock className="h-4 w-4" />} label="Đăng nhập cuối" value={formatLastLogin(displayUser.last_login)} />
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer actions */}
        <div className="flex-shrink-0 border-t bg-background px-6 py-4 pb-[max(1rem,calc(1rem+env(safe-area-inset-bottom)))]">
          {isEditing ? (
            <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleEditToggle}
                className="min-h-11"
                disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
              >
                Hủy
              </Button>
              <Button
                type="button"
                onClick={form.handleSubmit(handleSave)}
                className="min-h-11"
                disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
              >
                <Save className="h-4 w-4 mr-1.5" />
                {(form.formState.isSubmitting || updateProfileMutation.isPending) ? 'Đang lưu...' : 'Lưu thay đổi'}
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleChangePassword}
                className="min-h-11 flex-1"
              >
                <Key className="h-4 w-4 mr-1.5" />
                Đổi mật khẩu
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={onClose}
                className="h-11 w-11 text-muted-foreground"
                title="Đóng"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
};

export const modalConfig: ModalConfig = {
  id: 'user_profile_sheet',
  name: 'Thông tin cá nhân',
  description: 'Xem và chỉnh sửa thông tin cá nhân của bạn.',
  category: 'user',
  permissions: {
    action: 'read',
    subject: 'User',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: false,
  },
  requiresAuth: true,
  encryptData: false,
};

export default UserProfileSheet;
