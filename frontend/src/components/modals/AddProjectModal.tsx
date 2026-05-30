import { useSecureModal } from "@/hooks/useSecureModal";
import { useProjectData } from "@/hooks/projects/useProjectData";
import { AddProjectSheet } from "@/components/sheets/AddProjectSheet";
import { CreateProjectData } from "@/types/api/project.types";
import { toast } from "@/components/ui/sonner";
import { useNavigate } from "react-router-dom";
import { useCreateProject } from "@/hooks/api/useProjects";

/**
 * Route-based AddProject Modal
 * Integrates with the secure modal system for deep-linking
 */
export function AddProjectModal() {
  const navigate = useNavigate();
  const projectData = useProjectData();
  const createProjectMutation = useCreateProject();

  const modal = useSecureModal('project_create', {
    requiresAuth: true,
    onError: (error) => {
      console.error('AddProjectModal error:', error);
      toast({
        title: "Lỗi",
        description: "Có lỗi xảy ra khi tải modal.",
        variant: "destructive"
      });
    }
  });

  const handleProjectCreate = async (projectData: CreateProjectData) => {
    createProjectMutation.mutate(projectData, {
      onSuccess: () => {
        // Close modal by navigating back
        modal.close();
      }
    });
  };

  const handleClose = () => {
    modal.close();
    navigate(-1); // Go back in history
  };

  return (
    <AddProjectSheet
      isOpen={modal.isOpen}
      onClose={handleClose}
      onProjectCreate={handleProjectCreate}
    />
  );
}