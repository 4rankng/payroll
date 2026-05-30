import { useState, useMemo } from 'react';
import { User } from '@/types/user';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

export const useUserFilters = (users: User[]) => {
  const [searchTerm, setSearchTerm] = useState("");

  const filteredUsers = useMemo(() => {
    if (!searchTerm.trim()) return users;

    return users.filter(user =>
      (user.username && vietnameseIncludes(user.username, searchTerm)) ||
      (user.email && vietnameseIncludes(user.email, searchTerm)) ||
      (user.fullname && vietnameseIncludes(user.fullname, searchTerm)) ||
      (user.role && vietnameseIncludes(user.role, searchTerm))
    );
  }, [users, searchTerm]);

  const userStats = useMemo(() => {
    const total = users.length;
    const admins = users.filter(u => u.role === "admin").length;
    const partners = users.filter(u => u.role === "partner").length;

    return {
      total,
      admins,
      partners
    };
  }, [users]);

  return {
    searchTerm,
    setSearchTerm,
    filteredUsers,
    userStats,
  };
};