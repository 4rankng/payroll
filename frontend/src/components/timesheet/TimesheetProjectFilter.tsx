import { SearchableSelect } from '@/components/ui/searchable-select';
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

  // The 'none' entry doubles as the placeholder value; when no project is
  // active it becomes the disabled notice row, matching the old Select.
  const options = activeProjects.length === 0
    ? [{ value: 'none', label: 'Không có dự án đang hoạt động', disabled: true }]
    : [
        { value: 'none', label: 'Chọn dự án' },
        ...activeProjects.map((project) => ({
          value: project.id.toString(),
          label: `${project.code} - ${project.name}`,
        })),
      ];

  return (
    <div className="flex items-center gap-1.5">
      <span className="text-xs text-muted-foreground whitespace-nowrap">Dự án:</span>
      <SearchableSelect
        value={selectedProject?.id.toString() || 'none'}
        onChange={handleProjectChange}
        placeholder="Chọn dự án"
        options={options}
        triggerClassName="w-[180px]"
      />
    </div>
  );
}