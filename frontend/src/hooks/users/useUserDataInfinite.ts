import { useMemo } from "react";
import { CreateUserData, UpdateUserData } from "@/types/user";
import {
  useUsersInfinite,
  useCreateUser,
  useUpdateUser,
  useDeleteUser,
} from "@/hooks/api/useUsers";
import type { UserFilters } from "@/services/api/user.service";

export const useUserDataInfinite = (filters?: Omit<UserFilters, "page">) => {
  const {
    data: usersResponse,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useUsersInfinite(filters);

  const createUserMutation = useCreateUser();
  const updateUserMutation = useUpdateUser();
  const deleteUserMutation = useDeleteUser();

  const users = useMemo(() => {
    if (!usersResponse?.pages) return [];
    return usersResponse.pages.flatMap((page) => page.data);
  }, [usersResponse?.pages]);

  const pagination = useMemo(() => {
    const lastPage = usersResponse?.pages?.[usersResponse.pages.length - 1];
    return lastPage?.pagination;
  }, [usersResponse?.pages]);

  return {
    users,
    pagination,
    isLoading,
    error,
    isFetchingNextPage,
    hasMore: hasNextPage ?? false,
    fetchNextPage,
    handleAddUser: (userData: CreateUserData) =>
      createUserMutation.mutate(userData),
    handleEditUser: (userData: UpdateUserData & { id: number }) => {
      const { id, ...data } = userData;
      updateUserMutation.mutate({ id, data });
    },
    handleDeleteUser: (userId: number) => deleteUserMutation.mutate(userId),
    isCreating: createUserMutation.isPending,
    isUpdating: updateUserMutation.isPending,
    isDeleting: deleteUserMutation.isPending,
  };
};
