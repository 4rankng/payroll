import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { projectService } from '@/services/api/project.service';
import { useDebounce } from '@/hooks/useDebounce';
import { QueryKeys } from '@/lib/queryKeys';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';
import { invalidateCache } from '@/lib/cache/invalidationService';
import { authManager } from '@/lib/auth';
import type {
  CreateProjectData,
  UpdateProjectData,
  ProjectFilters,
  AssignEmployeeData,
  CreatePayRateData,
  Project,
  ProjectsListResponse,
  ProjectUser,
  GrantProjectAccessData,
  ProjectTimesheetFilters,
} from '@/types/api/project.types';
import type { AdvancedAssignmentFilters } from '@/types/api/project-employee.types';
import type { Employee, EmployeesResponse, CurrentProject, EmployeeProjectsResponse } from '@/types/api/employee.types';

// Get projects summary
export const useProjectsSummary = () => {
  const userRole = authManager.getUserRole();
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.projects.summary(),
    queryFn: () => projectService.getSummary(),
    enabled: hasPermission,
    retry: false,
  });
};

// Get paginated projects list
export const useProjects = (filters?: ProjectFilters, options?: { enabled?: boolean }) => {
  const userRole = authManager.getUserRole();
  // Only allow if explicitly admin or partner - deny if null, employee, or any other role
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.projects.list(filters),
    queryFn: async () => {
      const result = await projectService.getProjects(filters);
      return result;
    },
    enabled: options?.enabled !== undefined ? (options.enabled && hasPermission) : hasPermission,
    retry: false, // Don't retry permission errors
  });
};

// Search projects with autocomplete and debouncing (500ms delay)
export const useProjectSearch = (search: string, enabled: boolean = true) => {
  const debouncedSearch = useDebounce(search, 500);
  
  return useQuery({
    queryKey: QueryKeys.projects.search({ search: debouncedSearch }),
    queryFn: () => projectService.searchProjects({ search: debouncedSearch, pageSize: 10 }),
    enabled: enabled && debouncedSearch.length >= 3, // Minimum 3 characters
    refetchOnWindowFocus: false,
  });
};

// Search projects with debouncing (500ms delay)
export const useSearchProjects = (params?: { search: string; pageSize?: number; status?: string | string[] }) => {
  const debouncedSearch = useDebounce(params?.search || '', 500);
  const debouncedParams = params ? { ...params, search: debouncedSearch } : undefined;

  return useQuery({
    queryKey: QueryKeys.projects.search(debouncedParams as { search: string; pageSize?: number; status?: string; }),
    queryFn: () => debouncedSearch
      ? projectService.searchProjects(debouncedParams!)
      : Promise.resolve({
          status: 'success' as const,
          data: [],
          pagination: { page: 1, pageSize: 50, totalPages: 0, totalRecords: 0 }
        }),
    enabled: Boolean(debouncedSearch && debouncedSearch.trim().length > 0),
  });
};

// Get assignable projects (draft, active, and paused) for project assignment
export const useAssignableProjects = () => {
  const userRole = authManager.getUserRole();
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.projects.list({ status: ['draft', 'active', 'paused'], pageSize: 50 }),
    queryFn: () => projectService.getProjects({ status: ['draft', 'active', 'paused'], pageSize: 50 }),
    enabled: hasPermission,
    retry: false,
  });
};

// Get projects available for timesheet creation (only active projects)
export const useTimesheetProjects = (options?: { enabled?: boolean }) => {
  const userRole = authManager.getUserRole();
  const hasPermission = (userRole === 'admin' || userRole === 'partner') && userRole !== null;

  return useQuery({
    queryKey: QueryKeys.projects.list({ status: ['active'], pageSize: 50 }),
    queryFn: () => projectService.getProjects({ status: ['active'], pageSize: 50 }),
    enabled: hasPermission && (options?.enabled !== undefined ? options.enabled : true),
    retry: false,
  });
};

// Get single project
export const useProject = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.projects.detail(id),
    queryFn: () => projectService.getProjectById(id),
    enabled,
  });
};

// Get project employees
export const useProjectEmployees = (projectId: number, filters?: AdvancedAssignmentFilters, enabled = true) => {
  return useQuery({
    queryKey: [...QueryKeys.projects.employees(projectId), filters],
    queryFn: () => projectService.getProjectEmployees(projectId, filters),
    enabled,
  });
};

// Get project employees for simple filtering (returns minimal employee data)
export const useProjectEmployeesSimple = (projectId: number, enabled = true) => {
  return useQuery({
    queryKey: [...QueryKeys.projects.employees(projectId), 'simple'],
    queryFn: () => projectService.getProjectEmployees(projectId, { pageSize: 100 }),
    enabled: enabled && !!projectId && projectId !== 0,
    select: (data) => {
      if (!data?.data) return [];
      const items = Array.isArray(data.data) ? data.data : (data.data as { data?: unknown[] })?.data || [];
      return (items as import('@/types/api/project.types').ProjectEmployee[]).map(emp => ({
        id: emp.employee_id,
        fullname: emp.employee_name,
        cccd: emp.employee_cccd,
        email: '',
        username: '',
        status: 'active' as const,
        created_at: '',
        updated_at: '',
        current_projects: []
      }));
    }
  });
};

