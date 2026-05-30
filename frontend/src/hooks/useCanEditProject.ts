import { useMemo } from 'react';
import { Project } from '@/types/api/project.types';
import { canPartnerManageProject } from '@/lib/permissions';

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
