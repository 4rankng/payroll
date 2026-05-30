import { useState, useCallback } from 'react';
import { 
  useProjects, 
  useSearchProjects,
  useCreateProject, 
  useUpdateProject, 
  useDeleteProject 
} from '@/hooks/api/useProjects';
import type { ProjectFilters, CreateProjectData, UpdateProjectData } from '@/types/api/project.types';
import { useDebounce } from '@/hooks/useDebounce';

export const useProjectData = () => {
  const [filters, setFilters] = useState<ProjectFilters>({});
  const [searchTerm, setSearchTerm] = useState('');
  
  // Debounce search term to avoid excessive API calls
  const debouncedSearchTerm = useDebounce(searchTerm, 300);
  
  // Use the actual API hooks
  const { data: projectsData, isLoading } = useProjects(filters);
  const { data: searchResults, isLoading: isSearching } = useSearchProjects(
    debouncedSearchTerm.trim() 
      ? { search: debouncedSearchTerm, pageSize: 50, status: filters.status }
      : undefined
  );
  
  const createMutation = useCreateProject();
  const updateMutation = useUpdateProject();
  const deleteMutation = useDeleteProject();

  // Use search results if searching, otherwise use regular projects
  const projects = (debouncedSearchTerm.trim() ? searchResults?.data : projectsData?.data) || [];
  // Only use isLoading (initial page load) for the full-page skeleton.
  // isSearching must NOT be included — it is true on every new search query key,
  // which causes the entire page to unmount/remount on each keystroke.
  const loading = (!debouncedSearchTerm.trim() && isLoading) ||
                 createMutation.isPending || updateMutation.isPending || deleteMutation.isPending;

  const searchProjects = useCallback((search: string) => {
    setSearchTerm(search);
  }, []);

  const filterProjectsByStatus = useCallback((status: string) => {
    setFilters(prev => ({
      ...prev,
      status: status === 'all' ? undefined : status as ProjectFilters['status']
    }));
  }, []);

  const handleCreateProject = useCallback(async (projectData: CreateProjectData): Promise<void> => {
    await createMutation.mutateAsync(projectData);
  }, [createMutation]);

  const handleUpdateProject = useCallback(async (projectId: number, projectData: UpdateProjectData): Promise<void> => {
    await updateMutation.mutateAsync({ id: projectId, data: projectData });
  }, [updateMutation]);

  const handleDeleteProject = useCallback(async (projectId: number): Promise<void> => {
    await deleteMutation.mutateAsync(projectId);
  }, [deleteMutation]);

  return {
    projects,
    isLoading: loading,
    searchProjects,
    filterProjectsByStatus,
    handleCreateProject,
    handleUpdateProject,
    handleDeleteProject,
  };
};