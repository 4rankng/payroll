import { useLocation } from "react-router-dom";
import ProjectDetailsSheet from "@/components/sheets/ProjectDetailsSheet";
import { useProjectData } from "@/hooks/projects/useProjectData";
import { useProject } from "@/hooks/api/useProjects";
import { toast } from "@/components/ui/sonner";
import { useMemo } from "react";
import { useModalNavigation } from "@/hooks/useModalNavigation";

export const modalConfig = {
  id: 'project-details',
};

/**
 * Route-based Project Details Modal
 * Integrates with the secure modal system for deep-linking
 */
export function ProjectDetailsModal() {
  const location = useLocation();
  const { closeModal } = useModalNavigation();
  const projectData = useProjectData();

  // Use location.search for better reactivity to URL changes
  const searchParams = useMemo(() => new URLSearchParams(location.search), [location.search]);

  const modalId = searchParams.get('modal');
  const projectId = searchParams.get('id');
  const isOpen = modalId === 'project_details' && Boolean(projectId);

  // Fetch detailed project data using the individual project endpoint
  const { data: detailedProject, isLoading: projectLoading } = useProject(
    projectId ? parseInt(projectId) : 0,
    isOpen && !!projectId
  );

  // Find the project by ID from URL parameters as fallback
  const selectedProject = useMemo(() => {
    if (!projectId) return null;

    // Use detailed project data if available, otherwise fallback to list data
    if (detailedProject) {
      return detailedProject;
    }

    if (!projectData.projects) return null;

    const project = projectData.projects.find(p => p.id.toString() === projectId);

    // If project not found, show error
    if (!project && projectData.projects.length > 0 && !projectLoading) {
      toast({
        title: "Lỗi",
        description: "Không tìm thấy dự án.",
        variant: "destructive"
      });
    }

    return project || null;
  }, [projectId, detailedProject, projectData.projects, projectLoading]);

  const handleClose = () => {
    closeModal();
  };


  return (
    <ProjectDetailsSheet
      project={selectedProject}
      isOpen={isOpen}
      onClose={handleClose}
    />
  );
}
