import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { projectService } from "@/services/api/project.service";
import { toast } from "@/components/ui/sonner";
import { CreateProjectData, Project, ProjectsListResponse } from "@/types/api/project.types";
import { updateSummaryCount } from "@/utils/cacheUpdates";

export const useCreatePartnerProject = () => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: async (data: CreateProjectData): Promise<{ project: Project; message: string }> => {
      // Determine status based on start date
      const today = new Date();
      const startDate = new Date(data.start_date);

      // Set status to 'active' if start date is today or earlier, otherwise 'draft'
      const projectData: CreateProjectData = {
        ...data,
        status: startDate <= today ? 'active' : 'draft'
      };

      const response = await projectService.createProject(projectData);
      return { project: response.data!, message: response.message || "Tạo dự án thành công" };
    },
    onMutate: () => {
      setIsSubmitting(true);
    },
    onError: (error) => {
      console.error('Failed to create project:', error);

      toast({
        title: "Lỗi tạo dự án",
        description: "Không thể tạo dự án mới. Vui lòng thử lại.",
        variant: "destructive",
      });
      setIsSubmitting(false);
    },
    onSuccess: ({ project: newProject, message }) => {
      // Update all project list queries with the new project
      queryClient.setQueriesData(
        { queryKey: ['projects', 'list'] },
        (oldData: ProjectsListResponse | undefined) => {
          if (!oldData) return oldData;

          // Handle API response format with status, data, pagination
          if ('status' in oldData && 'data' in oldData && Array.isArray(oldData.data)) {
            return {
              ...oldData,
              data: [newProject, ...oldData.data],
              pagination: oldData.pagination ? {
                ...oldData.pagination,
                total: oldData.pagination.total + 1
              } : undefined
            };
          }

          return oldData;
        }
      );

      // Set the new project in cache for detail views
      queryClient.setQueryData(['projects', 'detail', newProject.id], newProject);

      // Invalidate project lists to ensure consistency
      queryClient.invalidateQueries({ queryKey: ['projects', 'list'] });

      // Update summary data using server response
      updateSummaryCount(queryClient, ['projects', 'partner-summary'], 'total_projects', 1);
      if (newProject.status === 'active') {
        updateSummaryCount(queryClient, ['projects', 'partner-summary'], 'active_projects', 1);
      }
      if (newProject.status === 'completed') {
        updateSummaryCount(queryClient, ['projects', 'partner-summary'], 'completed_projects', 1);
      }

      toast({
        title: message,
        description: `Dự án "${newProject.name}" đã được tạo thành công.`,
        variant: "default",
      });
      setIsSubmitting(false);
    },
    onSettled: () => {
      setIsSubmitting(false);
    }
  });

  const createProject = async (data: CreateProjectData): Promise<Project | null> => {
    try {
      const result = await createMutation.mutateAsync(data);
      return result.project;
    } catch (error) {
      console.error('Create failed:', error);
      return null;
    }
  };

  return {
    createProject,
    isLoading: createMutation.isPending,
    isSubmitting,
    error: createMutation.error,
    isSuccess: createMutation.isSuccess,
  };
};