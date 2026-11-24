import { format } from "date-fns";
import { useState, useMemo } from 'react';
import { Project, ProjectUser } from '@/types/api/project.types';
import { useGrantProjectAccess, useRevokeProjectAccess, useRequestProjectAccess, useReleaseProjectAccess } from '@/hooks/api/useProjects';
import { useProjectAccessManagement } from '@/hooks/useCanEditProject';
import { UserSelector } from '@/components/ui/user-selector';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { UserPlus, Trash2, Users, ShieldCheck, HandHeart, LogOut } from 'lucide-react';
import { authManager } from '@/lib/auth';

interface ProjectUserAccessTabProps {
  project: Project;
  /** Called after the caller's own grant is released — the host sheet closes
   * itself because the project has left the caller's accessible set. */
  onSelfReleased?: () => void;
}

export function ProjectUserAccessTab({ project, onSelfReleased }: ProjectUserAccessTabProps) {
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [isRevokeModalOpen, setIsRevokeModalOpen] = useState(false);
  const [isReleaseModalOpen, setIsReleaseModalOpen] = useState(false);
  const [userToRevoke, setUserToRevoke] = useState<ProjectUser | null>(null);

  const currentUserId = authManager.getUserId();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';
  const isPartner = userRole === 'partner';
  const isProjectCreator = project.created_by === currentUserId;

  // Shared gate with the details sheet (PAYR-30): a partner-admin is the
  // project creator OR the holder of an explicit project_users grant — both
  // appear in the access list the API returns.
  const { projectUsers, isLoadingUsers, canManageProjectAccess } =
    useProjectAccessManagement(project);

  // Mutations
  const grantAccessMutation = useGrantProjectAccess();
  const revokeAccessMutation = useRevokeProjectAccess();

  // Self-service (owner ruling 2026-11-24): claim grants management
  // immediately; release drops the caller's own grant immediately. Both are
  // audited server-side — no approval step exists for this flow.
  const requestAccessMutation = useRequestProjectAccess();
  const releaseAccessMutation = useReleaseProjectAccess();

  const handleClaimAccess = async () => {
    try {
      await requestAccessMutation.mutateAsync(project.id);
      // The users query invalidates and refetches; the management UI takes
      // over in place (no approval, no waiting state).
    } catch (error) {
      // Error handling is done globally
    }
  };

  const handleReleaseAccess = async () => {
    try {
      await releaseAccessMutation.mutateAsync(project.id);
      setIsReleaseModalOpen(false);
      // The project left this partner's accessible set — let the host sheet
      // close so they are not staring at a detail view they can no longer read.
      onSelfReleased?.();
    } catch (error) {
      // Error handling is done globally
    }
  };

  // Get user IDs that already have access to exclude from selector
  const excludeUserIds = useMemo(() => {
    const userIds = projectUsers.map(pu => pu.user_id);
    return userIds;
  }, [projectUsers]);

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

  if (isLoadingUsers) {
    return (
      <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2 mt-3">
        <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
        Đang tải...
      </div>
    );
  }

  // Show permission denied if the user does not manage this project —
  // partners get the instant self-service claim (no approval step).
  if (!canManageProjectAccess) {
    return (
      <div className="mt-3 console-panel p-4">
        <div className="flex flex-col items-center text-center space-y-2">
          <ShieldCheck className="h-8 w-8 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium">Không có quyền truy cập</p>
            <p className="text-xs text-muted-foreground mt-1">
              Chỉ quản trị viên hoặc người quản lý dự án mới có thể quản lý.
            </p>
          </div>
          {isPartner && (
            <Button
              size="sm"
              onClick={handleClaimAccess}
              disabled={requestAccessMutation.isPending}
              className="mt-1 gap-1.5"
            >
              {requestAccessMutation.isPending ? (
                <>
                  <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
                  Đang xử lý...
                </>
              ) : (
                <>
                  <HandHeart className="h-3.5 w-3.5" />
                  Yêu cầu quản lý dự án
                </>
              )}
            </Button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4 mt-3">
      {/* Add User Section */}
      <div className="console-panel p-3 space-y-2">
        <div className="flex items-center gap-1.5">
          <UserPlus className="h-3.5 w-3.5 text-muted-foreground" />
          <span className="console-kicker">Cấp quyền quản lý</span>
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
          <span className="console-kicker">Danh sách người dùng</span>
          <span className="text-xs text-muted-foreground">· có thể quản lý dự án này</span>
        </div>
        {isLoadingUsers ? (
          <div className="flex items-center justify-center py-6 text-xs text-muted-foreground gap-2">
            <div className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
            Đang tải...
          </div>
        ) : projectUsers.length === 0 ? (
          <div className="console-panel text-center py-6 text-xs text-muted-foreground">
            Chưa có người dùng nào được cấp quyền quản lý.
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-2">
            {projectUsers.map((projectUser) => (
              <div
                key={projectUser.id}
                className="console-panel flex items-center justify-between px-3 py-2.5 hover:bg-accent/50 transition-colors"
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
                        <Badge variant="secondary" className="text-xs px-1.5 py-0">
                          Người tạo
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{projectUser.user_email}</p>
                    <p className="text-xs text-muted-foreground">
                      Cấp quyền: {format(new Date(projectUser.granted_at), 'dd/MM/yyyy')}
                    </p>
                  </div>
                </div>

                {(projectUser.user_id !== project.created_by || isAdmin) && (
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

      {/* Self-release (owner ruling 2026-11-24): a granted partner — not the
          creator — may drop their own management grant instantly. */}
      {!isAdmin && !isProjectCreator && (
        <div className="console-panel p-3 space-y-2">
          <div className="flex items-center gap-1.5">
            <LogOut className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="console-kicker">Rời quản lý dự án</span>
          </div>
          <p className="text-xs text-muted-foreground">
            Bạn sẽ không còn quản lý hoặc xem dự án này trong danh sách của mình.
          </p>
          <div className="pt-1">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsReleaseModalOpen(true)}
              disabled={releaseAccessMutation.isPending}
              className="gap-1.5"
            >
              <LogOut className="h-3.5 w-3.5" />
              Yêu cầu loại khỏi dự án
            </Button>
          </div>
        </div>
      )}

      {/* Release Confirmation Modal */}
      <ConfirmDialog
        open={isReleaseModalOpen}
        onOpenChange={() => setIsReleaseModalOpen(false)}
        onConfirm={handleReleaseAccess}
        title="Yêu cầu loại khỏi dự án"
        description={
          <div>
            <p>
              Bạn có chắc chắn muốn rời quản lý dự án <strong>{project.name}</strong>?
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
              Quyền quản lý của bạn với dự án này được loại ngay lập tức.
            </p>
          </div>
        }
        confirmText="Loại khỏi dự án"
        cancelText="Hủy bỏ"
        confirmVariant="destructive"
        loading={releaseAccessMutation.isPending}
      />

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
