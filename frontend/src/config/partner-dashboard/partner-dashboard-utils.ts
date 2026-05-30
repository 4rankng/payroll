import { Project, ProjectStats, ProjectStatus } from './partner-dashboard-types';

// Status color mapping
export const getStatusColor = (status: ProjectStatus): string => {
  const colorMap: Record<ProjectStatus, string> = {
    'Đang chạy': 'bg-success',
    'Sắp hoàn thành': 'bg-warning',
    'Hoàn thành': 'bg-info',
    'Tạm dừng': 'bg-destructive',
  };
  return colorMap[status] || 'bg-muted';
};

// Calculate project statistics
export const calculateProjectStats = (projects: Project[]): ProjectStats => {
  return {
    totalProjects: projects.length,
    totalEmployees: projects.reduce((sum, project) => sum + project.employeeCount, 0),
    activeProjects: projects.filter(project => 
      project.status === 'Đang chạy' || project.status === 'Sắp hoàn thành'
    ).length,
  };
};

// Format project display data
export const formatProjectForDisplay = (project: Project) => {
  return {
    ...project,
    statusColor: getStatusColor(project.status),
    displayDate: `Ngày tạo: ${project.startDate}`,
    displayEmployees: `${project.employeeCount} nhân viên`,
  };
};

// Sort projects by status priority
export const sortProjectsByStatus = (projects: Project[]): Project[] => {
  const statusPriority: Record<ProjectStatus, number> = {
    'Đang chạy': 1,
    'Sắp hoàn thành': 2,
    'Hoàn thành': 3,
    'Tạm dừng': 4,
  };

  return [...projects].sort((a, b) => {
    const priorityA = statusPriority[a.status] || 5;
    const priorityB = statusPriority[b.status] || 5;
    return priorityA - priorityB;
  });
};