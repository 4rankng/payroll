import { formatVietnameseNumber } from '@/utils/vietnamese';

interface SalaryLabelProps {
  label: string;
  employeeCount: number;
}

export const SalaryLabel = ({ label, employeeCount }: SalaryLabelProps) => {
  return (
    <>
      <div>{label}</div>
      <div className="typography-body-small text-muted-foreground">
        (
        <span className="font-semibold text-primary">
          {formatVietnameseNumber(employeeCount)}
        </span>{' '}
        nhân viên)
      </div>
    </>
  );
};
