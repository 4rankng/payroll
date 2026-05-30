import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Project } from '@/types/api/project.types';

interface TimesheetProjectFilterProps {
  projects: Project[];
  selectedProject: Project | null;
  onProjectChange: (project: Project | null) => void;
}

export function TimesheetProjectFilter({
  projects,
  selectedProject,
  onProjectChange
}: TimesheetProjectFilterProps) {
  // Filter to only show active projects
  const activeProjects = projects.filter(project => project.status === 'active');

  const handleProjectChange = (value: string) => {
    if (value === 'none') {
      onProjectChange(null);
      return;
    }
    
    const project = activeProjects.find(p => p.id.toString() === value);
    onProjectChange(project || null);
  };

  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs text-muted-foreground whitespace-nowrap">Dự án:</span>
      <Select
        value={selectedProject?.id.toString() || 'none'}
        onValueChange={handleProjectChange}
      >
        <SelectTrigger className="h-8 w-[180px] text-xs">
          <SelectValue placeholder="Chọn dự án" />
        </SelectTrigger>
        <SelectContent>
          {activeProjects.length === 0 ? (
            <SelectItem value="none" disabled>
              Không có dự án đang hoạt động
            </SelectItem>
          ) : (
            <>
              <SelectItem value="none">Chọn dự án</SelectItem>
              {activeProjects.map((project) => (
                <SelectItem key={project.id} value={project.id.toString()}>
                  {project.code} - {project.name}
                </SelectItem>
              ))}
            </>
          )}
        </SelectContent>
      </Select>
    </div>
  );
}