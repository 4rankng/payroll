import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import PartnerSidebar from "@/components/PartnerSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { NotificationFAB } from "@/components/notifications/NotificationFAB";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { SidebarToggle } from "@/components/SidebarToggle";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { Briefcase, Users, Calendar, LayoutDashboard } from "lucide-react";
import type { NavGroup } from "@/components/MobileBottomNav";
import { useAuth } from "@/contexts";

const PARTNER_NAV_GROUPS: NavGroup[] = [
  { title: "Tổng quan", icon: LayoutDashboard, path: "/partner/dashboard", end: true },
  { title: "Dự án", icon: Briefcase, path: "/partner/projects" },
  { title: "Nhân viên", icon: Users, path: "/partner/employees" },
  { title: "Bảng công", icon: Calendar, path: "/partner/timesheet" },
];

const PartnerLayoutInner = ({ isPartner }: { isPartner: boolean }) => {
  useEffect(() => {
    if (!isPartner) return;
    document.documentElement.classList.add("partner-route-active");
    return () => document.documentElement.classList.remove("partner-route-active");
  }, [isPartner]);

  return (
    <div
      data-partner-ui={isPartner ? "" : undefined}
      data-theme={isPartner ? "congtruong" : undefined}
      className={isPartner ? "partner-shell-scope relative flex h-dvh w-full group/layout" : "relative flex h-dvh w-full group/layout"}
    >
      <PartnerSidebar />
      <SidebarToggle />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <main
          id="main-content"
          className="min-h-0 flex-1 overflow-auto bg-[radial-gradient(circle_at_top_left,rgba(8,120,62,0.07),transparent_30rem),linear-gradient(180deg,#f8fafb_0%,#f5f7f9_100%)]"
        >
          <div className="min-h-full mobile-main-content animate-page-enter max-w-[1320px] mx-auto">
            <SectionErrorBoundary sectionName="trang đối tác">
              <Outlet />
            </SectionErrorBoundary>
          </div>
        </main>
      </div>
    </div>
  );
};

const PartnerLayout = () => {
  const { user } = useAuth();
  const isPartner = user?.role === "partner";

  return (
    <ProtectedRoute requiredRole="partner">
      <SidebarProvider defaultOpen={true}>
        <PartnerLayoutInner isPartner={isPartner} />
        <MobileBottomNav groups={PARTNER_NAV_GROUPS} />
        <NotificationFAB />
      </SidebarProvider>
    </ProtectedRoute>
  );
};

export default PartnerLayout;
