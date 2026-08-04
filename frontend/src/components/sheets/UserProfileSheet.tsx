import React, { useState, useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { User2, Mail, Shield, Calendar, Clock, Edit2, Save, X, KeyRound, CreditCard, Phone } from 'lucide-react';
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
import { UserAvatar } from '@/components/ui/user-avatar';
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
  <li className="ct-list-row grid-cols-[2.5rem_minmax(0,1fr)] gap-2 px-0 py-3 first:pt-0 last:pb-0">
    <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]">
      {icon}
    </span>
    <div className="ct-list-col-grow min-w-0">
      <p className="employee-type-label text-[var(--employee-text-secondary)]">{label}</p>
      <p className="employee-type-body mt-0.5 break-all font-semibold text-[var(--employee-text)]">{value}</p>
    </div>
  </li>
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

  const storesMobileOnUser = displayUser?.role === 'admin' || displayUser?.role === 'partner';

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
      if (storesMobileOnUser && data.mobile !== undefined) {
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
      <SheetContent
        title="Thông tin cá nhân"
        description="Xem và chỉnh sửa thông tin tài khoản"
        data-theme="congtruong"
        className="flex h-full w-full flex-col gap-0 bg-[var(--employee-page)] p-0 sm:max-w-md"
      >
        <div className="ct-hero relative min-h-0 overflow-hidden bg-gradient-to-br from-employee-800 via-employee-700 to-employee-500 px-5 pb-6 pt-[max(1.5rem,env(safe-area-inset-top,0px))] text-white">
          <div className="ct-hero-content w-full max-w-none justify-between p-0">
            <div className="flex min-w-0 items-center gap-3">
              <div className="ct-avatar shrink-0">
                <div className="h-16 w-16 rounded-2xl ring-4 ring-white/20 ring-offset-2 ring-offset-employee-700">
                  <UserAvatar
                    email={displayUser.email}
                    name={displayUser.name}
                    size="xl"
                    className="h-16 w-16"
                  />
                </div>
              </div>
              <div className="min-w-0">
                <p className="employee-type-label-caps text-white/70">Tài khoản của bạn</p>
                <h2 className="employee-type-section-title mt-0.5 break-words text-white">{displayUser.name}</h2>
                <span className="ct-badge ct-badge-sm mt-2 border-white/20 bg-white/15 px-2.5 text-white">
                  {getRoleText(displayUser.role)}
                </span>
              </div>
            </div>
            {!isEditing && (
              <button
                type="button"
                onClick={handleEditToggle}
                className="inline-flex items-center justify-center rounded-full h-11 min-h-11 w-11 shrink-0 border-0 bg-white/10 p-0 text-white transition-colors hover:bg-white/20 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/30"
                aria-label="Chỉnh sửa thông tin cá nhân"
              >
                <Edit2 className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-5">
          {isEditing ? (
            <Form {...form}>
              <form className="rounded-xl space-y-4 border border-[var(--employee-border)] bg-white p-4 shadow-[var(--employee-shadow)]">
                <div>
                  <p className="employee-type-label-caps font-semibold text-[var(--employee-accent)]">Chỉnh sửa hồ sơ</p>
                  <p className="employee-type-body-sm mt-1 text-[var(--employee-text-secondary)]">Cập nhật các thông tin liên hệ của bạn.</p>
                </div>
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
                {storesMobileOnUser && <FormField
                  control={form.control}
                  name="mobile"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="text-xs text-muted-foreground">Số điện thoại</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type="tel"
                          inputMode="tel"
                          autoComplete="tel"
                          placeholder="Nhập số điện thoại"
                          disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
                          className="h-11"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />}
              </form>
            </Form>
          ) : (
            <div className="space-y-4">
              <section className="rounded-xl border border-[var(--employee-border)] bg-white shadow-[var(--employee-shadow)]">
                <div className="gap-3 p-4">
                  <div>
                    <div>
                      <p className="employee-type-label-caps font-semibold text-[var(--employee-accent)]">Liên hệ</p>
                      <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">Thông tin dùng để liên lạc với bạn</p>
                    </div>
                  </div>
                  <ul className="ct-list p-0">
                    <InfoRow icon={<User2 className="h-4 w-4" />} label="Họ và tên" value={displayUser.name} />
                    <InfoRow icon={<Mail className="h-4 w-4" />} label="Email" value={displayUser.email || 'Chưa cập nhật'} />
                  {completeUser?.cccd && (
                    <InfoRow icon={<CreditCard className="h-4 w-4" />} label="Số CCCD" value={completeUser.cccd} />
                  )}
                  {storesMobileOnUser && completeUser?.mobile && (
                    <InfoRow icon={<Phone className="h-4 w-4" />} label="Số điện thoại" value={completeUser.mobile} />
                  )}
                  <InfoRow icon={<Shield className="h-4 w-4" />} label="Vai trò" value={getRoleText(displayUser.role)} />
                  </ul>
                </div>
              </section>

              <section className="rounded-xl border border-[var(--employee-border)] bg-white shadow-[var(--employee-shadow)]">
                <div className="gap-3 p-4">
                  <div>
                    <p className="employee-type-label-caps font-semibold text-[var(--employee-accent)]">Tài khoản</p>
                    <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">Trạng thái và hoạt động gần đây</p>
                  </div>
                  <ul className="ct-list p-0">
                  <InfoRow icon={<Calendar className="h-4 w-4" />} label="Ngày tạo" value={formatCreatedAt(displayUser.created_at)} />
                  <InfoRow icon={<Clock className="h-4 w-4" />} label="Đăng nhập cuối" value={formatLastLogin(displayUser.last_login)} />
                  </ul>
                </div>
              </section>
            </div>
          )}
        </div>

        {/* Footer actions */}
        <div className="flex-shrink-0 border-t border-[var(--employee-border)] bg-white px-4 py-3 pb-[max(0.75rem,calc(0.75rem+env(safe-area-inset-bottom)))]">
          {isEditing ? (
            <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
              <button
                type="button"
                onClick={handleEditToggle}
                className="inline-flex items-center justify-center rounded-md min-h-11 border border-[var(--employee-border)] bg-white text-sm font-semibold normal-case text-[var(--employee-text)] shadow-none transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-50"
                disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
              >
                Hủy
              </button>
              <button
                type="button"
                onClick={form.handleSubmit(handleSave)}
                className="inline-flex items-center justify-center gap-1.5 rounded-md min-h-11 border-0 bg-[var(--employee-accent)] px-4 text-sm font-semibold normal-case text-white shadow-[var(--employee-cta-shadow)] transition-colors hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-50"
                disabled={form.formState.isSubmitting || updateProfileMutation.isPending}
              >
                <Save className="h-4 w-4 mr-1.5" />
                {(form.formState.isSubmitting || updateProfileMutation.isPending) ? 'Đang lưu...' : 'Lưu thay đổi'}
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleChangePassword}
                className="inline-flex items-center justify-center gap-1.5 min-h-11 flex-1 rounded-md border border-[var(--employee-accent)] bg-transparent px-4 text-sm font-semibold normal-case text-[var(--employee-accent)] transition-colors hover:bg-[var(--employee-accent-soft)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
              >
                <KeyRound className="h-4 w-4 mr-1.5" />
                Đổi mật khẩu
              </button>
              <button
                type="button"
                onClick={onClose}
                className="inline-flex items-center justify-center rounded-full h-11 min-h-11 w-11 p-0 text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-page)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]"
                aria-label="Đóng thông tin cá nhân"
              >
                <X className="h-4 w-4" />
              </button>
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
