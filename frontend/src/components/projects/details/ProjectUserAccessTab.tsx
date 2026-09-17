import { format } from "date-fns";
import { useState, useMemo } from 'react';
import { Project, ProjectUser } from '@/types/api/project.types';
import { User } from '@/types/user';
import { useProjectUsers, useGrantProjectAccess, useRevokeProjectAccess } from '@/hooks/api/useProjects';
import { useUser } from '@/hooks/api/useUsers';
import { UserSelector } from '@/components/ui/user-selector';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { UserPlus, Trash2, Users, ShieldCheck } from 'lucide-react';
import { authManager } from '@/lib/auth';

interface ProjectUserAccessTabProps {
  project: Project;
}

export function ProjectUserAccessTab({ project }: ProjectUserAccessTabProps) {
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [isRevokeModalOpen, setIsRevokeModalOpen] = useState(false);
  const [userToRevoke, setUserToRevoke] = useState<ProjectUser | null>(null);

  const currentUserId = authManager.getUserId();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';
  const isProjectCreator = project.created_by === currentUserId;
  const canManageProjectAccess = isAdmin || isProjectCreator;

  // Fetch project users
  const { data: projectUsers = [], isLoading: isLoadingUsers } = useProjectUsers(
    project.id,
    canManageProjectAccess // Fetch if user is admin or project creator
  );

  // Fetch project creator details
  const { data: creatorData, isLoading: isLoadingCreator } = useUser(
    project.created_by,
    canManageProjectAccess // Only fetch if user can manage access
  );

  // Mutations
  const grantAccessMutation = useGrantProjectAccess();
  const revokeAccessMutation = useRevokeProjectAccess();

  // Combine project users with creator to show complete access list
  const allProjectUsers = useMemo(() => {
    const users: ProjectUser[] = [...projectUsers];

    // Check if creator is already in the list
    const creatorInList = users.some(user => user.user_id === project.created_by);

    // If creator is not in the list and we have creator data, add them
    if (!creatorInList && creatorData) {
      const creatorUser: ProjectUser = {
        id: -1, // Use negative ID to distinguish from API users
        user_id: project.created_by,
        user_fullname: creatorData.fullname,
        user_email: creatorData.email || '',
        granted_by: project.created_by, // Self-granted
        granted_at: project.created_at, // Use project creation time
      };

      // Add creator at the beginning of the list
      users.unshift(creatorUser);
    }

    return users;
  }, [projectUsers, creatorData, project.created_by, project.created_at]);

  // Get user IDs that already have access to exclude from selector
  const excludeUserIds = useMemo(() => {
    const userIds = allProjectUsers.map(pu => pu.user_id);
    return userIds;
  }, [allProjectUsers]);

  // Handle granting access
  const handleGrantAccess = async () => {
    if (!selectedUserId) return;

    try {
      await grantAccessMutation.mutateAsync({
        projectId: project.id,
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
        projectId: project.id,
        userId: userToRevoke.user_id
      });
      setIsRevokeModalOpen(false);
      setUserToRevoke(null);
    } catch (error) {
      // Error handling is done globally
    }
  };

  // Show permission denied if not admin or project creator
  if (!canManageProjectAccess) {
    return (
      <div className="mt-3 rounded-xl border bg-muted/20 p-4">
        <div className="flex flex-col items-center text-center space-y-2">
          <ShieldCheck className="h-8 w-8 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium">Không có quyền truy cập</p>
            <p className="text-xs text-muted-foreground mt-1">
              Chỉ admin hoặc người tạo dự án mới có thể quản lý.
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
          Cho phép người dùng khác truy cập và quản lý dự án này.
          {isAdmin && !isProjectCreator && (
            <span className="block mt-0.5">Bạn có quyền quản lý dự án này với tư cách quản trị viên.</span>
          )}
        </p>
        <div className="flex gap-2 pt-1">
          <UserSelector
            value={selectedUserId}
            onValueChange={setSelectedUserId}
            placeholder="Chọn người dùng để cấp quyền..."
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
          <span className="text-xs text-muted-foreground">· có thể quản lý dự án này</span>
        </div>
        {isLoadingUsers || isLoadingCreator ? (
          <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
            <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
            Đang tải...
          </div>
        ) : allProjectUsers.length === 0 ? (
          <div className="text-center py-6 text-xs text-muted-foreground border bg-muted/30 rounded-xl">
            Chưa có người dùng nào được cấp quyền quản lý.
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-2">
            {allProjectUsers.map((projectUser) => (
              <div
                key={projectUser.id === -1 ? `creator-${projectUser.user_id}` : projectUser.id}
                className="flex items-center justify-between px-3 py-2.5 border rounded-xl hover:bg-accent/50 transition-colors"
              >
                <div className="flex items-center gap-2.5">
                  <UserAvatar
                    email={projectUser.user_email}
                    name={projectUser.user_fullname}
                    size="sm"
                  />
                  <div>
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm font-medium leading-tight">
                        {projectUser.user_fullname}
                      </span>
                      {projectUser.user_id === project.created_by && (
                        <Badge variant="secondary" className="text-[11px] px-1.5 py-0">
                          Người tạo
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{projectUser.user_email}</p>
                    <p className="text-[11px] text-muted-foreground">
                      Cấp quyền: {format(new Date(projectUser.granted_at), 'dd/MM/yyyy')}
                    </p>
                  </div>
                </div>

                {projectUser.user_id !== project.created_by && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setUserToRevoke(projectUser);
                      setIsRevokeModalOpen(true);
                    }}
                    disabled={revokeAccessMutation.isPending}
                    aria-label={`Thu hồi quyền của ${projectUser.user_fullname}`}
                    className="text-destructive hover:text-destructive h-11 w-11 shrink-0 p-0"
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
              Bạn có chắc chắn muốn thu hồi quyền truy cập dự án này từ người dùng{' '}
              <strong>{userToRevoke?.user_fullname}</strong>?
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
              Người dùng sẽ không thể truy cập hoặc quản lý dự án này nữa.
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
