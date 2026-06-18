import { useEmployeeProfile } from "@/hooks/api/useEmployeePortal";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import EmployeePage from "@/pages/employee/EmployeePage";
import FlexiblePayEmployeePage from "@/pages/employee/FlexiblePayEmployeePage";
import { AlertCircle, RefreshCw } from "lucide-react";

/**
 * Router component that selects the appropriate employee page
 * based on the employee's payment schedule:
 * - "flexible" -> FlexiblePayEmployeePage (salary advance feature)
 * - "weekly" | "monthly" -> EmployeePage (timesheet view)
 */
const EmployeeRouter = () => {
  const { data: profile, isLoading, isError, refetch } = useEmployeeProfile();

  if (isLoading) {
    return (
      <div className="min-h-[100dvh] bg-muted/50" style={{ paddingTop: 'env(safe-area-inset-top)' }}>
        <div className="sticky top-0 z-10 bg-card border-b border-border shadow-sm">
          <div className="max-w-2xl mx-auto px-4 py-4">
            <Skeleton className="h-12 w-48" />
          </div>
        </div>
        <div className="max-w-2xl mx-auto p-4 space-y-4">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-64 w-full" />
          <Skeleton className="h-48 w-full" />
        </div>
      </div>
    );
  }

  if (isError || !profile) {
    return (
      <SectionErrorBoundary sectionName="trang nhân viên">
        <div
          className="mobile-page min-h-[100dvh] bg-slate-50 px-4 py-8"
          style={{ paddingTop: 'max(env(safe-area-inset-top), 2rem)' }}
        >
          <div className="mx-auto flex min-h-[calc(100dvh-4rem)] max-w-md flex-col items-center justify-center text-center">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-50 text-amber-600 ring-1 ring-amber-200/70">
              <AlertCircle className="h-8 w-8" />
            </div>
            <h1 className="text-xl font-bold tracking-tight text-slate-900">
              Chưa tải được hồ sơ
            </h1>
            <p className="mt-2 text-sm leading-6 text-slate-500">
              Không lấy được thông tin nhân viên. Kiểm tra kết nối hoặc thử tải lại trang.
            </p>
            <Button
              type="button"
              className="mt-5 h-11 gap-2 rounded-xl"
              onClick={() => refetch()}
            >
              <RefreshCw className="h-4 w-4" />
              Tải lại
            </Button>
          </div>
        </div>
      </SectionErrorBoundary>
    );
  }

  // Route based on payment schedule
  if (profile?.payment_schedule === "flexible") {
    return (
      <SectionErrorBoundary sectionName="trang nhân viên">
        <div className="mobile-page"><FlexiblePayEmployeePage /></div>
      </SectionErrorBoundary>
    );
  }

  return (
    <SectionErrorBoundary sectionName="trang nhân viên">
      <div className="mobile-page"><EmployeePage /></div>
    </SectionErrorBoundary>
  );
};

export default EmployeeRouter;
