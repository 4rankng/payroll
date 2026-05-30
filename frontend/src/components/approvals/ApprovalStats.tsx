import { StatsCards } from '@/components/shared/StatsCards';
import { Clock, CheckCircle, XCircle, AlertTriangle, FileText, RotateCw } from 'lucide-react';

interface ApprovalStatsProps {
  stats: {
    total: number;
    pending: number;
    approved: number;
    rejected: number;
    reviewing: number;
    urgent: number;
  };
}

export const ApprovalStats = ({ stats }: ApprovalStatsProps) => {
  const statsData = [
    {
      title: 'Tổng yêu cầu',
      value: stats.total,
      icon: FileText,
      color: 'blue' as const
    },
    {
      title: 'Chờ phê duyệt',
      value: stats.pending,
      icon: Clock,
      color: 'yellow' as const
    },
    {
      title: 'Đã phê duyệt',
      value: stats.approved,
      icon: CheckCircle,
      color: 'green' as const
    },
    {
      title: 'Khẩn cấp',
      value: stats.urgent,
      icon: AlertTriangle,
      color: 'red' as const
    }
  ];

  return <StatsCards stats={statsData} columns={4} />;
};