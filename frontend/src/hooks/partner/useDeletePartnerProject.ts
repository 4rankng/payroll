import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { projectService } from "@/services/api/project.service";
import { toast } from "@/components/ui/sonner";
import { Project, ProjectsListResponse, PartnerProjectSummary } from "@/types/api/project.types";

export const useDeletePartnerProject = () => {
  const [isDeleting, setIsDeleting] = useState(false);
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: async (projectId: number): Promise<void> => {
      await projectService.deleteProject(projectId);
    },
    onMutate: async (projectId) => {
      setIsDeleting(true);

      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['projects'] });
      await queryClient.cancelQueries({ queryKey: ['project', projectId] });

      // No optimistic updates - wait for API response
      return {};
    },
    onError: (error) => {
      console.error('Failed to delete project:', error);

      toast({
        title: "Lỗi xóa dự án",
        description: "Không thể xóa dự án. Vui lòng thử lại.",
        variant: "destructive",
      });
      setIsDeleting(false);
    },
    onSuccess: (_, projectId) => {
      // Remove from cache completely
      queryClient.removeQueries({ queryKey: ['project', projectId] });

      // Invalidate projects list to refresh data from server
      queryClient.invalidateQueries({ queryKey: ['projects'] });
      queryClient.invalidateQueries({ queryKey: ['projects', 'partner-summary'] });

      toast({
        title: "Xóa thành công",
        description: "Dự án đã được xóa khỏi hệ thống.",
        variant: "default",
      });
      setIsDeleting(false);
    },
    onSettled: () => {
      setIsDeleting(false);
    }
  });

  const deleteProject = async (projectId: number, projectName?: string): Promise<boolean> => {
    try {
      await deleteMutation.mutateAsync(projectId);
      return true;
    } catch (error) {
      console.error('Delete failed:', error);
      return false;
    }
  };

  return {
    deleteProject,
    isLoading: deleteMutation.isPending,
    isDeleting,
    error: deleteMutation.error,
    isSuccess: deleteMutation.isSuccess,
  };
};