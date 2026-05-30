import { format } from "date-fns";
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar, Users, AlertCircle } from 'lucide-react';
import { getProjectStatusColor } from '@/utils/partnerProjectHelpers';
import { useProjectModals } from '@/hooks/useModalNavigation';
import type { Project } from '@/types/api/project.types';

interface PartnerProjectsListProps {
  projects: Project[];
  isLoading: boolean;
}

export const PartnerProjectsList = ({ projects, isLoading }: PartnerProjectsListProps) => {
  const { openPartnerProjectDetails } = useProjectModals();

  const handleProjectClick = (project: Project) => {
    openPartnerProjectDetails(project.id);
  };

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardContent className="p-4 space-y-3">
              <div className="h-5 bg-muted rounded" />
              <div className="h-4 bg-muted rounded w-3/4" />
              <div className="flex gap-2">
                <div className="h-6 bg-muted rounded w-16" />
                <div className="h-6 bg-muted rounded w-12" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (projects.length === 0) {
    return (
      <Card className="text-center py-12">
        <CardContent>
          <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="typography-headline-medium text-foreground mb-2">
            Chưa có dự án
          </h3>
          <p className="typography-body-medium text-muted-foreground">
            Hiện tại chưa có dự án nào được tạo
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {projects.map((project) => (
        <Card 
          key={project.id}
          className="cursor-pointer transition-shadow"
          onClick={() => handleProjectClick(project)}
        >
          <CardContent className="p-4 space-y-3">
            <div className="space-y-1">
              <h3 className="typography-headline-small text-foreground line-clamp-1">
                {project.name}
              </h3>
              <p className="typography-body-medium text-muted-foreground line-clamp-1">
                {project.client_name}
              </p>
            </div>
            
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <div className="flex items-center gap-1">
                <Calendar className="h-4 w-4" />
                <span>{format(new Date(project.start_date), 'dd/MM/yyyy')}</span>
              </div>
              <div className="flex items-center gap-1">
                <Users className="h-4 w-4" />
                <span>{project.employee_count}</span>
              </div>
            </div>
            
            <div className="flex items-center justify-between">
              <Badge 
                variant="secondary"
                className={getProjectStatusColor(project.status)}
              >
                {project.status}
              </Badge>
              <span className="typography-label-small text-muted-foreground">
                #{project.code}
              </span>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};