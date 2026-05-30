import { Outlet } from "react-router-dom";
import AdminSidebar from "@/components/AdminSidebar";
import ProtectedRoute from "@/components/ProtectedRoute";
import { SidebarProvider } from "@/components/ui/sidebar";
import { NotificationFAB } from "@/components/notifications/NotificationFAB";
import { MobileBottomNav } from "@/components/MobileBottomNav";
import { SidebarToggle } from "@/components/SidebarToggle";
import { SectionErrorBoundary } from "@/components/ErrorBoundary";
import { useAuth } from "@/contexts";
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
  Bell,
  Activity,
  Clock,
  ClipboardList,
} from "lucide-react";
import type { NavGroup } from "@/components/MobileBottomNav";

const ADV_PARTNER_NAV_GROUPS: NavGroup[] = [
  { title: "Ứng lương", icon: Wallet, path: "/adv-partner/advance-payments" },
  { title: "Người dùng", icon: UserCog, path: "/adv-partner/users" },
];

const ADMIN_NAV_GROUPS: NavGroup[] = [
  { title: "Tổng quan", icon: Home, path: "/admin", end: true },
  {
    title: "Quản lý",
    icon: Users,
    submenu: [
      { title: "Người dùng", icon: UserCog, path: "/admin/users" },
      { title: "Dự án", icon: Briefcase, path: "/admin/projects" },
      { title: "Nhân viên", icon: Users, path: "/admin/employees" },
      { title: "Bảng công", icon: Calendar, path: "/admin/timesheet" },
    ],
  },
  {
    title: "Tài chính",
    icon: BookOpen,
    submenu: [
      { title: "Sổ cái", icon: BookOpen, path: "/admin/transactions" },
      { title: "Khoản vay", icon: Landmark, path: "/admin/loans" },
      { title: "Ứng lương", icon: Wallet, path: "/admin/advance-payments" },
      { title: "Ví", icon: Wallet, path: "/admin/wallet" },
    ],
  },
  {
    title: "Hệ thống",
    icon: Settings,
    submenu: [
      { title: "Kiểm tra API", icon: Activity, path: "/admin/system-health" },
      { title: "Lịch công việc", icon: Clock, path: "/admin/cron-health" },
      { title: "Nhật ký", icon: ClipboardList, path: "/admin/audit-log" },
      { title: "Cài đặt", icon: Settings, path: "/admin/settings" },
      { title: "Gửi thông báo", icon: Bell, path: "/admin/send-notification" },
    ],
  },
];

const AdminLayoutInner = () => {
  return (
    <div className="relative flex h-dvh w-full group/layout">
      <AdminSidebar />
      <SidebarToggle />
      <div className="flex flex-col flex-1 overflow-hidden">
        <main
          id="main-content"
          className="flex-1 overflow-auto bg-gradient-subtle"
        >
          <div className="min-h-full mobile-main-content animate-page-enter max-w-[1280px] mx-auto">
            <SectionErrorBoundary sectionName="trang quản trị">
              <Outlet />
            </SectionErrorBoundary>
          </div>
        </main>
      </div>
    </div>
  );
};

const AdminLayout = () => {
  const { user } = useAuth();
  const isAdvPartner = user?.role === 'adv_partner';
  const navGroups = isAdvPartner ? ADV_PARTNER_NAV_GROUPS : ADMIN_NAV_GROUPS;

  return (
    <ProtectedRoute requiredRole={["admin", "adv_partner"]}>
      <SidebarProvider defaultOpen={true}>
        <AdminLayoutInner />
        <MobileBottomNav groups={navGroups} />
        <NotificationFAB />
      </SidebarProvider>
    </ProtectedRoute>
  );
};

export default AdminLayout;
