import { useProject } from '@/hooks/api/useProjects';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { ErrorState } from '@/components/ui/error-state';
import { AddEmployeesToProject } from './AddEmployeesToProject';

interface AddEmployeeToProjectSheetContainerProps {
  isOpen: boolean;
  onClose: () => void;
  projectId?: string | null;
}

export function AddEmployeeToProjectSheetContainer({ isOpen, onClose, projectId }: AddEmployeeToProjectSheetContainerProps) {
  const parsedId = Number(projectId);
  const validId = Number.isSafeInteger(parsedId) && parsedId > 0;
  const { data: project, isLoading, isError, refetch } = useProject(validId ? parsedId : 0, isOpen && validId);

  if (!isOpen) return null;

  if (!project || isError) {
    return (
      <SlideSheetTemplate isOpen={isOpen} onClose={onClose} title="Thêm nhân viên vào dự án">
        {validId && isLoading ? (
          <p role="status" className="py-8 text-center text-sm text-muted-foreground">Đang tải thông tin dự án...</p>
        ) : (
          <div role="alert">
            <ErrorState
              message={validId ? 'Không thể tải thông tin dự án. Vui lòng thử lại.' : 'Liên kết thiếu mã dự án hợp lệ.'}
              onRetry={validId ? () => void refetch() : undefined}
            />
          </div>
        )}
      </SlideSheetTemplate>
    );
  }

  return <AddEmployeesToProject project={project} isOpen={isOpen} onClose={onClose} />;
}
