import { useMemo } from 'react';
import { useProjects } from '@/hooks/api/useProjects';

interface UseProjectCodeGeneratorReturn {
  generateCode: () => string;
  isLoading: boolean;
}

export const useProjectCodeGenerator = (): UseProjectCodeGeneratorReturn => {
  const { data: projectsResponse, isLoading } = useProjects({ pageSize: 1000 });

  const generateCode = useMemo(() => {
    return () => {
      if (!projectsResponse?.data) {
        return 'PRJ001';
      }

      const projects = projectsResponse.data;

      // Extract numeric IDs from existing project codes
      const existingIds = projects
        .map(project => {
          if (project.code && project.code.startsWith('PRJ')) {
            const numStr = project.code.replace('PRJ', '');
            const num = parseInt(numStr, 10);
            return isNaN(num) ? 0 : num;
          }
          return 0;
        })
        .filter(id => id > 0)
        .sort((a, b) => b - a); // Sort descending to get the highest first

      // Get the next available ID
      const nextId = existingIds.length > 0 ? existingIds[0] + 1 : 1;

      // Format with leading zeros (e.g., PRJ001, PRJ010, PRJ100)
      return `PRJ${nextId.toString().padStart(3, '0')}`;
    };
  }, [projectsResponse?.data]);

  return {
    generateCode,
    isLoading,
  };
};