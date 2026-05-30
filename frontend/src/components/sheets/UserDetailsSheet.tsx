import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState, useEffect, useMemo, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { User, UpdateUserData } from "@/types/user";
import { UserHeader } from "@/components/users/details/UserHeader";
import { UserActions } from "@/components/users/details/UserActions";
import {
  User as LucideUser,
  Clock,
  Activity,
  FileText,
} from "lucide-react";
import { useUserActivities } from "@/hooks/api/useUsers";
import { ResetPasswordModal } from "@/components/modals/ResetPasswordModal";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { format, formatDistanceToNow } from "date-fns";
import { vi } from "date-fns/locale";

interface UserDetailsSheetProps {
  user: User | null;
  isOpen: boolean;
  onClose: () => void;
  onUpdate: (userId: number, userData: UpdateUserData) => void;
  onDelete: (user: User) => void;
  onResetPassword?: (userId: number, password: string) => void;
  loading?: boolean;
}

function UserDetailsSheet({
  user,
  isOpen,
  onClose,
  onUpdate,
  onDelete,
  onResetPassword,
  loading = false,
}: UserDetailsSheetProps) {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const [formData, setFormData] = useState<UpdateUserData>({
    email: "",
    username: "",
    fullname: "",
    role: "partner",
  });

  const { data: activityData, isLoading: activityLoading } = useUserActivities(
    user?.id || 0,
    30,
    !!user?.id && isOpen
  );

  const avatar = useMemo(() => ({
    custom: user ? <UserHeader user={user} showName={true} /> : null
  }), [user]);

  useEffect(() => {
    if (user) {
      setFormData({
        email: user.email || "",
        username: user.username || "",
        fullname: user.fullname || "",
        role: user.role,
      });
      setIsEditing(false);
    }
  }, [user]);

  const handleInputChange = useCallback((field: keyof UpdateUserData, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  }, []);

  const handleSave = useCallback(() => {
    if (user) {
      onUpdate(user.id, formData);
      setIsEditing(false);
    }
  }, [user, formData, onUpdate]);

  const handleCancel = useCallback(() => {
    if (user) {
      setFormData({
        email: user.email || "",
        username: user.username || "",
        fullname: user.fullname || "",
        role: user.role,
      });
    }
    setIsEditing(false);
  }, [user]);

  const handleResetPassword = useCallback((password: string) => {
    if (user && onResetPassword) {
      onResetPassword(user.id, password);
      setShowResetPasswordModal(false);
    }
  }, [user, onResetPassword]);

  const handleClose = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['users'] });
    onClose();
  }, [queryClient, onClose]);

  const handleDelete = useCallback(() => {
    if (user) {
      onDelete(user);
      setShowDeleteConfirm(false);
      handleClose();
    }
  }, [user, onDelete, handleClose]);


  if (!user) return null;

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={handleClose}
        avatar={avatar}
        footer={
          <div className="flex items-center gap-1.5 w-full">
            <UserActions
              isEditing={isEditing}
              onEdit={() => setIsEditing(true)}
              onSave={handleSave}
              onCancel={handleCancel}
              onDelete={() => setShowDeleteConfirm(true)}
              onResetPassword={() => setShowResetPasswordModal(true)}
              loading={loading}
            />
            {!isEditing && (
              <Button variant="default" size="sm" onClick={handleClose} className="h-9 px-5">
                Đóng
              </Button>
            )}
          </div>
        }
      >
        <div className="space-y-3">
          {/* ── Account fields — edit mode shows inputs, view mode shows info rows ── */}
          {isEditing ? (
            <div className="rounded-xl border bg-card p-4 space-y-3">
              <div className="flex items-center gap-2 mb-1">
                <div className="h-5 w-5 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
                  <LucideUser className="h-3 w-3 text-primary" />
                </div>
                <span className="text-xs font-semibold text-foreground">Thông tin tài khoản</span>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label htmlFor="fullname" className="text-xs text-muted-foreground">Họ tên</Label>
                  <Input id="fullname" value={formData.fullname || ""} onChange={(e) => handleInputChange("fullname", e.target.value)} disabled={loading} className="h-9 text-sm" />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="username" className="text-xs text-muted-foreground">Tên đăng nhập</Label>
                  <Input id="username" value={formData.username || ""} onChange={(e) => handleInputChange("username", e.target.value)} disabled={loading} className="h-9 text-sm font-mono" />
                </div>
                <div className="space-y-1 col-span-2">
                  <Label htmlFor="email" className="text-xs text-muted-foreground">Email</Label>
                  <Input id="email" type="email" value={formData.email || ""} onChange={(e) => handleInputChange("email", e.target.value)} disabled={loading} className="h-9 text-sm" />
                </div>
                <div className="space-y-1 col-span-2">
                  <Label htmlFor="role" className="text-xs text-muted-foreground">Vai trò</Label>
                  <Select value={formData.role} onValueChange={(value) => handleInputChange("role", value)} disabled={loading}>
                    <SelectTrigger className="h-9 text-sm"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="admin">Quản trị viên</SelectItem>
                      <SelectItem value="partner">Quản lý</SelectItem>
                      <SelectItem value="employee">Nhân viên</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          ) : (
            <>
              <div className="rounded-xl border bg-card">
                <div className="flex items-baseline justify-between gap-3 px-3 py-2 border-b border-border/50">
                  <span className="text-xs text-muted-foreground shrink-0">Họ tên</span>
                  <span className="text-xs font-medium text-right">{user.fullname || '-'}</span>
                </div>
                <div className="flex items-baseline justify-between gap-3 px-3 py-2 border-b border-border/50">
                  <span className="text-xs text-muted-foreground shrink-0">Tên đăng nhập</span>
                  <span className="text-xs font-mono font-medium text-right">{user.username || '-'}</span>
                </div>
                <div className="flex items-baseline justify-between gap-3 px-3 py-2 border-b border-border/50">
                  <span className="text-xs text-muted-foreground shrink-0">Email</span>
                  <span className="text-xs font-medium text-right">{user.email || '-'}</span>
                </div>
                <div className="flex items-baseline justify-between gap-3 px-3 py-2">
                  <span className="text-xs text-muted-foreground shrink-0">Vai trò</span>
                  <span className="text-xs font-medium text-right">
                    {user.role === 'admin' ? 'Quản trị viên' : user.role === 'partner' ? 'Quản lý' : 'Nhân viên'}
                  </span>
                </div>
              </div>

              {/* Timestamps */}
              <div className="rounded-xl border bg-card grid grid-cols-2 divide-x divide-border/50">
                <div className="flex flex-col gap-0.5 px-3 py-2.5">
                  <span className="text-[10px] text-muted-foreground">Ngày tạo</span>
                  <span className="text-sm font-semibold">{format(new Date(user.created_at), 'dd/MM/yyyy')}</span>
                </div>
                <div className="flex flex-col gap-0.5 px-3 py-2.5">
                  <span className="text-[10px] text-muted-foreground">Cập nhật cuối</span>
                  <span className="text-sm font-semibold">{format(new Date(user.updated_at), 'dd/MM/yyyy')}</span>
                </div>
              </div>

              {/* Activity summary */}
              <div className="space-y-2">
                <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Hoạt động (30 ngày)</p>
                {activityLoading ? (
                  <div className="grid grid-cols-3 gap-2">
                    {Array.from({ length: 3 }).map((_, i) => (
                      <div key={i} className="h-12 rounded-xl bg-muted/40 animate-pulse" />
                    ))}
                  </div>
                ) : (
                  <div className="grid grid-cols-3 gap-2">
                    <div className="flex items-center gap-2.5 rounded-xl border bg-card px-3 py-2.5">
                      <Clock className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        <p className="text-[10px] text-muted-foreground leading-none mb-0.5">Đăng nhập cuối</p>
                        <p className="text-sm font-bold truncate">
                          {activityData?.authentication?.last_login
                            ? formatDistanceToNow(new Date(activityData.authentication.last_login), { addSuffix: true, locale: vi })
                            : '—'}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2.5 rounded-xl border bg-card px-3 py-2.5">
                      <Activity className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        <p className="text-[10px] text-muted-foreground leading-none mb-0.5">Đăng nhập</p>
                        <p className="text-sm font-bold truncate text-blue-600">{activityData?.authentication?.total_logins ?? 0}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2.5 rounded-xl border bg-card px-3 py-2.5">
                      <FileText className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                      <div className="min-w-0">
                        <p className="text-[10px] text-muted-foreground leading-none mb-0.5">Công</p>
                        <p className="text-sm font-bold truncate">{activityData?.payroll_operations?.timesheets_managed ?? 0}</p>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </SlideSheetTemplate>

      <ResetPasswordModal
        open={showResetPasswordModal}
        onClose={() => setShowResetPasswordModal(false)}
        onResetPassword={(_, password) => handleResetPassword(password)}
        user={user}
        loading={loading}
      />

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={() => setShowDeleteConfirm(false)}
        onConfirm={handleDelete}
        title="Xác nhận xóa người dùng"
        description={`Bạn có chắc chắn muốn xóa người dùng "${user.fullname}" (@${user.username})? Hành động này không thể hoàn thành.`}
        confirmText="Xóa"
        cancelText="Hủy"
        confirmVariant="destructive"
      />
    </>
  );
}

export default UserDetailsSheet;
