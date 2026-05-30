import { toast } from '@/components/ui/sonner';
import type { NewEmployee } from '@/types/api/dashboard.types';
import { useEmployeeModals } from '@/hooks/useModalNavigation';

export const useDashboardActions = () => {
  const { openEmployeeDetails } = useEmployeeModals();

  const handleExport = () => {
    toast({
      title: "Xuất báo cáo thành công",
      description: "File Excel đã được tải về.",
    });
  };

  const handleAddEmployee = () => {
    toast({
      title: "Thêm nhân viên",
      description: "Chức năng thêm nhân viên sẽ được triển khai.",
    });
  };

  const handleCreateProject = () => {
    toast({
      title: "Tạo dự án",
      description: "Chức năng tạo dự án sẽ được triển khai.",
    });
  };

  const handleViewAllActivities = () => {
    toast({
      title: "Xem tất cả hoạt động",
      description: "Chuyển đến trang hoạt động chi tiết.",
    });
  };

  const handleNotificationSettings = () => {
    toast({
      title: "Cài đặt thông báo",
      description: "Mở cài đặt thông báo hệ thống.",
    });
  };

  const handleTaskAction = (taskId: number, taskAction: string) => {
    toast({
      title: `${taskAction} thành công`,
      description: `Đã thực hiện hành động cho tác vụ ID: ${taskId}.`,
    });
  };

  const handleEmployeeClick = (employee: NewEmployee) => {
    openEmployeeDetails(employee.id.toString());
  };

  const handleProjectClick = (projectId: number) => {
    toast({
      title: "Xem chi tiết dự án",
      description: `Chuyển đến trang dự án ID: ${projectId}`,
    });
  };

  const handleNavigateToActivities = () => {
    toast({
      title: "Xem tất cả hoạt động",
      description: "Chuyển đến trang hoạt động chi tiết.",
    });
  };

  const handleRefresh = () => {
    toast({
      title: "Làm mới dữ liệu",
      description: "Dữ liệu dashboard đã được cập nhật.",
    });
  };

  return {
    handleExport,
    handleAddEmployee,
    handleCreateProject,
    handleViewAllActivities,
    handleNotificationSettings,
    handleTaskAction,
    handleEmployeeClick,
    handleProjectClick,
    handleNavigateToActivities,
    handleRefresh
  };
};