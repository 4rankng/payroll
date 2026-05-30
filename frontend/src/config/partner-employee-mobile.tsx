import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Mail, Phone, MapPin, Calendar, Building2 } from 'lucide-react';
import { formatDate } from '@/utils/formatters';
import { formatCurrency } from '@/utils/employeeHelpers';
import type { MobileField } from '@/components/ui/mobile-table';
import type { Employee } from '@/types/api/employee.types';

interface CreatePartnerEmployeeMobileConfigProps {
  onRowClick?: (employee: Employee) => void;
}

export function createPartnerEmployeeMobileConfig({
  onRowClick,
}: CreatePartnerEmployeeMobileConfigProps = {}) {
  const mobileFields: MobileField<Employee>[] = [
    {
      key: "name",
      label: "Tên nhân viên",
      priority: 1,
      render: (employee) => (
        <div className="flex items-center gap-3">
          <UserAvatar name={employee.name} email={employee.email} src={employee.avatar} size="md" />
          <div className="min-w-0 flex-1">
            <div className="typography-body-medium font-medium text-foreground line-clamp-1">
              {employee.name}
            </div>
          </div>
        </div>
      ),
    },
    {
      key: "code",
      label: "Mã nhân viên",
      priority: 2,
      render: (employee) => (
        <div className="typography-body-small text-muted-foreground">
          #{employee.employee_code}
        </div>
      ),
    },
    {
      key: "position",
      label: "Vị trí",
      priority: 2,
      render: (employee) => (
        <div className="space-y-1">
          <div className="typography-body-small text-foreground">
            {employee.position || "Chưa xác định"}
          </div>
          {employee.department && (
            <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
              <Building2 className="h-3 w-3 flex-shrink-0" />
              <span className="line-clamp-1">{employee.department}</span>
            </div>
          )}
        </div>
      ),
    },
    {
      key: "email",
      label: "Email",
      priority: 3,
      render: (employee) => {
        if (!employee.email) return (
          <span className="typography-body-small text-muted-foreground">
            Chưa cập nhật
          </span>
        );
        
        return (
          <div className="flex items-center gap-2">
            <Mail className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="typography-body-small text-muted-foreground line-clamp-1">
              {employee.email}
            </span>
          </div>
        );
      },
    },
    {
      key: "phone",
      label: "Điện thoại",
      priority: 3,
      render: (employee) => {
        if (!employee.phone) return (
          <span className="typography-body-small text-muted-foreground">
            Chưa cập nhật
          </span>
        );
        
        return (
          <div className="flex items-center gap-2">
            <Phone className="h-3 w-3 text-muted-foreground flex-shrink-0" />
            <span className="typography-body-small text-muted-foreground">
              {employee.phone}
            </span>
          </div>
        );
      },
    },
    {
      key: "address",
      label: "Địa chỉ",
      priority: 4,
      render: (employee) => {
        if (!employee.address) return (
          <span className="typography-body-small text-muted-foreground">
            Chưa cập nhật
          </span>
        );
        
        return (
          <div className="flex items-start gap-2">
            <MapPin className="h-3 w-3 text-muted-foreground flex-shrink-0 mt-0.5" />
            <span className="typography-body-small text-muted-foreground line-clamp-2">
              {employee.address}
            </span>
          </div>
        );
      },
    },
    {
      key: "dates",
      label: "Ngày quan trọng",
      priority: 4,
      render: (employee) => (
        <div className="space-y-1">
          {employee.date_of_birth && (
            <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
              <Calendar className="h-3 w-3 flex-shrink-0" />
              <span>Sinh: {formatDate(employee.date_of_birth)}</span>
            </div>
          )}
          {employee.hired_date && (
            <div className="flex items-center gap-1 typography-body-small text-muted-foreground">
              <Calendar className="h-3 w-3 flex-shrink-0" />
              <span>Vào: {formatDate(employee.hired_date)}</span>
            </div>
          )}
        </div>
      ),
    },
    {
      key: "salary",
      label: "Lương cơ bản",
      priority: 3,
      render: (employee) => {
        if (!employee.base_salary) return (
          <span className="typography-body-small text-muted-foreground">
            Chưa thiết lập
          </span>
        );
        
        return (
          <div className="typography-data-small tabular-nums font-medium text-foreground">
            {formatCurrency(employee.base_salary)}
          </div>
        );
      },
    },
    {
      key: "status",
      label: "Trạng thái",
      priority: 1,
      render: (employee) => {
        const getStatusVariant = (status: string) => {
          switch (status?.toLowerCase()) {
            case 'active':
            case 'hoạt động':
              return 'default';
            case 'inactive':
            case 'ngừng hoạt động':
              return 'secondary';
            case 'pending':
            case 'chờ xử lý':
              return 'outline';
            case 'terminated':
            case 'đã nghỉ việc':
              return 'destructive';
            default:
              return 'secondary';
          }
        };

        const getStatusText = (status: string) => {
          switch (status?.toLowerCase()) {
            case 'active':
              return 'Đang dùng';
            case 'inactive':
              return 'Ngừng hoạt động';
            case 'pending':
              return 'Chờ xử lý';
            case 'terminated':
              return 'Đã nghỉ việc';
            default:
              return status || 'Không xác định';
          }
        };

        return (
          <Badge 
            variant={getStatusVariant(employee.status || 'active')}
            className="typography-label-small whitespace-nowrap"
          >
            {getStatusText(employee.status || 'active')}
          </Badge>
        );
      },
    },
  ];

  const rowTitle = (employee: Employee) => (
    <div className="flex items-center gap-3">
      <UserAvatar name={employee.name} email={employee.email} src={employee.avatar} size="md" />
      <div className="typography-body-medium font-medium text-foreground line-clamp-1">
        {employee.name}
      </div>
    </div>
  );

  const rowSubtitle = (employee: Employee) => (
    <div className="flex items-center gap-2 typography-body-small text-muted-foreground">
      <span>#{employee.employee_code}</span>
      <span>•</span>
      <span>{employee.position || "Chưa xác định vị trí"}</span>
      {employee.department && (
        <>
          <span>•</span>
          <span>{employee.department}</span>
        </>
      )}
    </div>
  );

  return {
    mobileFields,
    rowTitle,
    rowSubtitle,
  };
}
