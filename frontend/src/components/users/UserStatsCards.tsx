import { UserStatsCard, StatItemProps } from '@/components/users/UserStatsCard';
import { Skeleton } from '@/components/ui/skeleton';
import { UserSummary } from '@/types/user';
import { useCallback, useMemo } from 'react';
import { Users, ShieldCheck, Briefcase, UserCheck } from 'lucide-react';

interface UserStatsCardsProps {
  stats?: UserSummary;
  isLoading?: boolean;
  error?: boolean;
  onRoleSelect?: (role?: 'admin' | 'partner' | 'employee' | 'accountant') => void;
  selectedRole?: 'admin' | 'partner' | 'employee' | 'accountant';
  lastLoginToday?: boolean;
}

export const UserStatsCards = ({
  stats,
  isLoading = false,
  error = false,
  onRoleSelect,
  selectedRole,
  lastLoginToday = false,
}: UserStatsCardsProps) => {
  const handleAll = useCallback(() => onRoleSelect?.(undefined), [onRoleSelect]);
  const handleAdmin = useCallback(() => onRoleSelect?.('admin'), [onRoleSelect]);
  const handlePartner = useCallback(() => onRoleSelect?.('partner'), [onRoleSelect]);
  const handleEmployee = useCallback(() => onRoleSelect?.('employee'), [onRoleSelect]);

  // Merged stats in 2x2 grid: [Tổng, QTV] [QL, NV]
  const mergedStats = useMemo<StatItemProps[]>(() => {
    if (!stats) return [];
    return [
      {
        label: 'Tổng',
        value: stats.total_users || 0,
        icon: Users,
        onClick: onRoleSelect ? handleAll : undefined,
        isActive: !selectedRole && !lastLoginToday,
      },
      {
        label: 'Quản trị viên',
        value: stats.total_admins || 0,
        icon: ShieldCheck,
        onClick: onRoleSelect ? handleAdmin : undefined,
        isActive: selectedRole === 'admin',
      },
      {
        label: 'Quản lý',
        value: stats.total_partners || 0,
        icon: Briefcase,
        onClick: onRoleSelect ? handlePartner : undefined,
        isActive: selectedRole === 'partner',
      },
      {
        label: 'Nhân viên',
        value: stats.total_employees || 0,
        icon: UserCheck,
        onClick: onRoleSelect ? handleEmployee : undefined,
        isActive: selectedRole === 'employee',
      },
    ];
  }, [stats, onRoleSelect, handleAll, handleAdmin, handlePartner, handleEmployee, selectedRole, lastLoginToday]);

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-[52px] rounded-xl border border-border/60 bg-card px-3 py-2.5 flex flex-col gap-1.5">
            <Skeleton className="h-2.5 w-16" />
            <Skeleton className="h-4 w-10" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 border border-border rounded-xl bg-card text-center">
        <p className="typography-body-medium text-muted-foreground">
          Không thể tải thống kê người dùng
        </p>
      </div>
    );
  }

  return mergedStats.length > 0 ? <UserStatsCard stats={mergedStats} /> : null;
};
