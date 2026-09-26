import { useSearchParams } from 'react-router-dom';
import UserDetailsSheet from './UserDetailsSheet';
import { useUser, useUpdateUser, useDeleteUser, useResetPassword } from '@/hooks/api/useUsers';
import { LoadingSpinner } from '@/components/ui/loading-spinner';
import type { User, UpdateUserData } from '@/types/user';

interface UserDetailsSheetContainerProps {
  isOpen: boolean;
  onClose: () => void;
  id?: string;
  tab?: string;
}

export function UserDetailsSheetContainer({
  isOpen,
  onClose,
  id,
  tab
}: UserDetailsSheetContainerProps) {
  const [searchParams] = useSearchParams();
  
  // Get user ID from props or URL params
  const userId = id || searchParams.get('id');
  const initialTab = tab || searchParams.get('tab') || 'details';
  
  // Fetch user data
  const { data: user, isLoading, error } = useUser(
    userId ? parseInt(userId, 10) : 0,
    !!userId && isOpen
  );
  
  // Mutations
  const updateUserMutation = useUpdateUser();
  const deleteUserMutation = useDeleteUser();
  const resetPasswordMutation = useResetPassword();
  
  // Handle user update
  const handleUpdate = (userId: number, userData: UpdateUserData) => {
    updateUserMutation.mutate({ id: userId, data: userData });
  };
  
  // Handle user deletion
  const handleDelete = (user: User) => {
    deleteUserMutation.mutate(user.id, {
      onSuccess: () => {
        onClose();
      }
    });
  };
  
  
  // Handle password reset
  const handleResetPassword = (userId: number, password: string) => {
    resetPasswordMutation.mutate({ 
      id: userId, 
      password 
    });
  };
  
  // Show loading state
  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div className="bg-card rounded-xl p-6 shadow-sm">
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
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div className="bg-card rounded-xl p-6 shadow-sm max-w-md">
          <h3 className="text-lg font-semibold text-destructive mb-2">
            Lỗi tải dữ liệu
          </h3>
          <p className="text-muted-foreground mb-4">
            {!userId 
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
    <UserDetailsSheet
      user={user}
      isOpen={isOpen}
      onClose={onClose}
      onUpdate={handleUpdate}
      onDelete={handleDelete}
      onResetPassword={handleResetPassword}
      loading={
        updateUserMutation.isPending || 
        deleteUserMutation.isPending || 
        resetPasswordMutation.isPending
      }
      initialTab={initialTab}
    />
  );
}

export default UserDetailsSheetContainer;
