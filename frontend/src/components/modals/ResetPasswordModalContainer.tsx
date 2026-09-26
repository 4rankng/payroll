import { useSearchParams } from 'react-router-dom';
import { ResetPasswordModal } from './ResetPasswordModal';
import { useUser, useResetPassword } from '@/hooks/api/useUsers';
import { useEmployee, useChangeEmployeePassword } from '@/hooks/api/useEmployees';
import { LoadingSpinner } from '@/components/ui/loading-spinner';

type TargetType = 'user' | 'employee';

interface ResetPasswordModalContainerProps {
  isOpen: boolean;
  onClose: () => void;
  targetType?: TargetType;
  userId?: string;
  employeeId?: string;
}

const TARGET_CONFIG = {
  user: {
    loadingMessage: 'Đang tải thông tin người dùng...',
    missingIdMessage: 'Không tìm thấy ID người dùng trong URL',
    errorMessage: 'Không thể tải thông tin người dùng',
  },
  employee: {
    loadingMessage: 'Đang tải thông tin nhân viên...',
    missingIdMessage: 'Không tìm thấy ID nhân viên trong URL',
    errorMessage: 'Không thể tải thông tin nhân viên',
  },
} as const;

export function ResetPasswordModalContainer({
  isOpen,
  onClose,
  targetType = 'user',
  userId,
  employeeId
}: ResetPasswordModalContainerProps) {
  const [searchParams] = useSearchParams();

  const config = TARGET_CONFIG[targetType];

  // Resolve ID from props or URL params
  const idParam = targetType === 'employee'
    ? employeeId || searchParams.get('employeeId') || searchParams.get('id')
    : userId || searchParams.get('userId') || searchParams.get('id');

  // All hooks must be called unconditionally (rules-of-hooks).
  // The `enabled` flag prevents network requests when modal is closed or
  // the wrong target type is active.
  const userQuery = useUser(
    idParam && targetType === 'user' ? parseInt(idParam, 10) : 0,
    !!idParam && isOpen && targetType === 'user'
  );
  const resetPasswordMutation = useResetPassword();

  const employeeQuery = useEmployee(
    idParam && targetType === 'employee' ? parseInt(idParam, 10) : 0,
    !!idParam && isOpen && targetType === 'employee'
  );
  const changeEmployeePasswordMutation = useChangeEmployeePassword();

  // Don't render anything when the modal is closed.
  // Without this guard, the disabled queries return no data and the
  // error overlay (fixed inset-0 z-50) would cover the entire page.
  if (!isOpen) return null;

  // Select the right data/mutation based on target type
  const isLoading = targetType === 'user' ? userQuery.isLoading : employeeQuery.isLoading;
  const error = targetType === 'user' ? userQuery.error : employeeQuery.error;
  const target = targetType === 'user'
    ? userQuery.data ?? null
    : employeeQuery.data ?? null;

  const isPending = targetType === 'user'
    ? resetPasswordMutation.isPending
    : changeEmployeePasswordMutation.isPending;

  // Handle password reset
  const handleResetPassword = (id: number, password: string) => {
    if (targetType === 'user') {
      resetPasswordMutation.mutate({ id, password }, { onSuccess: () => onClose() });
    } else {
      changeEmployeePasswordMutation.mutate({ id, password }, { onSuccess: () => onClose() });
    }
  };

  // Show loading state
  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-card/80 backdrop-blur-sm">
        <div className="bg-card rounded-xl p-6 shadow-sm">
          <LoadingSpinner className="mx-auto" />
          <p className="mt-4 text-center text-muted-foreground">
            {config.loadingMessage}
          </p>
        </div>
      </div>
    );
  }

  // Show error state
  if (error || !target) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-card/80 backdrop-blur-sm">
        <div className="bg-card rounded-xl p-6 shadow-sm max-w-md">
          <h3 className="text-lg font-semibold text-destructive mb-2">
            Lỗi tải dữ liệu
          </h3>
          <p className="text-muted-foreground mb-4">
            {!idParam ? config.missingIdMessage : config.errorMessage}
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
    <ResetPasswordModal
      open={isOpen}
      onClose={onClose}
      onResetPassword={handleResetPassword}
      target={target}
      targetType={targetType}
      loading={isPending}
    />
  );
}

export default ResetPasswordModalContainer;
