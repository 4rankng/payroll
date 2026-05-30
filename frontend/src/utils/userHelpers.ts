export const getInitials = (username: string) => {
  return username
    .split(' ')
    .slice(0, 2)
    .map(n => n[0])
    .join('')
    .toUpperCase();
};

export const getRoleColor = (role: string): string => {
  switch (role) {
    case 'admin':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
    case 'partner':
      return 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300';
    case 'employee':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
  }
};

export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'active':
      return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
    case 'inactive':
      return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
  }
};

type UserLookup = { id: number; fullname: string; username: string };

export const getUserFullName = (
  userId: number | undefined | null,
  userMap: Map<number, UserLookup> | Record<number, UserLookup> | undefined
): string => {
  if (!userId) return '-';
  if (!userMap) return `ID #${userId}`;

  const user = userMap instanceof Map ? userMap.get(userId) : (userMap as Record<number, UserLookup>)[userId];
  return user?.fullname || user?.username || `ID #${userId}`;
};