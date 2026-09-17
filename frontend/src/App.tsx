import React, { Suspense } from "react";
import { lazyWithReload as lazy } from "@/lib/lazy";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AppProviders } from "@/contexts";
import { QueryClient, QueryClientProvider, MutationCache, QueryCache } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router-dom";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ResponsivePage } from "@/components/ResponsivePage";
import Login from "./pages/Login";
import OTPLogin from "./pages/OTPLogin";
import ForgotPassword from "./pages/ForgotPassword";
import ResetPassword from "./pages/ResetPassword";
import ZaloResetPassword from "./pages/ZaloResetPassword";
import AdminLayout from "./layouts/AdminLayout";
import PartnerLayout from "./layouts/PartnerLayout";

// Admin Desktop Pages (lazy-loaded)
const AdminDashboard = lazy(() => import("./pages/admin/DashboardPage"));
const UsersPage = lazy(() => import("./pages/admin/UsersPage"));
const ProjectsPage = lazy(() => import("./pages/admin/ProjectsPage"));
const EmployeesPage = lazy(() => import("./pages/admin/EmployeesPage"));
const TimesheetPage = lazy(() => import("./pages/admin/TimesheetPage"));
const TransactionsPage = lazy(() => import("./pages/admin/TransactionsPage"));
const LoansPage = lazy(() => import("./pages/admin/LoansPage"));
const AdvancePaymentsPage = lazy(() => import("./pages/admin/AdvancePaymentsPage"));
const AdvPartnerAdvancePaymentsPage = lazy(() => import("./pages/admin/AdvancePaymentsPage/AdvPartnerView"));
const CheckInSettingsPage = lazy(() => import("./components/advance-payment/CheckInSettingsPage"));
const SettingsPage = lazy(() => import("./pages/admin/SettingsPage"));
const WalletPage = lazy(() => import("./pages/admin/WalletPage"));
const EmailPage = lazy(() => import("./pages/admin/EmailPage"));
const SystemHealthPage = lazy(() => import("./pages/admin/SystemHealthPage"));
const CronHealthPage = lazy(() => import("./pages/admin/CronHealthPage"));
const PayrateEditPage = lazy(() => import("./pages/admin/PayrateEditPage"));
const AuditLogPage = lazy(() => import("./pages/admin/AuditLogPage"));
const AdminPaymentHistoryPage = lazy(() => import("./pages/admin/PaymentHistoryPage"));

// Admin Mobile Pages (lazy-loaded)
const AdminDashboardMobile = lazy(() => import("./pages/mobile/admin/DashboardPage"));
const UsersPageMobile = lazy(() => import("./pages/mobile/admin/UsersPage"));
const ProjectsPageMobile = lazy(() => import("./pages/mobile/admin/ProjectsPage"));
const EmployeesPageMobile = lazy(() => import("./pages/mobile/admin/EmployeesPage"));
const TimesheetPageMobile = lazy(() => import("./pages/mobile/admin/TimesheetPage"));
const LedgerEntriesPageMobile = lazy(() => import("./pages/mobile/admin/LedgerEntriesPage"));
const LoansPageMobile = lazy(() => import("./pages/mobile/admin/LoansPage"));
const AdvancePaymentsPageMobile = lazy(() => import("./pages/mobile/admin/AdvancePaymentsPage"));
const AdvancePaymentsEmployeeListPageMobile = lazy(() => import("./pages/mobile/admin/AdvancePaymentsPage/EmployeeListPage"));
const LendersPageMobile = lazy(() => import("./pages/mobile/admin/LoansPage/LendersPage"));
const ActivityUsersPageMobile = lazy(() => import("./pages/mobile/admin/DashboardPage/ActivityUsersPage"));
const SettingsPageMobile = lazy(() => import("./pages/mobile/admin/SettingsPage"));
const SystemHealthPageMobile = lazy(() => import("./pages/mobile/admin/SystemHealthPage"));
const CronHealthPageMobile = lazy(() => import("./pages/mobile/admin/CronHealthPage"));
const WalletPageMobile = lazy(() => import("./pages/mobile/admin/WalletPage"));
const AuditLogPageMobile = lazy(() => import("./pages/mobile/admin/AuditLogPage"));
const PayrateEditPageMobile = lazy(() => import("./pages/mobile/admin/PayrateEditPage"));

