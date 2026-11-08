import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import PartnerSidebar from "@/components/PartnerSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/shared/ThemeToggle";
import { NotificationFAB } from "@/components/notifications/NotificationFAB";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { Briefcase01, Calendar, Grid01, Users01 } from "@untitledui/icons";
import type { NavGroup } from "@/components/MobileBottomNav";
import { useAuth } from "@/contexts";

const PARTNER_NAV_GROUPS: NavGroup[] = [
  { title: "Tổng quan", icon: Grid01, path: "/partner/dashboard", end: true },
  { title: "Dự án", icon: Briefcase01, path: "/partner/projects" },
  { title: "Nhân viên", icon: Users01, path: "/partner/employees" },
  { title: "Bảng công", icon: Calendar, path: "/partner/timesheet" },
];

const PartnerLayoutInner = ({ isPartner }: { isPartner: boolean }) => {
  useEffect(() => {
    if (!isPartner) return;
    // The toto console design now covers partner routes too: the same token
    // blocks, structural rules and static-ink re-skin layers key off
    // html.admin-console-toto (variables.css maps the partner alias at the
    // --admin-* tokens). partner-route-active keeps the role scope for
    // partner-specific styling.
    document.documentElement.classList.add("partner-route-active", "admin-console-toto");
    return () => document.documentElement.classList.remove("partner-route-active", "admin-console-toto");
  }, [isPartner]);

  return (
    <div
      data-partner-ui={isPartner ? "" : undefined}
      data-theme={isPartner ? "congtruong" : undefined}
      className={isPartner ? "partner-shell-scope relative flex h-dvh w-full group/layout" : "relative flex h-dvh w-full group/layout"}
    >
      {/* Sticky appearance switch, mirroring AdminLayout (phones use the
          page-header toggle inside MobilePageHeader). */}
      {isPartner && (
        <div className="fixed right-3 top-3 z-40 hidden md:block">
          <ThemeToggle className="bg-card/80 shadow-sm backdrop-blur-sm" />
        </div>
      )}
      <PartnerSidebar />
      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <main
          id="main-content"
          className="console-shell min-h-0 flex-1 overflow-auto"
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
