import { Users } from "lucide-react";

interface PartnerEmployeesHeaderProps {
  totalEmployees: number;
}

export const PartnerEmployeesHeader = ({ totalEmployees }: PartnerEmployeesHeaderProps) => {
  return (
    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
      <div className="space-y-1">
        <h1 className="typography-headline-medium sm:typography-headline-large lg:typography-display-small text-foreground">
          Danh sách Nhân viên
        </h1>
        <p className="typography-body-small sm:typography-body-medium lg:typography-body-large text-muted-foreground">
          Quản lý thông tin {totalEmployees} nhân viên trong các dự án
        </p>
      </div>
    </div>
  );
};