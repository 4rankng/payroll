import { useLocation, useNavigate } from "react-router-dom";
import { useCallback, useEffect } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Home, ArrowLeft } from "lucide-react";
import { authManager } from "@/lib/auth";
import { logNotFoundPath } from "@/lib/logger";

const NotFound = () => {
  const location = useLocation();
  const navigate = useNavigate();

  const isAuthenticated = authManager.isTokenValid();
  const userRole = authManager.getUserRole();

  useEffect(() => {
    logNotFoundPath(location.pathname);
  }, [location.pathname]);

  const handleDashboardNavigation = useCallback(() => {
    navigate(isAuthenticated ? '/' : '/login');
  }, [isAuthenticated, navigate]);

  const handleGoBack = useCallback(() => {
    if (window.history.length > 1) {
      navigate(-1);
    } else {
      handleDashboardNavigation();
    }
  }, [navigate, handleDashboardNavigation]);

  // Pre-memoized navigation handlers to avoid inline arrow functions in JSX
  const handleAdminUsers = useCallback(() => navigate('/admin/users'), [navigate]);
  const handleAdminProjects = useCallback(() => navigate('/admin/projects'), [navigate]);
  const handleAdminEmployees = useCallback(() => navigate('/admin/employees'), [navigate]);
  const handlePartnerEmployees = useCallback(() => navigate('/partner/employees'), [navigate]);
  const handlePartnerTimesheet = useCallback(() => navigate('/partner/timesheet'), [navigate]);

  return (
    <div id="main-content" className="min-h-screen flex items-center justify-center bg-card p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-6xl font-bold text-muted-foreground mb-4">
            404
          </CardTitle>
          <CardTitle className="typography-headline-large">
            Không tìm thấy trang
          </CardTitle>
          <CardDescription className="typography-body-large">
            Trang bạn đang tìm kiếm không tồn tại hoặc đã được di chuyển.
            Vui lòng kiểm tra lại đường dẫn hoặc quay về trang chủ.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Button
              onClick={handleDashboardNavigation}
              className="w-full"
              variant="default"
            >
              <Home className="mr-2 h-4 w-4" />
              {isAuthenticated ? 'Về bảng điều khiển' : 'Đăng nhập'}
            </Button>
            <Button
              onClick={handleGoBack}
              className="w-full"
              variant="outline"
            >
              <ArrowLeft className="mr-2 h-4 w-4" />
              Quay lại
            </Button>
          </div>

          {isAuthenticated && (userRole === 'admin' || userRole === 'partner') && (
            <div className="pt-4 border-t">
              <p className="typography-body-medium text-muted-foreground text-center mb-3">
                Hoặc điều hướng đến:
              </p>
              <div className="grid gap-2">
                {userRole === 'admin' && (
                  <>
                    <Button
                      onClick={handleAdminUsers}
                      variant="ghost"
                      size="sm"
                      className="justify-start"
                    >
                      Người dùng
                    </Button>
                    <Button
                      onClick={handleAdminProjects}
                      variant="ghost"
                      size="sm"
                      className="justify-start"
                    >
                      Dự án
                    </Button>
                    <Button
                      onClick={handleAdminEmployees}
                      variant="ghost"
                      size="sm"
                      className="justify-start"
                    >
                      Nhân viên
                    </Button>
                  </>
                )}
                {userRole === 'partner' && (
                  <>
                    <Button
                      onClick={handlePartnerEmployees}
                      variant="ghost"
                      size="sm"
                      className="justify-start"
                    >
                      Nhân viên
                    </Button>
                    <Button
                      onClick={handlePartnerTimesheet}
                      variant="ghost"
                      size="sm"
                      className="justify-start"
                    >
                      Chấm công
                    </Button>
                  </>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default NotFound;
