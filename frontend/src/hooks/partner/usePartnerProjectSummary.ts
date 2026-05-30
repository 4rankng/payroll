import { useQuery } from '@tanstack/react-query';
import { projectService } from '@/services/api/project.service';
import { authManager } from '@/lib/auth';

const QUERY_KEYS = {
  partnerSummary: ['projects', 'partner-summary'] as const,
};

export const usePartnerProjectSummary = () => {
  const userRole = authManager.getUserRole();
  // Only allow if explicitly partner - deny if null or any other role
  const hasPermission = userRole === 'partner' && userRole !== null;

  return useQuery({
    queryKey: QUERY_KEYS.partnerSummary,
    queryFn: () => projectService.getPartnerSummary(),
    enabled: hasPermission,
    retry: false, // Don't retry permission errors
    refetchOnWindowFocus: false,
  });
};