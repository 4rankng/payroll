import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { useProjects } from '@/hooks/api/useProjects';
import { Briefcase, Loader2 } from 'lucide-react';

interface Project {
  id: number;
  name: string;
  code: string;
}

interface ProjectSelectorProps {
  selectedProject: Project | null;
  onProjectChange: (project: Project | null) => void;
}

export function ProjectSelector({
  selectedProject,
  onProjectChange
}: ProjectSelectorProps) {
  const { data: projectsData, isLoading } = useProjects();
  const projects = projectsData?.data || [];

  const handleValueChange = (value: string) => {
    if (value === 'none') {
      onProjectChange(null);
    } else {
      const project = projects.find(proj => proj.id.toString() === value);
      onProjectChange(project || null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
        <span className="ml-2 typography-body-medium text-muted-foreground">Đang tải dự án...</span>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <Select
        value={selectedProject?.id.toString() || 'none'}
        onValueChange={handleValueChange}
      >
        <SelectTrigger className="h-12 border-2 border-border/50 hover:border-border transition-colors">
          <SelectValue placeholder="Chọn dự án">
            {selectedProject && (
              <div className="flex items-center gap-3">
                <div className="h-8 w-8 rounded-xl bg-primary/10 flex items-center justify-center">
                  <Briefcase className="h-4 w-4 text-primary" />
                </div>
                <div className="flex flex-col items-start">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="typography-body-small px-2 py-0">
                      {selectedProject.code}
                    </Badge>
                    <span className="font-medium text-foreground">
                      {selectedProject.name}
                    </span>
                  </div>
                </div>
              </div>
            )}
          </SelectValue>
        </SelectTrigger>
        <SelectContent className="max-h-60">
          <SelectItem value="none">
            <div className="flex items-center gap-3">
              <Briefcase className="h-8 w-8 text-muted-foreground" />
              <span className="text-muted-foreground">Chọn dự án</span>
            </div>
          </SelectItem>
          {projects.map((project) => (
            <SelectItem key={project.id} value={project.id.toString()}>
              <div className="flex items-center gap-3 py-1">
                <div className="h-8 w-8 rounded-xl bg-primary/10 flex items-center justify-center">
                  <Briefcase className="h-4 w-4 text-primary" />
                </div>
                <div className="flex flex-col items-start">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="typography-body-small px-2 py-0">
                      {project.code}
                    </Badge>
                    <span className="font-medium">{project.name}</span>
                  </div>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      
      {projects.length === 0 && (
        <div className="text-center py-8 text-muted-foreground">
          <Briefcase className="h-12 w-12 mx-auto mb-2 opacity-50" />
          <p className="typography-body-medium">Chưa có dự án nào</p>
        </div>
      )}
    </div>
  );
}