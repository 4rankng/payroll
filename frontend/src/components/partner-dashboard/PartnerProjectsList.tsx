import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { PartnerProjectCard } from "./PartnerProjectCard";
import { Project } from "@/config/partner-dashboard/partner-dashboard-types";


interface PartnerProjectsListProps {
  projects: Project[];
  onViewProject: (projectId: number) => void;
  onEditProject?: (projectId: number) => void;
  loading?: boolean;
}

export const PartnerProjectsList = ({
  projects,
  onViewProject,
  onEditProject,
  loading = false
}: PartnerProjectsListProps) => {
  if (loading) {
    return (
      <Card className="shadow-card">
        <CardContent className="p-6">
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse">
                <div className="flex items-center justify-between p-4 bg-muted/50 rounded-xl">
                  <div className="flex items-center space-x-4">
                    <div className="w-12 h-12 bg-muted rounded-xl" />
                    <div className="space-y-2">
                      <div className="h-4 bg-muted rounded w-32" />
                      <div className="h-3 bg-muted rounded w-48" />
                    </div>
                  </div>
                  <div className="flex items-center space-x-3">
                    <div className="h-6 bg-muted rounded w-16" />
                    <div className="h-8 bg-muted rounded w-16" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="shadow-card">
      <CardHeader />
      <CardContent>
        <div className="space-y-4">
          {projects.map((project) => (
            <PartnerProjectCard
              key={project.id}
              project={project}
              onView={onViewProject}
              onEdit={onEditProject}
            />
          ))}
        </div>


      </CardContent>
    </Card>
  );
};
