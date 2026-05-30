import { useEmployeeProfile } from "@/hooks/api/useEmployeePortal";
import { Skeleton } from "@/components/ui/skeleton";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import EmployeePage from "@/pages/employee/EmployeePage";
import FlexiblePayEmployeePage from "@/pages/employee/FlexiblePayEmployeePage";

/**
 * Router component that selects the appropriate employee page
 * based on the employee's payment schedule:
 * - "flexible" -> FlexiblePayEmployeePage (salary advance feature)
 * - "weekly" | "monthly" -> EmployeePage (timesheet view)
 */
const EmployeeRouter = () => {
  const { data: profile, isLoading } = useEmployeeProfile();

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
