import { useState } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Badge } from '@/components/ui/badge';
import { User } from 'lucide-react';
import { EmptyState } from '@/components/shared/EmptyState';

interface Employee {
  id: number;
  fullname: string;
  employee_code: string;
  position: string;
  avatar?: string;
}

interface EmployeeSelectorProps {
  employees: Employee[];
  selectedEmployee: Employee | null;
  onEmployeeChange: (employee: Employee | null) => void;
}

export function EmployeeSelector({
  employees,
  selectedEmployee,
  onEmployeeChange
}: EmployeeSelectorProps) {
  const handleValueChange = (value: string) => {
    if (value === 'none') {
      onEmployeeChange(null);
    } else {
      const employee = employees.find(emp => emp.id.toString() === value);
      onEmployeeChange(employee || null);
    }
  };

  return (
    <div className="space-y-3">
      <Select
        value={selectedEmployee?.id.toString() || 'none'}
        onValueChange={handleValueChange}
      >
        <SelectTrigger className="h-12 border-2 border-border/50 hover:border-border transition-colors">
          <SelectValue placeholder="Chọn nhân viên">
            {selectedEmployee && (
              <div className="flex items-center gap-3">
                <Avatar className="h-8 w-8">
                  <AvatarImage src={selectedEmployee.avatar} />
                  <AvatarFallback className="bg-primary/10 text-primary">
                    {selectedEmployee.fullname.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col items-start">
                  <span className="typography-body-medium text-foreground">
                    {selectedEmployee.fullname}
                  </span>
                  <span className="typography-body-small text-muted-foreground">
                    {selectedEmployee.employee_code}
                  </span>
                </div>
              </div>
            )}
          </SelectValue>
        </SelectTrigger>
        <SelectContent className="max-h-60">
          <SelectItem value="none">
            <div className="flex items-center gap-3">
              <User className="h-8 w-8 text-muted-foreground" />
              <span className="text-muted-foreground">Chọn nhân viên</span>
            </div>
          </SelectItem>
          {employees.map((employee) => (
            <SelectItem key={employee.id} value={employee.id.toString()}>
              <div className="flex items-center gap-3 py-1">
                <Avatar className="h-8 w-8">
                  <AvatarImage src={employee.avatar} />
                  <AvatarFallback className="bg-primary/10 text-primary typography-body-medium">
                    {employee.fullname.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="flex flex-col items-start">
                  <span className="typography-body-medium">{employee.fullname}</span>
                  <div className="flex items-center gap-2">
                    <span className="typography-body-small text-muted-foreground">
                      {employee.employee_code}
                    </span>
                    <Badge variant="secondary" className="typography-body-small px-2 py-0">
                      {employee.position}
                    </Badge>
                  </div>
                </div>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      
      {employees.length === 0 && (
        <EmptyState title="Chưa có nhân viên trong dự án" size="sm" className="py-4" />
      )}
    </div>
  );
}
