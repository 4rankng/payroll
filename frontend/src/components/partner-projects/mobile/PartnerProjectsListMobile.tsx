import { format } from "date-fns";
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar, Users, AlertCircle, ChevronRight } from 'lucide-react';
import { getProjectStatusColor } from '@/utils/partnerProjectHelpers';
import { useProjectModals } from '@/hooks/useModalNavigation';
import type { Project } from '@/types/api/project.types';

interface PartnerProjectsListMobileProps {
  projects: Project[];
  isLoading: boolean;
}

export const PartnerProjectsListMobile = ({ projects, isLoading }: PartnerProjectsListMobileProps) => {
  const { openPartnerProjectDetails } = useProjectModals();

  const handleProjectClick = (project: Project) => {
    openPartnerProjectDetails(String(project.id));
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
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
          <AlertCircle className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
          <h3 className="text-sm font-semibold text-foreground mb-2">
            Chưa có dự án
          </h3>
          <p className="text-sm text-muted-foreground">
            Hiện tại chưa có dự án nào
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {projects.map((project) => (
        <Card
          key={project.id}
          className="cursor-pointer active:bg-muted/50 transition-colors overflow-hidden"
          onClick={() => handleProjectClick(project)}
        >
          <CardContent className="p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0 space-y-2">
                <div className="space-y-1">
                  <h3 className="text-sm font-semibold text-foreground line-clamp-1">
                    {project.name}
                  </h3>
                  <p className="text-sm text-muted-foreground line-clamp-1">
                    {project.client_name}
                  </p>
                </div>
                
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <div className="flex items-center gap-1">
                    <Calendar className="h-3.5 w-3.5" />
                    <span>{format(new Date(project.start_date), 'dd/MM/yyyy')}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <Users className="h-3.5 w-3.5" />
                    <span>{project.employee_count}</span>
                  </div>
                </div>
                
                <div className="flex items-center justify-between">
                  <Badge 
                    variant="secondary"
                    className={`${getProjectStatusColor(project.status)} text-xs font-medium text-muted-foreground`}
                  >
                    {project.status}
                  </Badge>
                  <span className="text-xs font-medium text-muted-foreground">
                    #{project.code}
                  </span>
                </div>
              </div>
              
              <ChevronRight className="h-5 w-5 text-muted-foreground shrink-0" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};