import { useSearchParams } from 'react-router-dom';
import { ResetPasswordModal } from './ResetPasswordModal';
import { useUser, useResetPassword } from '@/hooks/api/useUsers';
import { LoadingSpinner } from '@/components/ui/loading-spinner';

interface ResetPasswordModalContainerProps {
  isOpen: boolean;
  onClose: () => void;
  userId?: string;
}

export function ResetPasswordModalContainer({
  isOpen,
  onClose,
  userId
}: ResetPasswordModalContainerProps) {
  const [searchParams] = useSearchParams();
  
  // Get user ID from props or URL params
  const userIdParam = userId || searchParams.get('userId') || searchParams.get('id');
  
  // Fetch user data
  const { data: user, isLoading, error } = useUser(
    userIdParam ? parseInt(userIdParam, 10) : 0,
    !!userIdParam && isOpen
  );
  
  // Reset password mutation
  const resetPasswordMutation = useResetPassword();
  
  // Handle password reset
  const handleResetPassword = (userId: number, password: string) => {
    resetPasswordMutation.mutate({ 
      id: userId, 
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
            Đang tải thông tin người dùng...
          </p>
        </div>
      </div>
    );
  }
  
  // Show error state
  if (error || !user) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
        <div className="bg-background rounded-xl p-6 shadow-sm max-w-md">
          <h3 className="text-lg font-semibold text-destructive mb-2">
            Lỗi tải dữ liệu
          </h3>
          <p className="text-muted-foreground mb-4">
            {!userIdParam 
              ? 'Không tìm thấy ID người dùng trong URL'
              : 'Không thể tải thông tin người dùng'
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
    <ResetPasswordModal
      open={isOpen}
      onClose={onClose}
      onResetPassword={handleResetPassword}
      user={user}
      loading={resetPasswordMutation.isPending}
    />
  );
}

export default ResetPasswordModalContainer;