import { useState, useEffect, useCallback } from "react";
import { projectService } from "@/services/api/project.service";
import { Project, ProjectEmployee } from "@/types/api/project.types";

export const usePartnerProject = (projectId: string) => {
  const [project, setProject] = useState<Project | null>(null);
  const [employees, setEmployees] = useState<ProjectEmployee[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const loadProject = useCallback(async () => {
    if (!projectId) return;
    
    setIsLoading(true);
    try {
      const [projectResponse, employeesResponse] = await Promise.all([
        projectService.getProjectById(parseInt(projectId)),
        projectService.getProjectEmployees(parseInt(projectId))
      ]);
      
      setProject(projectResponse);
      if (employeesResponse.success && employeesResponse.data) {
        setEmployees(employeesResponse.data);
      }
    } catch (error) {
      console.error('Failed to load project:', error);
    } finally {
      setIsLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    loadProject();
  }, [projectId, loadProject]);

  const refreshProject = async (): Promise<void> => {
    await loadProject();
  };

  return {
    project,
    employees,
    isLoading,
    refreshProject,
  };
};
