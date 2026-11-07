import { PageHeader } from '@/components/shared/PageHeader';
import { useLocation } from 'react-router-dom';
import { getConsoleKicker } from '@/lib/console';
import { Plus, Briefcase } from 'lucide-react';

interface ProjectPageHeaderProps {
  onCreateProject: () => void;
  /** Record count already shown by the page's pagination footer. */
  meta?: string;
}

export const ProjectPageHeader = ({ onCreateProject, meta }: ProjectPageHeaderProps) => {
  const { pathname } = useLocation();

  return (
    <PageHeader
      kicker={getConsoleKicker(pathname)}
      title="Dự án"
      meta={meta}
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