// Partner Desktop Pages (lazy-loaded)
const PartnerProjectsPage = lazy(() => import("./pages/partner/ProjectsPage"));
const PartnerEmployeesPage = lazy(() => import("./pages/partner/EmployeesPage"));
const PartnerTimesheetsPage = lazy(() => import("./pages/partner/TimesheetsPage"));
const PartnerDashboardPage = lazy(() => import("./pages/partner/DashboardPage"));
const PartnerPaymentHistoryPage = lazy(() => import("./pages/partner/PaymentHistoryPage"));

// Partner Mobile Pages (lazy-loaded)
const PartnerProjectsPageMobile = lazy(() => import("./pages/mobile/partner/ProjectsPage"));
const PartnerEmployeesPageMobile = lazy(() => import("./pages/mobile/partner/EmployeesPage"));
const PartnerTimesheetsPageMobile = lazy(() => import("./pages/mobile/partner/TimesheetsPage"));
const PartnerDashboardMobile = lazy(() => import("./pages/mobile/partner/DashboardPage"));

// Adv Partner Desktop Pages (lazy-loaded)
const AdvPartnerUsersPage = lazy(() => import("./pages/adv-partner/UsersPage"));
const AccountantPage = lazy(() => import("./pages/accountant/AccountantPage"));

// Employee Pages (lazy-loaded)
const EmployeeRouter = lazy(() => import("./pages/employee/EmployeeRouter"));

import NotFound from "./pages/NotFound";
import ProtectedRoute from "./components/ProtectedRoute";
import { authManager } from "@/lib/auth";
import { SecureModalProvider } from "@/components/modals/SecureModalProvider";
import ModalRouter from "@/components/modals/ModalRouter";
import { EmailPromptGate } from "@/components/EmailPromptGate";
import { ForceChangePasswordDialog } from "@/components/ForceChangePasswordDialog";
import { showErrorNotification, isNetworkError, handleNetworkError } from "@/utils/error-handler";
import { initInvalidationService } from "@/lib/cache/invalidationService";
import { setupQueryPersistence } from "@/lib/cache/queryPersister";
import { isIncognitoMode } from "@/utils/dom-safety";
import { EnvironmentBanner } from "@/components/EnvironmentBanner";
import { developmentBannerText, shouldShowDevelopmentBanner } from "@/utils/environment";

// Minimal suspense fallback for lazy-loaded pages
const PageSuspenseFallback = React.memo(() => (
  <div className="flex h-[50vh] items-center justify-center">
    <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
  </div>
));

const LogoutRedirect = () => {
  authManager.removeToken();
  return <Navigate to="/login" replace />;
};

// Component to handle root redirects safely
const RootRedirect = () => {
  const isTokenValid = authManager.isTokenValid();
  const userRole = isTokenValid ? authManager.getUserRole() : null;

  if (isTokenValid && userRole === "admin") {
    return <Navigate to="/admin" replace />;
  }
  if (isTokenValid && userRole === "partner") {
    return <Navigate to="/partner/dashboard" replace />;
  }
  if (isTokenValid && userRole === "adv_partner") {
    return <Navigate to="/adv-partner/advance-payments" replace />;
  }
  if (isTokenValid && userRole === "accountant") {
    return <Navigate to="/accountant" replace />;
  }
  if (isTokenValid && userRole === "employee") {
    return <Navigate to="/employee" replace />;
  }
  return <Navigate to="/login" replace />;
};

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error) => {
      // Handle network errors specifically
      if (isNetworkError(error)) {
        handleNetworkError();
        return;
      }

      // For other query errors, only show notification if it's a critical error
      // Most query errors are handled by components showing loading states
      console.error('Query Error:', error);
    },
  }),
  mutationCache: new MutationCache({
    onSuccess: (_data, _variables, _context, mutation) => {
      // Only invalidate related queries based on mutation metadata
      // This prevents unnecessary refetching of unrelated data
      if (mutation.meta?.invalidates) {
        const queryKeys = Array.isArray(mutation.meta.invalidates)
          ? mutation.meta.invalidates
          : [mutation.meta.invalidates];
        queryKeys.forEach((queryKey: unknown[]) => {
          queryClient.invalidateQueries({ queryKey });
        });
      }
    },
    onError: (error, _variables, _context, mutation) => {
      // Skip error notification if the mutation has its own onError handler
      // This prevents double notifications
      if (mutation.options.onError) {
        return;
      }

      // Skip error notification if the mutation explicitly opts out
      if (mutation.meta?.skipGlobalError) {
        return;
      }

      // Handle network errors specifically
      if (isNetworkError(error)) {
        handleNetworkError();
        return;
      }

      // Show error notification for all other mutation failures
      showErrorNotification(error);
    },
  }),
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000, // 30 seconds
      gcTime: 30 * 60 * 1000, // 30 minutes garbage collection
      // Note: keepPreviousData is now placeholderData in TanStack Query v5
      placeholderData: (previousData: unknown) => previousData,
    },
  },
});

