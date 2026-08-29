import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "@/contexts";
import { toast } from "@/components/ui/sonner";
import { AuthLoadingScreen } from "@/components/ui/auth-loading";

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredRole?: "admin" | "partner" | "employee" | "adv_partner" | "accountant" | ("admin" | "partner" | "employee" | "adv_partner" | "accountant")[];
}

const hasRequiredRole = (userRole: string, requiredRole: ProtectedRouteProps["requiredRole"]): boolean => {
  if (!requiredRole) return true;
  if (Array.isArray(requiredRole)) return requiredRole.includes(userRole as ProtectedRouteProps["requiredRole"] extends Array<infer R> ? R : never);
  return userRole === requiredRole;
};

const ProtectedRoute = ({ children, requiredRole }: ProtectedRouteProps) => {
  const navigate = useNavigate();
  const { user, isAuthenticated, isLoading } = useAuth();

  useEffect(() => {
    if (isLoading) return;

    if (!isAuthenticated || !user) {
      // Add a small delay to prevent immediate redirect during initial load
      const timer = setTimeout(() => {
        toast({
          title: "Phiên làm việc đã hết hạn",
          description: "Vui lòng đăng nhập lại để tiếp tục.",
          variant: "destructive",
        });

        // Preserve the current URL (including query params) for redirect after login
        const currentUrl = `${window.location.pathname}${window.location.search}`;
        const loginUrl = `/login?redirect=${encodeURIComponent(currentUrl)}`;
        navigate(loginUrl, { replace: true });
      }, 100);

      return () => clearTimeout(timer);
    }

    if (requiredRole && !hasRequiredRole(user.role, requiredRole)) {
      // Redirect to appropriate dashboard if user has wrong role
      const timer = setTimeout(() => {
        toast({
          title: "Không có quyền truy cập",
          description: "Bạn không có quyền truy cập trang này.",
          variant: "destructive",
        });

        if (user.role === "admin") {
          navigate("/admin", { replace: true });
        } else if (user.role === "partner") {
          navigate("/partner/dashboard", { replace: true });
        } else if (user.role === "adv_partner") {
          navigate("/adv-partner/advance-payments", { replace: true });
        } else if (user.role === "accountant") {
          navigate("/accountant", { replace: true });
        } else if (user.role === "employee") {
          navigate("/employee", { replace: true });
        }
      }, 100);

      return () => clearTimeout(timer);
    }
  }, [isLoading, isAuthenticated, user, requiredRole, navigate]);

  // Listen for token expiry events
  useEffect(() => {
    const handleTokenExpiry = () => {
      toast({
        title: "Phiên làm việc sắp hết hạn",
        description: "Vui lòng lưu công việc và đăng nhập lại.",
        variant: "destructive",
      });
    };

    const handleTokenExpired = () => {
      toast({
        title: "Phiên làm việc đã hết hạn",
        description: "Đang chuyển về trang đăng nhập...",
        variant: "destructive",
      });
    };

    window.addEventListener("token-expiry-warning", handleTokenExpiry);
    window.addEventListener("token-expired", handleTokenExpired);

    return () => {
      window.removeEventListener("token-expiry-warning", handleTokenExpiry);
      window.removeEventListener("token-expired", handleTokenExpired);
    };
  }, []);

  if (isLoading) {
    return <AuthLoadingScreen />;
  }

  if (!isAuthenticated || !user) {
    // Show loading screen instead of null to prevent blank white screen
    // while the useEffect above fires navigate('/login') after its 100ms delay.
    return <AuthLoadingScreen />;
  }

  if (requiredRole && !hasRequiredRole(user.role, requiredRole)) {
    return <AuthLoadingScreen />;
  }

  return <>{children}</>;
};

export default ProtectedRoute;
