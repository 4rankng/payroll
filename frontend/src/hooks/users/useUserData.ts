import { useState } from 'react';
import { toast } from '@/components/ui/sonner';
import { User, CreateUserData, UpdateUserData } from '@/types/user';
import {
  useUsers,
  useCreateUser,
  useUpdateUser,
  useDeleteUser
} from '@/hooks/api/useUsers';
import type { UserFilters } from '@/services/api/user.service';

export const useUserData = (filters?: UserFilters & {
  page?: number;
  pageSize?: number;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
}) => {

  // API hooks
  const { data: usersResponse, isLoading, error } = useUsers(filters);
  const createUserMutation = useCreateUser();
  const updateUserMutation = useUpdateUser();
  const deleteUserMutation = useDeleteUser();

  const users = usersResponse?.data || [];
  const pagination = usersResponse?.pagination;

  const handleAddUser = (userData: CreateUserData) => {
    createUserMutation.mutate(userData);
  };

  const handleEditUser = (userData: UpdateUserData & { id: number }) => {
    const { id, ...data } = userData;
    updateUserMutation.mutate({ id, data });
  };

  const handleDeleteUser = (userId: number) => {
    deleteUserMutation.mutate(userId);
  };

  return {
    users,
    pagination,
    onPageChange: filters?.onPageChange,
    onPageSizeChange: filters?.onPageSizeChange,
    isLoading: isLoading || createUserMutation.isPending || updateUserMutation.isPending || deleteUserMutation.isPending,
    error,
    handleAddUser,
    handleEditUser,
    handleDeleteUser,
  };
};