// Create project
export const useCreateProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateProjectData) => projectService.createProject(data),
    onSuccess: async (response) => {
      const newProject = response.data!;
      // Set the new project in cache for detail views
      if (newProject.id) {
        queryClient.setQueryData(QueryKeys.projects.detail(newProject.id), newProject);
      }

      // Update the projects list cache immediately with backend response
      queryClient.setQueriesData(
        { queryKey: QueryKeys.projects.lists() },
        (oldData: ProjectsListResponse | undefined) => {
          if (oldData?.data) {
            return {
              ...oldData,
              data: [newProject, ...oldData.data], // Add new project at the beginning
              pagination: oldData.pagination ? {
                ...oldData.pagination,
                totalRecords: ((oldData.pagination as unknown as { totalRecords?: number }).totalRecords ?? oldData.pagination.total ?? 0) + 1
              } : undefined
            };
          }
          return oldData;
        }
      );

      // Also invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.lists() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.summary() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Update project
export const useUpdateProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateProjectData }) =>
      projectService.updateProject(id, data),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      // Optimistically update the project detail first
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );

      // Use new invalidation service for related cache updates
      await invalidateCache('project:update', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Start project
export const useStartProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.startProject(id),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );
      await invalidateCache('project:start', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });
      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Complete project
export const useCompleteProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.completeProject(id),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      // Optimistically update the project detail first
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );

      // Use new invalidation service for related cache updates
      await invalidateCache('project:complete', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Pause project
export const usePauseProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.pauseProject(id),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      // Optimistically update the project detail first
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );

      // Use new invalidation service for related cache updates
      await invalidateCache('project:update', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Resume project
export const useResumeProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.resumeProject(id),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      // Optimistically update the project detail first
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );

      // Use new invalidation service for related cache updates
      await invalidateCache('project:update', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Cancel project
export const useCancelProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.cancelProject(id),
    onSuccess: async (response) => {
      const updatedProject = response.data!;
      // Optimistically update the project detail first
      queryClient.setQueryData(
        QueryKeys.projects.detail(updatedProject.id),
        updatedProject
      );

      // Use new invalidation service for related cache updates
      await invalidateCache('project:cancel', {
        projectId: updatedProject.id,
        data: updatedProject as unknown as { employee_id?: number; project_id?: number; [key: string]: unknown },
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Delete project
export const useDeleteProject = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => projectService.deleteProject(id),
    onSuccess: async (response, deletedId) => {
      // Remove the project detail query immediately
      queryClient.removeQueries({ queryKey: QueryKeys.projects.detail(deletedId) });

      // Use new invalidation service for comprehensive cleanup
      await invalidateCache('project:delete', {
        projectId: deletedId,
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Assign employee to project
export const useAssignEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, data, selectedProject }: {
      projectId: number;
      data: AssignEmployeeData;
      selectedProject?: Project;
    }) =>
      projectService.assignEmployee(projectId, [data]), // Convert single assignment to array
    onSuccess: (response, variables) => {
      const { projectId, data, selectedProject } = variables;
      const employeeId = data.employee_id;

      // The API returns the created assignment(s)
      if (response.data && Array.isArray(response.data.data) && response.data.data.length > 0) {
        const assignment = (response.data as { data: import('@/types/api/project.types').ProjectAssignmentItem[] }).data[0];

        // Create new project entry for current_projects array
        const newCurrentProject = {
          project_id: projectId,
          name: selectedProject?.name || `Project ${projectId}`,
          code: selectedProject?.code || `PRJ${projectId}`,
          client_name: selectedProject?.client_name || '',
          position: assignment.position,
          start_date: assignment.start_date,
          last_date: assignment.last_date || null
        };

        // Optimistically update employee detail query
        queryClient.setQueryData(['employees', 'detail', employeeId], (oldEmployee: Employee | undefined) => {
          if (!oldEmployee) return oldEmployee;

          const existingProjects = oldEmployee.current_projects || [];
          // Check if already assigned to this project
          const alreadyAssigned = existingProjects.some((p: CurrentProject) => p.project_id === projectId);

          return {
            ...oldEmployee,
            current_projects: alreadyAssigned
              ? existingProjects.map((p: CurrentProject) =>
                  p.project_id === projectId ? newCurrentProject : p
                )
              : [newCurrentProject, ...existingProjects]
          };
        });

        // Optimistically update employee projects list
        queryClient.setQueryData(['employees', employeeId, 'projects'], (oldProjects: EmployeeProjectsResponse | undefined) => {
          if (!oldProjects) return { data: [assignment] };

          const existingData = oldProjects.data || [];
          // Check if assignment already exists (shouldn't happen, but safe)
          const alreadyExists = existingData.some((p) => p.project_id === projectId);

          if (!alreadyExists) {
            return {
              ...oldProjects,
              data: [assignment, ...existingData]
            };
          }
          return oldProjects;
        });

        // Update all employee list queries to reflect the assignment
        queryClient.setQueriesData(
          {
            predicate: (query) => {
              const key = query.queryKey;
              return key[0] === 'employees' && key[1] === 'list';
            }
          },
          (oldData: EmployeesResponse | undefined) => {
            if (!oldData?.data) return oldData;

            return {
              ...oldData,
              data: oldData.data.map((emp: Employee) => {
                if (emp.id === employeeId) {
                  const existingProjects = emp.current_projects || [];
                  const alreadyAssigned = existingProjects.some((p: CurrentProject) => p.project_id === projectId);

                  return {
                    ...emp,
                    current_projects: alreadyAssigned
                      ? existingProjects.map((p: CurrentProject) =>
                          p.project_id === projectId ? newCurrentProject : p
                        )
                      : [newCurrentProject, ...existingProjects]
                  };
                }
                return emp;
              })
            };
          }
        );
      }

      // Still invalidate project employees to refresh project view
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.employees(projectId) });

      // Also invalidate employee data to ensure fresh data
      queryClient.invalidateQueries({ queryKey: ['employees', 'detail', employeeId] });
      queryClient.invalidateQueries({ queryKey: ['employees', employeeId, 'projects'] });
      // Invalidate all employee list queries (with any filters) to update employee table
      // This will automatically invalidate ['employees', 'list', filters] due to fuzzy matching
      queryClient.invalidateQueries({
        queryKey: ['employees', 'list']
      });
      // Also invalidate search queries that might include this employee
      queryClient.invalidateQueries({
        queryKey: ['employees', 'search']
      });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Remove employee from project
export const useRemoveEmployee = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      projectId,
      employeeId,
      lastDate,
    }: {
      projectId: number;
      employeeId: number;
      lastDate?: string;
    }) => projectService.removeEmployee(projectId, employeeId, lastDate),
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.employees(variables.projectId) });
      // Invalidate employee data to refresh current project status
      queryClient.invalidateQueries({ queryKey: ['employees'] });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Create pay rate
