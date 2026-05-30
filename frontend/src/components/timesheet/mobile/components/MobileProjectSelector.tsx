import { Card, CardContent } from '@/components/ui/card';
import { Briefcase } from 'lucide-react';
import { ProjectSelector } from '../ProjectSelector';

interface Project {
  id: number;
  name: string;
  code: string;
}

interface MobileProjectSelectorProps {
  selectedProject: Project | null;
  onProjectChange: (project: Project | null) => void;
}

export function MobileProjectSelector({
  selectedProject,
  onProjectChange
}: MobileProjectSelectorProps) {
  return (
    <Card className="border-none shadow-sm">
      <CardContent className="pt-4 sm:pt-6 pb-4 sm:pb-6">
        <div className="space-y-3">
          <div className="flex items-center gap-2 mb-2 sm:mb-3">
            <Briefcase className="h-4 w-4 text-muted-foreground" />
            <label className="typography-body-medium font-medium">Dự án</label>
          </div>
          <ProjectSelector
            selectedProject={selectedProject}
            onProjectChange={onProjectChange}
          />
        </div>
      </CardContent>
    </Card>
  );
}