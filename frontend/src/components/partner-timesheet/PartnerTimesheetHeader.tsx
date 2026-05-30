interface PartnerTimesheetHeaderProps {
  totalEmployees: number;
}

export const PartnerTimesheetHeader = ({ totalEmployees }: PartnerTimesheetHeaderProps) => {
  return (
    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
      <div className="space-y-1">
        <h1 className="typography-headline-medium sm:typography-headline-large lg:typography-display-small text-foreground">
          Bảng công
        </h1>
        <p className="typography-body-small sm:typography-body-medium lg:typography-body-large text-muted-foreground">
          Nhập và Bảng công {totalEmployees} nhân viên
        </p>
      </div>
    </div>
  );
};
