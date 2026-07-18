import { Outlet } from "react-router-dom";
import { useEffect } from "react";
import AdminSidebar from "@/components/AdminSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { NotificationFAB } from "@/components/notifications/NotificationFAB";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { SidebarToggle } from "@/components/SidebarToggle";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { useAuth } from "@/contexts";
import { useIsMobile } from "@/hooks/useBreakpoint";
import {
  Home,
  Users,
  Briefcase,
  Calendar,
  BookOpen,
  Landmark,
  Wallet,
  UserCog,
  Settings,
  Activity,
  Clock,
  ClipboardList,
} from "lucide-react";
import type { NavGroup, NavLeaf } from "@/components/MobileBottomNav";

const ADV_PARTNER_NAV_GROUPS: NavGroup[] = [
  { title: "Ứng lương", icon: Wallet, path: "/adv-partner/advance-payments" },
  { title: "Người dùng", icon: UserCog, path: "/adv-partner/users" },
];

const ADMIN_NAV_GROUPS: NavGroup[] = [
  { title: "Tổng quan", icon: Home, path: "/admin", end: true },
  { title: "Lương tuần", icon: Clock, path: "/admin/timesheet" },
  { title: "Ứng lương", icon: Wallet, path: "/admin/advance-payments" },
];

const ADMIN_MORE_ITEMS: NavLeaf[] = [
  { title: "Nhật ký", icon: ClipboardList, path: "/admin/audit-log" },
  { title: "API", icon: Activity, path: "/admin/system-health" },
  { title: "Người dùng", icon: UserCog, path: "/admin/users" },
  { title: "Dự án", icon: Briefcase, path: "/admin/projects" },
  { title: "Nhân viên", icon: Users, path: "/admin/employees" },
  { title: "Sổ cái", icon: BookOpen, path: "/admin/ledger" },
  { title: "Khoản vay", icon: Landmark, path: "/admin/loans" },
  { title: "Ví", icon: Wallet, path: "/admin/wallet" },
  { title: "Lịch CV", icon: Calendar, path: "/admin/cron-health" },
  { title: "Cài đặt", icon: Settings, path: "/admin/settings" },
];

const AdminLayoutInner = () => {
  const isMobile = useIsMobile();

  return (
    <div
      className={
        isMobile
          ? "admin-shell relative flex min-h-dvh w-full group/layout"
          : "admin-shell relative flex h-dvh w-full group/layout"
      }
    >
      <AdminSidebar />
      <SidebarToggle />
      <div
        className={
          isMobile
            ? "flex min-w-0 flex-1 flex-col"
            : "flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden"
        }
      >
        <main
          id="main-content"
          className={
            isMobile
              ? "admin-main min-h-dvh flex-1 overflow-visible"
              : "admin-main min-h-0 flex-1 overflow-auto"
          }
        >
          <div className="admin-shell-rail" aria-hidden="true" />
          <div className="admin-page-frame mobile-main-content animate-page-enter">
            <div className="admin-page-inner">
              <SectionErrorBoundary sectionName="trang quản trị">
                <Outlet />
              </SectionErrorBoundary>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

const AdminLayout = () => {
  const { user } = useAuth();
  const isAdvPartner = user?.role === 'adv_partner';

  useEffect(() => {
    document.documentElement.classList.add("admin-route-active");
    return () => document.documentElement.classList.remove("admin-route-active");
  }, []);

  return (
    <ProtectedRoute requiredRole={["admin", "adv_partner"]}>
      <SidebarProvider defaultOpen={true}>
        <div data-admin-ui="" data-theme="congtruong" className="admin-shell-scope">
          <AdminLayoutInner />
          <MobileBottomNav
            groups={isAdvPartner ? ADV_PARTNER_NAV_GROUPS : ADMIN_NAV_GROUPS}
            moreItems={isAdvPartner ? undefined : ADMIN_MORE_ITEMS}
          />
          <NotificationFAB />
        </div>
      </SidebarProvider>
    </ProtectedRoute>
  );
};

export default AdminLayout;