export const useCreatePayRate = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, data }: { projectId: number; data: CreatePayRateData }) =>
      projectService.createPayRate(projectId, data),
    onSuccess: (response, variables) => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.payrates(variables.projectId) });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Check for approved timesheets (to prevent project deletion)
export const useProjectApprovedTimesheets = (projectId: number, enabled = true) => {
  return useQuery({
    queryKey: [...QueryKeys.projects.timesheets(projectId), 'approved'],
    queryFn: () => projectService.getProjectTimesheets(projectId, {
      status: 'approved',
      pageSize: 1 // We only need to know if any exist
    }),
    enabled: enabled && !!projectId,
  });
};

// Get project timesheets with flexible filtering
export const useProjectTimesheets = (
  projectId: number,
  filters?: ProjectTimesheetFilters,
  enabled = true
) => {
  return useQuery({
    queryKey: [...QueryKeys.projects.timesheets(projectId), filters],
    queryFn: () => projectService.getProjectTimesheets(projectId, filters),
    enabled: enabled && !!projectId,
  });
};

// Get project users (who have access to the project)
export const useProjectUsers = (projectId: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.projects.users(projectId),
    queryFn: () => projectService.getProjectUsers(projectId),
    enabled: enabled && !!projectId,
  });
};

// Grant access to a project for a specific user
export const useGrantProjectAccess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, data }: { projectId: number; data: GrantProjectAccessData }) =>
      projectService.grantProjectAccess(projectId, data),
    onSuccess: async (response, variables) => {
      // Invalidate and refetch project users query to refresh the list immediately
      await queryClient.invalidateQueries({ queryKey: QueryKeys.projects.users(variables.projectId) });
      // Force a refetch to ensure fresh data
      await queryClient.refetchQueries({ queryKey: QueryKeys.projects.users(variables.projectId) });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};

// Revoke access to a project from a specific user
export const useRevokeProjectAccess = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, userId }: { projectId: number; userId: number }) =>
      projectService.revokeProjectAccess(projectId, userId),
    onSuccess: (response, variables) => {
      // Optimistically remove user from the project users list
      queryClient.setQueryData(
        QueryKeys.projects.users(variables.projectId),
        (oldUsers: ProjectUser[] | undefined) => {
          if (!oldUsers) return oldUsers;
          return oldUsers.filter(user => user.user_id !== variables.userId);
        }
      );

      // Also invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: QueryKeys.projects.users(variables.projectId) });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query
  });
};