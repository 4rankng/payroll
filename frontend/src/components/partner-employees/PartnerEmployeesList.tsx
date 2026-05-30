import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Mail, Phone, MapPin, AlertCircle } from 'lucide-react';
import { getEmployeeStatusColor } from '@/utils/partnerProjectHelpers';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import type { Employee } from '@/types/api/employee.types';

interface PartnerEmployeesListProps {
  employees: Employee[];
  isLoading: boolean;
}

export const PartnerEmployeesList = ({ employees, isLoading }: PartnerEmployeesListProps) => {
  const { openEmployeeDetails } = useEmployeeModals();

  const handleEmployeeClick = (employee: Employee) => {
    openEmployeeDetails(employee.id.toString());
  };

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} className="animate-pulse">
            <CardContent className="p-4 space-y-3">
              <div className="h-5 bg-muted rounded" />
              <div className="h-4 bg-muted rounded w-3/4" />
              <div className="space-y-2">
                <div className="h-3 bg-muted rounded" />
                <div className="h-3 bg-muted rounded w-2/3" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (employees.length === 0) {
    return (
      <Card className="text-center py-12">
        <CardContent>
          <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
          <h3 className="typography-headline-medium text-foreground mb-2">
            Chưa có nhân viên
          </h3>
          <p className="typography-body-medium text-muted-foreground">
            Hiện tại chưa có nhân viên nào
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {employees.map((employee) => (
        <Card 
          key={employee.id}
          className="cursor-pointer transition-shadow"
          onClick={() => handleEmployeeClick(employee)}
        >
          <CardContent className="p-4 space-y-3">
            <div className="space-y-1">
              <h3 className="typography-headline-small text-foreground">
                {employee.fullname || 'Không có tên'}
              </h3>
              <p className="typography-body-medium text-muted-foreground">
                {employee.position || 'Nhân viên'}
              </p>
            </div>
            
            <div className="space-y-2 typography-body-small text-muted-foreground">
              {employee.email && (
                <div className="flex items-center gap-2">
                  <Mail className="h-4 w-4" />
                  <span className="line-clamp-1">{employee.email}</span>
                </div>
              )}
              {employee.mobile && (
                <div className="flex items-center gap-2">
                  <Phone className="h-4 w-4" />
                  <span>{employee.mobile}</span>
                </div>
              )}
              {employee.address && (
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4" />
                  <span className="line-clamp-1">{employee.address}</span>
                </div>
              )}
            </div>
            
            <div className="flex items-center justify-between">
              <Badge 
                variant="secondary"
                className={getEmployeeStatusColor(employee.status || 'active')}
              >
                {employee.status === 'active' ? 'Đang dùng' : 'Không hoạt động'}
              </Badge>
              <span className="typography-label-small text-muted-foreground">
                #{employee.cccd || employee.id}
              </span>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};