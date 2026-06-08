import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Mail, Phone, AlertCircle, ChevronRight } from 'lucide-react';
import { getEmployeeStatusColor } from '@/utils/partnerProjectHelpers';
import { useEmployeeModals } from '@/hooks/useModalNavigation';
import type { Employee } from '@/types/api/employee.types';

interface PartnerEmployeesListMobileProps {
  employees: Employee[];
  isLoading: boolean;
}

export const PartnerEmployeesListMobile = ({ employees, isLoading }: PartnerEmployeesListMobileProps) => {
  const { openEmployeeDetails } = useEmployeeModals();

  const handleEmployeeClick = (employee: Employee) => {
    openEmployeeDetails(employee.id.toString());
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
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
          <AlertCircle className="w-10 h-10 text-muted-foreground mx-auto mb-3" />
          <h3 className="text-sm font-semibold text-foreground mb-2">
            Chưa có nhân viên
          </h3>
          <p className="text-sm text-muted-foreground">
            Hiện tại chưa có nhân viên nào
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {employees.map((employee) => (
        <Card
          key={employee.id}
          className="cursor-pointer active:bg-muted/50 transition-colors overflow-hidden"
          onClick={() => handleEmployeeClick(employee)}
        >
          <CardContent className="p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0 space-y-2">
                <div className="space-y-1">
                  <h3 className="text-sm font-semibold text-foreground">
                    {employee.fullname}
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    {employee.position || 'Nhân viên'}
                  </p>
                </div>
                
                <div className="space-y-1 text-sm text-muted-foreground">
                  {employee.email && (
                    <div className="flex items-center gap-2">
                      <Mail className="h-3.5 w-3.5" />
                      <span className="line-clamp-1">{employee.email}</span>
                    </div>
                  )}
                  {employee.mobile && (
                    <div className="flex items-center gap-2">
                      <Phone className="h-3.5 w-3.5" />
                      <span>{employee.mobile}</span>
                    </div>
                  )}
                </div>
                
                <div className="flex items-center justify-between">
                  <Badge 
                    variant="secondary"
                    className={`${getEmployeeStatusColor(employee.status || 'active')} text-xs font-medium text-muted-foreground`}
                  >
                    {employee.status || 'Đang dùng'}
                  </Badge>
                  <span className="text-xs font-medium text-muted-foreground">
                    #{employee.employee_code}
                  </span>
                </div>
              </div>
              
              <ChevronRight className="h-5 w-5 text-muted-foreground shrink-0" />
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};