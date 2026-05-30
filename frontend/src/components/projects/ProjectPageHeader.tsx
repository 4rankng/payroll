import { PageHeader } from '@/components/shared/PageHeader';
import { Plus, Briefcase } from 'lucide-react';

interface ProjectPageHeaderProps {
  onCreateProject: () => void;
  onAssignEmployees?: () => void;
  onGenerateReport?: () => void;
}

export const ProjectPageHeader = ({ onCreateProject }: ProjectPageHeaderProps) => {
  return (
    <PageHeader
      title="Dự án"
      description="Theo dõi và quản lý tất cả dự án trong công ty"
      icon={Briefcase}
      actions={[
        {
          label: 'Tạo dự án',
          onClick: onCreateProject,
          icon: Plus,
          variant: 'default' as const,
        },
      ]}
    />
  );
};
