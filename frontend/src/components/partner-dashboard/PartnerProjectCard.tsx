import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Calendar, Eye, Edit } from "lucide-react";
import { Project } from "@/config/partner-dashboard/partner-dashboard-types";
import { formatProjectForDisplay } from "@/config/partner-dashboard/partner-dashboard-utils";
import { canPartnerManageProject } from "@/lib/permissions";

interface PartnerProjectCardProps {
  project: Project;
  onView: (projectId: number) => void;
  onEdit?: (projectId: number) => void;
}

export const PartnerProjectCard = ({ project, onView, onEdit }: PartnerProjectCardProps) => {
  const displayProject = formatProjectForDisplay(project);
  const canEditProject = canPartnerManageProject();

  return (
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between p-4 bg-gradient-card rounded-xl border border-border hover:shadow-card transition-smooth touch-manipulation">
      <div className="flex items-center space-x-4 mb-3 sm:mb-0">
        <div className="w-12 h-12 bg-partner-accent/10 rounded-xl flex items-center justify-center">
          <Calendar className="w-6 h-6 text-partner-accent" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="typography-title-large text-foreground truncate-mobile">{project.name}</h3>
          <div className="flex flex-col sm:flex-row sm:items-center sm:space-x-4 mt-1 space-y-1 sm:space-y-0">
            <p className="typography-body-small sm:typography-body-medium text-muted-foreground">
              {displayProject.displayDate}
            </p>
            <p className="typography-body-small sm:typography-body-medium text-muted-foreground">
              {displayProject.displayEmployees}
            </p>
          </div>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 sm:gap-3">
        <Badge className={`${displayProject.statusColor} text-white typography-status justify-center sm:justify-start`}>
          {project.status}
        </Badge>
        <div className="flex items-center gap-2">
          <Button 
            size="sm" 
            variant="outline"
            onClick={() => onView(project.id)}
            className="flex-1 sm:flex-none min-h-[44px] sm:min-h-[36px] touch-manipulation"
          >
            <Eye className="w-4 h-4 mr-2" />
            <span className="typography-label-medium">Xem</span>
          </Button>
          {canEditProject && onEdit && (
            <Button 
              size="sm" 
              variant="outline"
              onClick={() => onEdit(project.id)}
              className="flex-1 sm:flex-none min-h-[44px] sm:min-h-[36px] touch-manipulation"
            >
              <Edit className="w-4 h-4 mr-2" />
              <span className="typography-label-medium">Sửa</span>
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};