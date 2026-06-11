import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Briefcase, Calendar, Clock, DollarSign, ChevronLeft, ChevronRight } from 'lucide-react';
import { useEmployeeProjects } from '@/hooks/api/useEmployees';
import { formatCurrency } from '@/utils/formatters';
import { formatDate } from '@/utils/formatters';
import type { Employee, EmployeeProjectAssignment } from '@/types/api/employee.types';

interface EmployeeProjectsListProps {
  employee: Employee;
  className?: string;
}

export function EmployeeProjectsList({ employee, className }: EmployeeProjectsListProps) {
  const [status, setStatus] = useState<'active' | 'ended' | 'inactive' | 'all'>('all');
  const [page, setPage] = useState(1);
  const pageSize = 10;

  const { data: projectsData, isLoading, error } = useEmployeeProjects(
    employee.id,
    {
      status: status === 'all' ? undefined : status,
      page,
      pageSize,
    }
  );

  const projects = projectsData?.data || [];
  const pagination = projectsData?.pagination || { page: 1, pageSize: 10, totalPages: 1, totalRecords: 0 };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge className="bg-green-100 text-green-800 border-green-300">Đang làm</Badge>;
      case 'ended':
        return <Badge variant="secondary">Kết thúc</Badge>;
      case 'inactive':
        return <Badge variant="outline">Không hoạt động</Badge>;
      default:
        return <Badge variant="secondary">{status}</Badge>;
    }
  };

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Briefcase className="h-5 w-5" />
            Dự án đã tham gia
          </CardTitle>
          <CardDescription>
            Danh sách các dự án mà nhân viên đã và đang tham gia
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-6 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
                <Skeleton className="h-4 w-1/4" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Briefcase className="h-5 w-5" />
            Dự án đã tham gia
          </CardTitle>
          <CardDescription>
            Danh sách các dự án mà nhân viên đã và đang tham gia
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <p className="text-muted-foreground">
              Không thể tải danh sách dự án
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
          <Briefcase className="h-5 w-5" />
          Dự án đã tham gia
          <Badge variant="outline" className="ml-auto">
            {pagination.totalRecords}
          </Badge>
        </CardTitle>
        <CardDescription>
          Danh sách các dự án mà {employee.fullname} đã và đang tham gia
        </CardDescription>

        {/* Filter */}
        <div className="flex items-center gap-2">
          <Select value={status} onValueChange={(value: 'active' | 'ended' | 'inactive' | 'all') => {
            setStatus(value);
            setPage(1); // Reset to first page when filter changes
          }}>
            <SelectTrigger className="w-48">
              <SelectValue placeholder="Lọc theo trạng thái" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tất cả trạng thái</SelectItem>
              <SelectItem value="active">Đang làm</SelectItem>
              <SelectItem value="ended">Kết thúc</SelectItem>
              <SelectItem value="inactive">Không hoạt động</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardHeader>

      <CardContent>
        {projects.length === 0 ? (
          <div className="text-center py-8">
            <Briefcase className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <p className="text-muted-foreground">
              {status ? 'Không tìm thấy dự án nào với trạng thái này' : 'Nhân viên chưa tham gia dự án nào'}
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {/* Table for larger screens */}
            <div className="hidden md:block">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Dự án</TableHead>
                    <TableHead>Mã nhân viên</TableHead>
                    <TableHead>Thời gian</TableHead>
                    <TableHead>Ca làm việc</TableHead>
                    <TableHead>Thu nhập</TableHead>
                    <TableHead>Trạng thái</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {projects.map((project) => (
                    <TableRow key={project.assignment_id}>
                      <TableCell>
                        <div>
                          <div className="font-medium">{project.project_name}</div>
                          <div className="typography-body-medium text-muted-foreground">{project.project_code}</div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="font-mono typography-body-medium">{project.factory_employee_code}</div>
                      </TableCell>
                      <TableCell>
                        <div className="typography-body-medium">
                          <div>{formatDate(project.start_date)}</div>
                          {project.end_date && (
                            <div className="text-muted-foreground">→ {formatDate(project.end_date)}</div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Clock className="h-4 w-4 text-muted-foreground" />
                          {project.total_hours_worked} giờ
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1 text-green-600">
                          <DollarSign className="h-4 w-4" />
                          {formatCurrency(project.total_earned_vnd)}
                        </div>
                      </TableCell>
                      <TableCell>{getStatusBadge(project.status)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {/* Cards for smaller screens */}
            <div className="md:hidden space-y-4">
              {projects.map((project) => (
                <Card key={project.assignment_id}>
                  <CardContent className="p-4">
                    <div className="flex justify-between items-start mb-2">
                      <div>
                        <h4 className="font-medium">{project.project_name}</h4>
                        <p className="typography-body-medium text-muted-foreground">{project.project_code}</p>
                      </div>
                      {getStatusBadge(project.status)}
                    </div>

                    <div className="grid grid-cols-2 gap-4 typography-body-medium">
                      <div>
                        <span className="text-muted-foreground">Mã NV:</span>
                        <div className="font-mono">{project.factory_employee_code}</div>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Ca làm việc:</span>
                        <div>{project.total_hours_worked} giờ</div>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Bắt đầu:</span>
                        <div>{formatDate(project.start_date)}</div>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Thu nhập:</span>
                        <div className="text-green-600">{formatCurrency(project.total_earned_vnd)}</div>
                      </div>
                    </div>

                    {project.end_date && (
                      <div className="mt-2 typography-body-medium">
                        <span className="text-muted-foreground">Kết thúc: </span>
                        {formatDate(project.end_date)}
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>

            {/* Pagination */}
            {pagination.totalPages > 1 && (
              <div className="flex items-center justify-between">
                <div className="typography-body-medium text-muted-foreground">
                  Trang {pagination.page} của {pagination.totalPages} ({pagination.totalRecords} kết quả)
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(page - 1)}
                    disabled={page <= 1}
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Trước
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage(page + 1)}
                    disabled={page >= pagination.totalPages}
                  >
                    Tiếp
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
