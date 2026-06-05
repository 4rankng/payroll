import { useState, useEffect, useMemo, useCallback } from 'react';
import type { TimesheetFormData, TimesheetFormOptions } from '@/types/timesheet';
import { validateTimesheetFormData, validateTimesheetFormDataWithPayrate } from '@/types/timesheet';
import type { Employee } from '@/types/api/employee.types';
import { useCurrentPayRate } from '@/hooks/api/usePayRates';
import { useActiveProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { useTimesheetProjects } from '@/hooks/api/useProjects';
import {
  calculateFormOptions,
  createInitialFormData,
  resetFormForNewEntry
} from '../utils/timesheetHelpers';

interface TimesheetData {
  employee_id: number;
  project_id: number;
  date: string;
  hours_worked: number;
  position: string;
  hour_type: string;
}

interface UseTimesheetFormStateProps {
  timesheet?: TimesheetData;
  employeeId?: number;
  projectId?: number;
  isOpen: boolean;
}

export const useTimesheetFormState = ({
  timesheet,
  employeeId,
  projectId,
  isOpen
}: UseTimesheetFormStateProps) => {
  const [formData, setFormData] = useState<TimesheetFormData>(() =>
    createInitialFormData(timesheet, employeeId, projectId)
  );

  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);
  const [saturdayDayType, setSaturdayDayType] = useState<'Ngày thường' | 'Ngày nghỉ'>('Ngày thường');

  // API hooks
  const { data: payRateData, isLoading: isLoadingPayRate } = useCurrentPayRate(formData.project_id || 0);
  const { data: projectEmployeesData, isLoading: isLoadingProjectEmployees } = useActiveProjectEmployees(
    formData.project_id || 0,
    Boolean(formData.project_id)
  );
  const { data: assignableProjectsData } = useTimesheetProjects({ enabled: isOpen });

  // Derived data
  const availableEmployees = useMemo(() => {
    if (!projectEmployeesData?.data) return [];
    return projectEmployeesData.data.map(assignment => ({
      id: assignment.employee_id,
      fullname: assignment.employee_name,
      cccd: assignment.employee_cccd,
      email: '',
      position: assignment.position,
      current_projects: [],
      assignment
    }));
  }, [projectEmployeesData?.data]);

  const formOptions: TimesheetFormOptions = useMemo(() =>
    calculateFormOptions(payRateData, formData),
    [payRateData, formData]
  );

  const validation = useMemo(() => {
    return validateTimesheetFormDataWithPayrate(formData, formOptions);
  }, [formData, formOptions]);

  // Memoized handlers
  const handleInputChange = useCallback((field: keyof TimesheetFormData, value: string | number) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  }, []);

  const handleProjectChange = useCallback((newProjectId: number) => {
    setFormData(prev => ({
      ...prev,
      project_id: newProjectId,
      employee_id: 0,
      position: '',
      hour_type: ''
    }));
    setSelectedEmployee(null);
  }, []);

  const handleEmployeeChange = useCallback((employee: Employee | null) => {
    const newEmployeeId = employee?.id || 0;
    setSelectedEmployee(employee);
    setFormData(prev => ({
      ...prev,
      employee_id: newEmployeeId
    }));
  }, []);

  const resetForm = useCallback(() => {
    const newFormData = resetFormForNewEntry(formData);
    setFormData(newFormData);
    setSelectedEmployee(null);
    setSaturdayDayType('Ngày thường');
  }, [formData]);

  // Effects for form initialization and auto-population
  useEffect(() => {
    if (timesheet) {
      setFormData(createInitialFormData(timesheet));
      const employee = availableEmployees.find(e => e.id === timesheet.employee_id);
      setSelectedEmployee(employee || null);
    } else {
      setFormData(createInitialFormData(undefined, employeeId, projectId));
      setSelectedEmployee(null);
    }
  }, [timesheet, projectId, employeeId, isOpen, availableEmployees]);

  // Auto-populate position when employee is selected
  useEffect(() => {
    if (selectedEmployee && 'assignment' in selectedEmployee && selectedEmployee.assignment) {
      const assignment = (selectedEmployee as Employee & { assignment: { position: string } }).assignment;
      setFormData(prev => ({
        ...prev,
        position: assignment.position,
        employee_id: selectedEmployee.id
      }));
    }
  }, [selectedEmployee]);

  // Auto-populate project if not provided
  useEffect(() => {
    if (!projectId && !employeeId && assignableProjectsData?.data?.length > 0 && !formData.project_id) {
      const activeProject = assignableProjectsData.data.find(p => p.status === 'active');
      if (activeProject) {
        setFormData(prev => ({ ...prev, project_id: activeProject.id }));
      }
    }
  }, [projectId, employeeId, assignableProjectsData, formData.project_id]);

  // Auto-populate employee from project if not provided
  useEffect(() => {
    if (!employeeId && formData.project_id && availableEmployees.length > 0 && !formData.employee_id) {
      const firstEmployee = availableEmployees[0];
      setFormData(prev => ({ ...prev, employee_id: firstEmployee.id }));
      setSelectedEmployee(firstEmployee);
    }
  }, [employeeId, formData.project_id, availableEmployees, formData.employee_id]);

  // Auto-populate position and hour type
  useEffect(() => {
    if (!formData.position && formOptions.positions.length > 0) {
      setFormData(prev => ({ ...prev, position: formOptions.positions[0] }));
    }
  }, [formOptions.positions, formData.position]);

  useEffect(() => {
    if (!formData.hour_type && formOptions.hourTypes.length > 0) {
      setFormData(prev => ({ ...prev, hour_type: formOptions.hourTypes[0] }));
    }
  }, [formOptions.hourTypes, formData.hour_type]);

  return {
    formData,
    selectedEmployee,
    saturdayDayType,
    setSaturdayDayType,
    availableEmployees,
    formOptions,
    validation,
    isLoadingPayRate,
    isLoadingProjectEmployees,
    handleInputChange,
    handleProjectChange,
    handleEmployeeChange,
    resetForm
  };
};