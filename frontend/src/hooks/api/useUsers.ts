import { useMutation, useQuery, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { userService } from '@/services/api/user.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification, showErrorNotification } from '@/utils/error-handler';
import type {
  CreateUserData,
  UpdateUserData,
  ResetPasswordData,
  UserActivitySummary,
} from '@/types/user';
import type { UserFilters, UserListResponse } from '@/services/api/user.service';

// Get users summary
export const useUsersSummary = () => {
  return useQuery({
    queryKey: QueryKeys.users.summary(),
    queryFn: () => userService.getSummary(),
  });
};

// Get paginated users list
export const useUsers = (filters?: UserFilters, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: QueryKeys.users.list(filters),
    queryFn: () => userService.getUsers(filters),
    enabled: options?.enabled ?? true,
  });
};

// Get infinite paginated users list (for infinite scroll)
export const useUsersInfinite = (filters?: Omit<UserFilters, 'page'>) => {
  const pageSize = filters?.pageSize || 20;

  return useInfiniteQuery({
    queryKey: QueryKeys.users.infiniteList(filters),
    queryFn: ({ pageParam = 1 }) =>
      userService.getUsers({
        ...filters,
        page: pageParam,
        pageSize,
      }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      if (!lastPage?.pagination) return undefined;
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
  });
};

// Get single user
export const useUser = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.users.detail(id),
    queryFn: () => userService.getUserById(id),
    enabled,
  });
};

// Create user
export const useCreateUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserData) => userService.createUser(data),
    onSuccess: (response) => {
      const newUser = response.data!;

      // Update the users list cache immediately with backend response
      queryClient.setQueriesData(
        { queryKey: QueryKeys.users.lists() },
        (oldData: UserListResponse | undefined) => {
          if (oldData?.data) {
            return {
              ...oldData,
              data: [newUser, ...oldData.data], // Add new user at the beginning
              pagination: oldData.pagination ? {
                ...oldData.pagination,
                totalRecords: oldData.pagination.totalRecords + 1
              } : undefined
            };
          }
          return oldData;
        }
      );

      // Also invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.lists() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.summary() });

      if (response?.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Update user
export const useUpdateUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateUserData }) =>
      userService.updateUser(id, data),
    onSuccess: (response) => {
      const updatedUser = response.data!;

      // Update specific user cache
      queryClient.setQueryData(
        QueryKeys.users.detail(updatedUser.id),
        updatedUser
      );
      // Invalidate list queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.lists() });

      if (response?.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};


// Reset password
export const useResetPassword = () => {

  return useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) =>
      userService.resetPassword(id, { password }),
    onSuccess: (response) => {
      if (response?.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Delete user
// Note: DELETE endpoint returns HTTP 204 No Content (no message)
export const useDeleteUser = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => userService.deleteUser(id),
    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: QueryKeys.users.detail(deletedId) });

      // Invalidate list queries
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.lists() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.users.summary() });

      // No message from backend - HTTP 204 returns no content
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Get user activities
export const useUserActivities = (id: number, days?: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.users.activities(id, days),
    queryFn: () => userService.getUserActivities(id, days),
    enabled,
  });
};

// Load ALL users once and cache as a plain record — use this anywhere you need to
// resolve user details by ID. Plain object avoids Map serialization issues with React Query.
export const useAllUsers = () => {
  return useQuery({
    queryKey: [...QueryKeys.users.lists(), 'all-map'],
    queryFn: async () => {
      const response = await userService.getUsers({ pageSize: 100, page: 1 });
      const record: Record<number, UserListResponse['data'][0]> = {};
      response.data.forEach(u => { record[u.id] = u; });
      return record;
    },
  });
};

// Batch user lookup by IDs — sends only the ids param, no pagination.
// Backend bypasses pagination when ids are provided and returns all matches directly.
export const useUsersByIds = (ids: number[], options?: { enabled?: boolean }) => {
  const idsKey = [...ids].sort((a, b) => a - b).join(',');

  return useQuery({
    queryKey: [...QueryKeys.users.lists(), 'batch', idsKey],
    queryFn: async (): Promise<Record<number, UserListResponse['data'][0]>> => {
      // Pass ONLY ids — no page/pageSize, backend handles it without pagination
      const response = await userService.getUsersByIds(idsKey);
      const record: Record<number, UserListResponse['data'][0]> = {};
      response.data.forEach(u => { record[u.id] = u; });
      return record;
    },
    enabled: (options?.enabled !== false) && ids.length > 0,
  });
};