import { Card, CardContent } from '@/components/ui/card';
import { User } from 'lucide-react';
import { EmployeeSelector } from '../EmployeeSelector';

interface Employee {
  id: number;
  fullname: string;
  employee_code: string;
  position: string;
  avatar?: string;
}

interface MobileEmployeeSelectorProps {
  employees: Employee[];
  selectedEmployee: Employee | null;
  onEmployeeChange: (employee: Employee | null) => void;
  onPositionChange: (position: string) => void;
}

export function MobileEmployeeSelector({
  employees,
  selectedEmployee,
  onEmployeeChange,
  onPositionChange
}: MobileEmployeeSelectorProps) {
  return (
    <Card className="border-none shadow-sm">
      <CardContent className="pt-4 sm:pt-6 pb-4 sm:pb-6">
        <div className="space-y-3">
          <div className="flex items-center gap-2 mb-2 sm:mb-3">
            <User className="h-4 w-4 text-muted-foreground" />
            <label className="typography-body-medium font-medium">Nhân viên</label>
          </div>
          <EmployeeSelector
            employees={employees}
            selectedEmployee={selectedEmployee}
            onEmployeeChange={(employee) => {
              onEmployeeChange(employee);
              if (employee) {
                onPositionChange(employee.position || '');
              }
            }}
          />
        </div>
      </CardContent>
    </Card>
  );
}