// Initialize the global invalidation service
initInvalidationService(queryClient);

// Persist query cache to sessionStorage (survives page refresh)
setupQueryPersistence(queryClient);

// Component that uses location hook inside Router context
const AppContent = () => {
  const location = useLocation();

  const { headline: developmentHeadline, detail: developmentDetail } = developmentBannerText;

  // Add DOM error recovery for production
  React.useEffect(() => {
    const handleError = (event: ErrorEvent) => {
      if (event.error?.message?.includes('removeChild') ||
          event.error?.message?.includes('insertBefore') ||
          event.error?.message?.includes('appendChild')) {

        console.warn('DOM manipulation error detected, preventing crash:', event.error.message);
        event.preventDefault();

        // Log if in incognito mode for debugging
        isIncognitoMode().then(isIncognito => {
          if (!isIncognito) {
            console.info('DOM error occurred in normal browsing mode - likely browser extension interference');
          }
        });
      }
    };

    window.addEventListener('error', handleError);
    return () => window.removeEventListener('error', handleError);
  }, []);

  const developmentBannerElement = React.useMemo(() => {
    if (!shouldShowDevelopmentBanner) {
      return null;
    }

    return (
      <EnvironmentBanner
        headline={developmentHeadline}
        detail={developmentDetail}
      />
    );
  }, [developmentDetail, developmentHeadline]);

  return (
    <>
      {developmentBannerElement}
      <Toaster />
      <Suspense fallback={<PageSuspenseFallback />}>
      <Routes>
        {/* Redirect root to appropriate dashboard or login */}
        <Route
          path="/"
          element={<RootRedirect />}
        />
        {/* no /home route by design */}

        {/* Login Route */}
        <Route path="/login" element={<Login />} />
        {/* OTP second-step login (admin/partner when OTP_ENABLE is on) */}
        <Route path="/login/otp" element={<OTPLogin />} />
        {/* Self-service password reset (email magic link + Zalo OTP) */}
        <Route path="/forgot-password" element={<ForgotPassword />} />
        <Route path="/reset-password" element={<ResetPassword />} />
        <Route path="/zalo-reset-password" element={<ZaloResetPassword />} />

        {/* Logout */}
        <Route path="/logout" element={<LogoutRedirect />} />

        {/* Admin Routes */}
        <Route path="/admin" element={<ProtectedRoute requiredRole="admin"><AdminLayout /></ProtectedRoute>}>
          <Route index element={<ResponsivePage desktopComponent={AdminDashboard} mobileComponent={AdminDashboardMobile} />} />
          <Route path="dashboard/activity" element={<ActivityUsersPageMobile />} />
          <Route path="users" element={<ResponsivePage desktopComponent={UsersPage} mobileComponent={UsersPageMobile} />} />
          <Route path="projects" element={<ResponsivePage desktopComponent={ProjectsPage} mobileComponent={ProjectsPageMobile} />} />
          <Route path="employees" element={<ResponsivePage desktopComponent={EmployeesPage} mobileComponent={EmployeesPageMobile} />} />
          <Route path="timesheet" element={<ResponsivePage desktopComponent={TimesheetPage} mobileComponent={TimesheetPageMobile} />} />
          <Route path="payment-history" element={<AdminPaymentHistoryPage />} />
          <Route path="timesheets" element={<Navigate to="/admin/timesheet" replace />} />
          {/* Legacy "Giao dịch" entry point — Sổ cái (/admin/ledger) is the single
              double-entry workspace; old links redirect there. */}
          <Route path="transactions" element={<Navigate to="/admin/ledger" replace />} />
          <Route path="ledger" element={<ResponsivePage desktopComponent={TransactionsPage} mobileComponent={LedgerEntriesPageMobile} />} />
          <Route path="loans" element={<ResponsivePage desktopComponent={LoansPage} mobileComponent={LoansPageMobile} />} />
          <Route path="loans/lenders" element={<LendersPageMobile />} />
          <Route path="advance-payments" element={<ResponsivePage desktopComponent={AdvancePaymentsPage} mobileComponent={AdvancePaymentsPageMobile} />} />
          <Route path="advance-payments/check-in-settings" element={<CheckInSettingsPage />} />
          <Route path="advance-payments/employees" element={<AdvancePaymentsEmployeeListPageMobile />} />
          <Route path="advance-payment-fees" element={<Navigate to="/admin/settings?tab=fee-config" replace />} />
          <Route path="approvals" element={<ResponsivePage desktopComponent={TimesheetPage} mobileComponent={TimesheetPageMobile} />} />
          <Route path="settings" element={<ResponsivePage desktopComponent={SettingsPage} mobileComponent={SettingsPageMobile} />} />
          <Route path="wallet" element={<ResponsivePage desktopComponent={WalletPage} mobileComponent={WalletPageMobile} />} />
          <Route path="system" element={<Navigate to="/admin/system-health" replace />} />
          <Route path="system-health" element={<ResponsivePage desktopComponent={SystemHealthPage} mobileComponent={SystemHealthPageMobile} />} />
          <Route path="cron-health" element={<ResponsivePage desktopComponent={CronHealthPage} mobileComponent={CronHealthPageMobile} />} />
          <Route path="audit-log" element={<ResponsivePage desktopComponent={AuditLogPage} mobileComponent={AuditLogPageMobile} />} />
          <Route path="send-notification" element={<Navigate to="/admin/settings?tab=notifications" replace />} />
          <Route path="email" element={<EmailPage />} />
          <Route path="projects/:projectId/payrates/:payrateId/edit" element={<ResponsivePage desktopComponent={PayrateEditPage} mobileComponent={PayrateEditPageMobile} />} />
          <Route path="*" element={<NotFound />} />
        </Route>

        {/* Partner Routes */}
        <Route path="/partner" element={<ProtectedRoute requiredRole="partner"><PartnerLayout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/partner/dashboard" replace />} />
          <Route path="dashboard" element={<ResponsivePage desktopComponent={PartnerDashboardPage} mobileComponent={PartnerDashboardMobile} />} />
          <Route path="projects" element={<ResponsivePage desktopComponent={PartnerProjectsPage} mobileComponent={PartnerProjectsPageMobile} />} />
          <Route path="employees" element={<ResponsivePage desktopComponent={PartnerEmployeesPage} mobileComponent={PartnerEmployeesPageMobile} />} />
          <Route path="timesheet" element={<ResponsivePage desktopComponent={PartnerTimesheetsPage} mobileComponent={PartnerTimesheetsPageMobile} />} />
          <Route path="timesheet/payment-history" element={<PartnerPaymentHistoryPage />} />
          <Route path="projects/:projectId/payrates/:payrateId/edit" element={<ResponsivePage desktopComponent={PayrateEditPage} mobileComponent={PayrateEditPageMobile} />} />
        </Route>

        {/* Employee Routes */}
        <Route path="/employee" element={<ProtectedRoute requiredRole="employee"><EmployeeRouter /></ProtectedRoute>} />

        {/* Accountant Routes — single no-sidebar workspace (duyệt công, chuyển lô, sao kê) */}
        <Route path="/accountant" element={<ProtectedRoute requiredRole="accountant"><AccountantPage /></ProtectedRoute>} />

        {/* Advance Payment Partner Routes */}
        <Route path="/adv-partner" element={<ProtectedRoute requiredRole="adv_partner"><AdminLayout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/adv-partner/advance-payments" replace />} />
          <Route
            path="advance-payments"
            element={<ResponsivePage desktopComponent={AdvPartnerAdvancePaymentsPage} mobileComponent={AdvancePaymentsPageMobile} />}
          />
          <Route path="advance-payments/check-in-settings" element={<CheckInSettingsPage />} />
          <Route path="advance-payments/employees" element={<AdvancePaymentsEmployeeListPageMobile />} />
          <Route
            path="users"
            element={<AdvPartnerUsersPage />}
          />
          <Route path="*" element={<NotFound />} />
        </Route>

        {/* 404 Route */}
        <Route path="*" element={<NotFound />} />
      </Routes>
      </Suspense>

      {/* Modal Router - centralized URL-driven modal system */}
      <ModalRouter />

      {/* Prompts admin/partner/adv_partner users with no email to provide one. */}
      <EmailPromptGate />

      {/* Blocks employee accounts still on the default password behind a compulsory change-password dialog. */}
      <ForceChangePasswordDialog />
    </>
  );
};

const App = () => {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <TooltipProvider>
          <BrowserRouter>
            <SecureModalProvider>
              <AppProviders>
                <AppContent />
              </AppProviders>
            </SecureModalProvider>
          </BrowserRouter>
        </TooltipProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
};

export default App;
