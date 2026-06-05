import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { projectService } from "@/services/api/project.service";
import { toast } from "@/components/ui/sonner";
import { UpdateProjectData, Project, UpdateProjectStatusData } from "@/types/api/project.types";
import { updateItemInList, updateSummaryFields } from "@/utils/cacheUpdates";

interface UpdateProjectInput {
  projectId: number;
  data: UpdateProjectData;
}

export const useUpdatePartnerProject = () => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: async ({ projectId, data }: UpdateProjectInput): Promise<Project> => {
      const response = await projectService.updateProject(projectId, data);
      return response.data;
    },
    onMutate: async ({ projectId, data }) => {
      setIsSubmitting(true);

      // Cancel outgoing refetches to prevent race conditions
      await queryClient.cancelQueries({ queryKey: ['project', projectId] });
      await queryClient.cancelQueries({ queryKey: ['projects'] });

      // No optimistic updates - wait for API response
      return {};
    },
    onError: (error) => {
      console.error('Failed to update project:', error);

      toast({
        title: "Lỗi cập nhật dự án",
        description: "Không thể cập nhật thông tin dự án. Vui lòng thử lại.",
        variant: "destructive",
      });
      setIsSubmitting(false);
    },
    onSuccess: (updatedProject, { projectId }) => {
      // Update individual project cache with server response
      queryClient.setQueryData(['project', projectId], updatedProject);

      // Update project in all relevant lists using server response
      updateItemInList(queryClient, ['projects'], updatedProject as Record<string, unknown>);

      // Invalidate summary data to ensure consistency
      queryClient.invalidateQueries({ queryKey: ['projects', 'partner-summary'] });

      toast({
        title: "Cập nhật thành công",
        description: "Thông tin dự án đã được cập nhật.",
        variant: "default",
      });
      setIsSubmitting(false);
    },
    onSettled: () => {
      setIsSubmitting(false);
    }
  });

  // Status update mutation
  const statusUpdateMutation = useMutation({
    mutationFn: async ({ projectId, status }: { projectId: number; status: UpdateProjectStatusData }): Promise<Project> => {
      const response = await projectService.updateProjectStatus(projectId, status);
      return response.data;
    },
    onMutate: async ({ projectId }) => {
      setIsSubmitting(true);

      // Cancel outgoing refetches to prevent race conditions
      await queryClient.cancelQueries({ queryKey: ['project', projectId] });
      await queryClient.cancelQueries({ queryKey: ['projects'] });
      await queryClient.cancelQueries({ queryKey: ['projects', 'partner-summary'] });

      // No optimistic updates - wait for API response
      return {};
    },
    onError: (error) => {
      console.error('Failed to update project status:', error);

      toast({
        title: "Lỗi cập nhật trạng thái",
        description: "Không thể cập nhật trạng thái dự án. Vui lòng thử lại.",
        variant: "destructive",
      });
      setIsSubmitting(false);
    },
    onSuccess: (updatedProject, { projectId }) => {
      // Update individual project cache with server response
      queryClient.setQueryData(['project', projectId], updatedProject);

      // Update project in all relevant lists using server response
      updateItemInList(queryClient, ['projects'], updatedProject as Record<string, unknown>);

      // Refresh summary data with server state
      queryClient.invalidateQueries({ queryKey: ['projects', 'partner-summary'] });

      toast({
        title: "Cập nhật thành công",
        description: "Trạng thái dự án đã được cập nhật.",
        variant: "default",
      });
      setIsSubmitting(false);
    },
    onSettled: () => {
      setIsSubmitting(false);
    }
  });

  const updateProject = async (projectId: number, data: UpdateProjectData): Promise<Project | null> => {
    try {
      const result = await updateMutation.mutateAsync({ projectId, data });
      return result;
    } catch (error) {
      console.error('Update failed:', error);
      return null;
    }
  };

  const updateProjectStatus = async (projectId: number, status: UpdateProjectStatusData): Promise<Project | null> => {
    try {
      const result = await statusUpdateMutation.mutateAsync({ projectId, status });
      return result;
    } catch (error) {
      console.error('Status update failed:', error);
      return null;
    }
  };

  return {
    updateProject,
    updateProjectStatus,
    isLoading: updateMutation.isPending || statusUpdateMutation.isPending,
    isSubmitting,
    error: updateMutation.error || statusUpdateMutation.error,
    isSuccess: updateMutation.isSuccess || statusUpdateMutation.isSuccess,
  };
};