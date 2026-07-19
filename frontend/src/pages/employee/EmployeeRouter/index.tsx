import { useEmployeeProfile } from "@/hooks/api/useEmployeePortal";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import EmployeePage from "@/pages/employee/EmployeePage";
import FlexiblePayEmployeePage from "@/pages/employee/FlexiblePayEmployeePage";
import { AlertCircle, RefreshCw } from "lucide-react";
import { EmployeeMobileShell } from "@/components/employees/EmployeeMobileShell";
import { cn } from "@/lib/utils";

/**
 * Router component that selects the appropriate employee page
 * based on the employee's payment schedule:
 * - "flexible" -> FlexiblePayEmployeePage (salary advance feature)
 * - "weekly" | "monthly" -> EmployeePage (timesheet view)
 */
const EmployeeRouter = () => {
  const { data: profile, isLoading, isError, isFetching, refetch } = useEmployeeProfile();
  const handleRetry = () => {
    if (isFetching) return;
    void refetch();
  };

  if (isLoading) {
    return (
      <EmployeeMobileShell chrome="skeleton" contentClassName="max-w-lg space-y-5">
        <Skeleton className="h-14 w-full rounded-xl" />
        <Skeleton className="h-48 w-full rounded-2xl" />
        <div className="space-y-2.5">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-32 w-full rounded-xl" />
        </div>
      </EmployeeMobileShell>
    );
  }

  if (!profile) {
    return (
      <SectionErrorBoundary sectionName="trang nhân viên">
        <EmployeeMobileShell chrome="error" contentClassName="max-w-md">
          <div className="flex min-h-[calc(100dvh-9rem)] flex-col items-center justify-center text-center" role="alert">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-50 text-amber-600 ring-1 ring-amber-200/70">
              <AlertCircle className="h-8 w-8" />
            </div>
            <h1 className="employee-type-section-title text-slate-900">
              Chưa tải được hồ sơ
            </h1>
            <p className="mt-2 text-sm leading-6 text-slate-500">
              Không lấy được thông tin nhân viên. Kiểm tra kết nối hoặc thử tải lại trang.
            </p>
            <Button
              type="button"
              className="mt-5 h-11 gap-2 rounded-xl"
              onClick={handleRetry}
              disabled={isFetching}
            >
              <RefreshCw className={cn("h-4 w-4", isFetching && "animate-spin")} />
              {isFetching ? "Đang tải lại…" : "Tải lại"}
            </Button>
          </div>
        </EmployeeMobileShell>
      </SectionErrorBoundary>
    );
  }

  const employeePage = profile.payment_schedule === "flexible"
    ? <FlexiblePayEmployeePage />
    : <EmployeePage />;

  return (
    <SectionErrorBoundary sectionName="trang nhân viên">
      <div className="mobile-page">{employeePage}</div>
    </SectionErrorBoundary>
  );
};

export default EmployeeRouter;
