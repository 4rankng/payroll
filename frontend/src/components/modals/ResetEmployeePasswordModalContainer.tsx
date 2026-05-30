import { useSearchParams } from 'react-router-dom';
import { ResetEmployeePasswordModal } from './ResetEmployeePasswordModal';
import { useEmployee, useChangeEmployeePassword } from '@/hooks/api/useEmployees';
import { LoadingSpinner } from '@/components/ui/loading-spinner';

interface ResetEmployeePasswordModalContainerProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId?: string;
}

export function ResetEmployeePasswordModalContainer({
  isOpen,
  onClose,
  employeeId
}: ResetEmployeePasswordModalContainerProps) {
  const [searchParams] = useSearchParams();

  // Get employee ID from props or URL params
  const employeeIdParam = employeeId || searchParams.get('employeeId') || searchParams.get('id');

  // Fetch employee data
  const { data: employee, isLoading, error } = useEmployee(
    employeeIdParam ? parseInt(employeeIdParam, 10) : 0,
    !!employeeIdParam && isOpen
  );

  // Change password mutation
  const changePasswordMutation = useChangeEmployeePassword();

  // Handle password reset
  const handleResetPassword = (employeeId: number, password: string) => {
    changePasswordMutation.mutate({
      id: employeeId,
      password
    }, {
      onSuccess: () => {
        onClose();
      }
    });
  };

  // Show loading state
  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
        <div className="bg-background rounded-xl p-6 shadow-sm">
          <LoadingSpinner className="mx-auto" />
          <p className="mt-4 text-center text-muted-foreground">
            Đang tải thông tin nhân viên...
          </p>
        </div>
      </div>
    );
  }

  // Show error state
  if (error || !employee) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
        <div className="bg-background rounded-xl p-6 shadow-sm max-w-md">
          <h3 className="text-lg font-semibold text-destructive mb-2">
            Lỗi tải dữ liệu
          </h3>
          <p className="text-muted-foreground mb-4">
            {!employeeIdParam
              ? 'Không tìm thấy ID nhân viên trong URL'
              : 'Không thể tải thông tin nhân viên'
            }
          </p>
          <button
            className="w-full px-4 py-2 bg-primary text-primary-foreground rounded-xl"
            onClick={onClose}
          >
            Đóng
          </button>
        </div>
      </div>
    );
  }

  return (
    <ResetEmployeePasswordModal
      open={isOpen}
      onClose={onClose}
      onResetPassword={handleResetPassword}
      employee={employee}
      loading={changePasswordMutation.isPending}
    />
  );
}

export default ResetEmployeePasswordModalContainer;
