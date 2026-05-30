import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useModalNavigation } from '@/hooks/useSecureModal';

export const usePartnerDashboardActions = () => {
  const navigate = useNavigate();
  const { navigateToModal } = useModalNavigation();

  const handleViewProject = useCallback((projectId: number) => {
    navigate(`/partner/project/${projectId}`);
  }, [navigate]);

  const handleAddProject = useCallback(() => {
    // Partners can now create projects - open the create project modal
    navigateToModal('project_create');
  }, [navigateToModal]);

  const handleEditProject = useCallback((projectId: number) => {
    // Partners can now edit projects - open the edit project modal
    navigateToModal('edit_project', { projectId });
  }, [navigateToModal]);

  return {
    handleViewProject,
    handleAddProject,
    handleEditProject,
  };
};