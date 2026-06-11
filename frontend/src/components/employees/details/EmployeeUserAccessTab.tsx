import { format } from "date-fns";
import { useState, useMemo } from 'react';
import { Employee, EmployeeUser } from '@/types/api/employee.types';
import { useEmployeeUsers, useGrantEmployeeAccess, useRevokeEmployeeAccess } from '@/hooks/api/useEmployees';
import { useUser } from '@/hooks/api/useUsers';
import { UserSelector } from '@/components/ui/user-selector';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { UserPlus, Trash2, Users, ShieldCheck } from 'lucide-react';
import { authManager } from '@/lib/auth';

interface EmployeeUserAccessTabProps {
  employee: Employee;
}

export function EmployeeUserAccessTab({ employee }: EmployeeUserAccessTabProps) {
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [isRevokeModalOpen, setIsRevokeModalOpen] = useState(false);
  const [userToRevoke, setUserToRevoke] = useState<EmployeeUser | null>(null);



  const currentUserId = authManager.getUserId();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';
  const isEmployeeCreator = employee.created_by === currentUserId;
  const canManageEmployeeAccess = isAdmin || isEmployeeCreator;

  // Fetch employee users
  const { data: employeeUsersData, isLoading: isLoadingUsers } = useEmployeeUsers(
    employee.id,
    canManageEmployeeAccess // Fetch if user is admin or employee creator
  );

  const employeeUsers = useMemo(() => employeeUsersData?.data || [], [employeeUsersData?.data]);

  // Fetch employee creator details
  const { data: creatorData, isLoading: isLoadingCreator } = useUser(
    employee.created_by || 0,
    canManageEmployeeAccess && !!employee.created_by // Only fetch if user can manage access and created_by exists
  );

  // Mutations
  const grantAccessMutation = useGrantEmployeeAccess();
  const revokeAccessMutation = useRevokeEmployeeAccess();

  // Combine employee users with creator to show complete access list
  const allEmployeeUsers = useMemo(() => {
    const users: EmployeeUser[] = [...employeeUsers];

    // Check if creator is already in the list
    const creatorInList = users.some(user => user.user_id === employee.created_by);

    // If creator is not in the list and we have creator data, add them
    if (!creatorInList && creatorData && employee.created_by) {
      const creatorUser: EmployeeUser = {
        id: 0, // Use 0 to distinguish creator (API spec says creator has id: 0)
        user_id: employee.created_by,
        user_fullname: creatorData.fullname,
        user_email: creatorData.email || '',
        granted_by: employee.created_by, // Self-granted
        granted_at: employee.created_at || new Date().toISOString(), // Use employee creation time
      };

      // Add creator at the beginning of the list
      users.unshift(creatorUser);
    }

    return users;
  }, [employeeUsers, creatorData, employee.created_by, employee.created_at]);

  // Get user IDs that already have access to exclude from selector
  const excludeUserIds = useMemo(() => {
    const userIds = allEmployeeUsers.map(eu => eu.user_id);
    return userIds;
  }, [allEmployeeUsers]);

  // Handle granting access
  const handleGrantAccess = async () => {
    if (!selectedUserId) return;

    try {
      await grantAccessMutation.mutateAsync({
        employeeId: employee.id,
        data: { user_id: parseInt(selectedUserId) }
      });
      setSelectedUserId(''); // Reset selection
    } catch (error) {
      // Error handling is done globally
    }
  };

  // Handle revoking access
  const handleRevokeAccess = async () => {
    if (!userToRevoke) return;

    try {
      await revokeAccessMutation.mutateAsync({
        employeeId: employee.id,
        userId: userToRevoke.user_id
      });
      setIsRevokeModalOpen(false);
      setUserToRevoke(null);
    } catch (error) {
      // Error handling is done globally
    }
  };

  // Show permission denied if not admin or employee creator
  if (!canManageEmployeeAccess) {
    return (
      <div className="mt-3 rounded-xl border bg-muted/20 p-4">
        <div className="flex flex-col items-center text-center space-y-2">
          <ShieldCheck className="h-8 w-8 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium">Không có quyền truy cập</p>
            <p className="text-xs text-muted-foreground mt-1">
              Chỉ quản trị viên hoặc người tạo nhân viên mới có thể quản lý.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4 mt-3">
      {/* Add User Section */}
      <div className="rounded-xl border bg-muted/20 p-3 space-y-2">
        <div className="flex items-center gap-1.5">
          <UserPlus className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="text-sm font-medium">Cấp quyền quản lý</span>
        </div>
        <p className="text-xs text-muted-foreground">
          Cho phép quản lý khác truy cập và quản lý nhân viên này.
          {isAdmin && !isEmployeeCreator && (
            <span className="block mt-0.5">Bạn có quyền quản lý nhân viên này với tư cách quản trị viên.</span>
          )}
        </p>
        <div className="flex gap-2 pt-1">
          <UserSelector
            value={selectedUserId}
            onValueChange={setSelectedUserId}
            placeholder="Chọn quản lý để cấp quyền..."
            excludeUserIds={excludeUserIds}
            filterRole="partner"
            className="flex-1"
          />
          <Button
            onClick={handleGrantAccess}
            disabled={!selectedUserId || grantAccessMutation.isPending}
            size="sm"
            className="shrink-0"
          >
            {grantAccessMutation.isPending ? (
              <>
                <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin mr-1.5" />
                Đang cấp...
              </>
            ) : (
              <>
                <UserPlus className="w-3.5 h-3.5 mr-1.5" />
                Cấp quyền
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Users List Section */}
      <div className="space-y-2">
        <div className="flex items-center gap-1.5 px-0.5">
          <Users className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="text-sm font-medium">Danh sách người dùng</span>
          <span className="text-xs text-muted-foreground">· có thể quản lý nhân viên này</span>
        </div>
        {isLoadingUsers || isLoadingCreator ? (
          <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
            <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
            Đang tải...
          </div>
        ) : allEmployeeUsers.length === 0 ? (
          <div className="text-center py-6 text-xs text-muted-foreground border rounded-xl">
            Chưa có quản lý nào được cấp quyền.
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-2">
            {allEmployeeUsers.map((employeeUser) => (
              <div
                key={employeeUser.id === 0 ? `creator-${employeeUser.user_id}` : employeeUser.id}
                className="flex items-center justify-between px-3 py-2.5 border rounded-xl hover:bg-accent/50 transition-colors"
              >
                <div className="flex items-center gap-2.5">
                  <UserAvatar
                    email={employeeUser.user_email}
                    name={employeeUser.user_fullname}
                    size="sm"
                  />
                  <div>
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm font-medium leading-tight">
                        {employeeUser.user_fullname}
                      </span>
                      {employeeUser.user_id === employee.created_by && (
                        <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                          Người tạo
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{employeeUser.user_email}</p>
                    <p className="text-[10px] text-muted-foreground">
                      Cấp quyền: {format(new Date(employeeUser.granted_at), 'dd/MM/yyyy')}
                    </p>
                  </div>
                </div>

                {employeeUser.user_id !== employee.created_by && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setUserToRevoke(employeeUser);
                      setIsRevokeModalOpen(true);
                    }}
                    disabled={revokeAccessMutation.isPending}
                    className="text-destructive hover:text-destructive h-7 w-7 p-0"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Revoke Access Confirmation Modal */}
      <ConfirmDialog
        open={isRevokeModalOpen}
        onOpenChange={() => {
          setIsRevokeModalOpen(false);
          setUserToRevoke(null);
        }}
        onConfirm={handleRevokeAccess}
        title="Thu hồi quyền truy cập"
        description={
          <div>
            <p>
              Bạn có chắc chắn muốn thu hồi quyền truy cập nhân viên này từ người quản lý{' '}
              <strong>{userToRevoke?.user_fullname}</strong>?
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
              Quản lý sẽ không thể truy cập nhân viên này nữa.
            </p>
          </div>
        }
        confirmText="Thu hồi quyền"
        cancelText="Hủy bỏ"
        confirmVariant="destructive"
        loading={revokeAccessMutation.isPending}
      />
    </div>
  );
}
