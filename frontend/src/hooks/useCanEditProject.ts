import { useMemo } from 'react';
import { Project, ProjectUser } from '@/types/api/project.types';
import { useProjectUsers } from '@/hooks/api/useProjects';
import { authManager } from '@/lib/auth';
import { canPartnerManageProject } from '@/lib/permissions';

/**
 * Hook exposing a project's access list together with whether the current
 * user may manage it (PAYR-30). The access list doubles as the
 * management-scope check: the API returns the creator (unless an admin
 * revoked their owner access) plus every granted manager, so a partner-admin
 * — creator OR granted manager — is exactly a user listed here.
 * Both the details sheet and the access tab must gate on this same signal.
 */
export function useProjectAccessManagement(project?: Project | null): {
  projectUsers: ProjectUser[];
  isLoadingUsers: boolean;
  canManageProjectAccess: boolean;
} {
  const currentUserId = authManager.getUserId();
  const isAdmin = authManager.getUserRole() === 'admin';

  const { data: projectUsers = [], isLoading: isLoadingUsers } = useProjectUsers(
    project?.id ?? 0,
    !!project
  );

  const isProjectManager = useMemo(
    () => projectUsers.some((pu) => pu.user_id === currentUserId),
    [projectUsers, currentUserId]
  );

  return {
    projectUsers,
    isLoadingUsers,
    canManageProjectAccess: isAdmin || isProjectManager,
  };
}

/**
 * Hook to check if a project can be edited
 * @param project - The project to check
 * @returns boolean indicating if the project can be edited
 */
export function useCanEditProject(project?: Project | null): boolean {
  return useMemo(() => {
    if (!project) return false;

    // Check user permissions first
    const hasPermission = canPartnerManageProject();
    if (!hasPermission) return false;

    // Check project status - cannot edit completed or cancelled projects
    const canEditStatus = project.status !== 'completed' && project.status !== 'cancelled';

    return canEditStatus;
  }, [project]);
}

/**
 * Hook to check if employees can be added/removed from a project
 * @param project - The project to check
 * @returns boolean indicating if employees can be managed
 */
export function useCanManageProjectEmployees(project?: Project | null): boolean {
  return useMemo(() => {
    if (!project) return false;

    // Check user permissions first
    const hasPermission = canPartnerManageProject();
    if (!hasPermission) return false;

    // Check project status - cannot manage employees for completed or cancelled projects
    // Allow employee management for paused projects
    const canManageStatus = project.status !== 'completed' && project.status !== 'cancelled';

    return canManageStatus;
  }, [project]);
}

/**
 * Hook to check if timesheet entries can be added to a project
 * @param project - The project to check
 * @returns boolean indicating if timesheet entries can be added
 */
export function useCanAddTimesheet(project?: Project | null): boolean {
  return useMemo(() => {
    if (!project) return false;

    // Only allow timesheet entries for active projects
    return project.status === 'active';
  }, [project]);
}
