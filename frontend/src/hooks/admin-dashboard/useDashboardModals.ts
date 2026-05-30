import { useState } from 'react';
import { Employee } from '@/types/api/employee.types';
import { Project } from '@/types/api/project.types';

export const useDashboardModals = () => {
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [showEmployeeModal, setShowEmployeeModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [showExportModal, setShowExportModal] = useState(false);

  const openEmployeeModal = (employee: Employee | null = null) => {
    setSelectedEmployee(employee);
    setShowEmployeeModal(true);
  };

  const closeEmployeeModal = () => {
    setShowEmployeeModal(false);
    setSelectedEmployee(null);
  };

  const openProjectModal = (project: Project | null = null) => {
    setSelectedProject(project);
    setShowProjectModal(true);
  };

  const closeProjectModal = () => {
    setShowProjectModal(false);
    setSelectedProject(null);
  };

  const openExportModal = () => {
    setShowExportModal(true);
  };

  const closeExportModal = () => {
    setShowExportModal(false);
  };

  return {
    // State
    selectedEmployee,
    selectedProject,
    showEmployeeModal,
    showProjectModal,
    showExportModal,
    // Actions
    openEmployeeModal,
    closeEmployeeModal,
    openProjectModal,
    closeProjectModal,
    openExportModal,
    closeExportModal
  };
};