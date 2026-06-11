import { format } from "date-fns";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { DollarSign, Calendar, TrendingUp, CreditCard } from 'lucide-react';
import { useEmployeeSummary } from '@/hooks/api/useEmployees';
import { formatCurrency } from '@/utils/formatters';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeSummaryCardProps {
  employee: Employee;
  className?: string;
}

export function EmployeeSummaryCard({ employee, className }: EmployeeSummaryCardProps) {
  const { data: summary, isLoading, error } = useEmployeeSummary(employee.id);

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <DollarSign className="h-5 w-5" />
            Thống kê lương
          </CardTitle>
          <CardDescription>
            Tổng quan về thanh toán lương cho nhân viên
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-6 w-3/4" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-6 w-3/4" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-6 w-3/4" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-6 w-3/4" />
            </div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error || !summary) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <DollarSign className="h-5 w-5" />
            Thống kê lương
          </CardTitle>
          <CardDescription>
            Tổng quan về thanh toán lương cho nhân viên
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <p className="text-muted-foreground">
              {error ? 'Không thể tải dữ liệu thống kê' : 'Chưa có dữ liệu thống kê'}
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <DollarSign className="h-5 w-5" />
          Thống kê lương
        </CardTitle>
        <CardDescription>
          Tổng quan về thanh toán lương cho {employee.fullname}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Total Payments */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <CreditCard className="h-4 w-4 text-blue-600" />
              <span className="typography-body-medium typography-label-medium text-muted-foreground">
                Số lần thanh toán
              </span>
            </div>
            <div className="typography-headline-large">
              {summary.total_payroll_payments}
            </div>
          </div>

          {/* Total Earnings */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <DollarSign className="h-4 w-4 text-green-600" />
              <span className="typography-body-medium typography-label-medium text-muted-foreground">
                Tổng thu nhập
              </span>
            </div>
            <div className="typography-headline-large text-green-600">
              {formatCurrency(summary.total_earnings_vnd)}
            </div>
          </div>

          {/* Last Payment Date */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Calendar className="h-4 w-4 text-orange-600" />
              <span className="typography-body-medium typography-label-medium text-muted-foreground">
                Thanh toán cuối
              </span>
            </div>
            <div className="typography-title-large">
              {summary.last_payment_date ? (
                format(new Date(summary.last_payment_date), 'dd/MM/yyyy')
              ) : (
                <span className="text-muted-foreground">Chưa có</span>
              )}
            </div>
          </div>

          {/* Average Weekly Earnings */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-purple-600" />
              <span className="typography-body-medium typography-label-medium text-muted-foreground">
                Thu nhập TB/tuần
              </span>
            </div>
            <div className="typography-title-large text-purple-600">
              {formatCurrency(summary.avg_weekly_earnings_vnd)}
            </div>
          </div>
        </div>


      </CardContent>
    </Card>
  );
}
