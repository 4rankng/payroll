import { useMemo } from 'react';
import { useProjects } from '@/hooks/api/useProjects';
import { PAGINATION_DEFAULTS } from '@/config/api.config';

export const usePartnerDashboardData = () => {
  // Use React Query hook to fetch projects with pagination
  const { data: projectsResponse, isLoading: loading, refetch: loadProjects } = useProjects({
    page: 1,
    pageSize: PAGINATION_DEFAULTS.maxPageSize, // Get all projects for partner dashboard
  });

  // Extract projects from API response
  const projects = useMemo(() => {
    return projectsResponse?.data || [];
  }, [projectsResponse?.data]);

  // Calculate stats from projects
  const stats = useMemo(() => {
    const totalEmployees = projects.reduce((sum, project) => sum + (project.employee_count || 0), 0);

    return {
      totalProjects: projects.length,
      totalEmployees,
    };
  }, [projects]);

  return {
    projects,
    stats,
    loading,
    loadProjects,
    pagination: projectsResponse?.pagination,
  };
};
