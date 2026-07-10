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
      <div className="min-h-[100dvh] bg-[#F6F8FA]">
        <div className="border-b border-[#E4E7EC] bg-white">
          <div className="mx-auto flex max-w-lg items-center justify-between px-4 pb-3" style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}>
            <div className="space-y-2">
              <Skeleton className="h-7 w-44" />
              <Skeleton className="h-4 w-28" />
            </div>
            <div className="flex gap-2">
              <Skeleton className="h-11 w-11 rounded-xl" />
              <Skeleton className="h-11 w-11 rounded-xl" />
            </div>
          </div>
        </div>
        <div className="mx-auto max-w-lg space-y-5 p-4">
          <Skeleton className="h-14 w-full rounded-xl" />
          <Skeleton className="h-48 w-full rounded-2xl" />
          <div className="space-y-2.5">
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-32 w-full rounded-xl" />
          </div>
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
            <h1 className="employee-type-section-title text-slate-900">
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
