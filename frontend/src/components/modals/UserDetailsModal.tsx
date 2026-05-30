import { useSearchParams } from "react-router-dom";
import UserDetailsSheet from "@/components/sheets/UserDetailsSheet";
import { useUser, useUpdateUser, useDeleteUser, useResetPassword } from "@/hooks/api/useUsers";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import type { User, UpdateUserData } from "@/types/user";

export const modalConfig = {
  id: 'user-details',
};

/**
 * Route-based User Details Modal
 * Fetches the specific user by ID — no need to load the full list.
 */
export function UserDetailsModal() {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const updateUserMutation = useUpdateUser();
  const deleteUserMutation = useDeleteUser();
  const resetPasswordMutation = useResetPassword();

  const modalId = searchParams.get('modal');
  const userId = searchParams.get('id');
  const isOpen = modalId === 'user_details_sheet' && Boolean(userId);
  const numericId = userId ? parseInt(userId, 10) : 0;

  // Fetch only the specific user — no full-list fetch
  const { data: fetchedUser } = useUser(numericId, isOpen && !!numericId);
  const selectedUser = fetchedUser ?? null;

  const handleClose = () => {
    closeModal();
  };

  const handleUpdate = (userId: number, userData: UpdateUserData) => {
    updateUserMutation.mutate({ id: userId, data: userData });
  };

  const handleDelete = (user: User) => {
    deleteUserMutation.mutate(user.id, {
      onSuccess: () => {
        handleClose();
      }
    });
  };

  const handleResetPassword = (userId: number, password: string) => {
    resetPasswordMutation.mutate({ id: userId, password });
  };

  if (!isOpen) return null;

  return (
    <UserDetailsSheet
      user={selectedUser}
      isOpen={isOpen}
      onClose={handleClose}
      onUpdate={handleUpdate}
      onDelete={handleDelete}
      onResetPassword={handleResetPassword}
    />
  );
}
