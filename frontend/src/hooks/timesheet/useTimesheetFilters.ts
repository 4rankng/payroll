import { useState, useMemo } from "react";
import { Timesheet } from "@/types/api/timesheet.types";
import { vietnameseIncludes } from "@/utils/vietnameseNormalization";

export const useTimesheetFilters = (timesheets: Timesheet[]) => {
  const getCurrentMonth = () => {
    const currentDate = new Date();
    const currentMonth = currentDate.getMonth() + 1;
    const currentYear = currentDate.getFullYear();
    return `${currentYear}-${String(currentMonth).padStart(2, '0')}`;
  };

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedMonth, setSelectedMonth] = useState<string>(getCurrentMonth());
  const [selectedProject, setSelectedProject] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<Timesheet['status'] | Timesheet['payment_status'] | 'all'>('all');

  const filteredTimesheets = useMemo(() => {
    return timesheets.filter(timesheet => {
      const matchesSearch =
        (timesheet.employeeName && vietnameseIncludes(timesheet.employeeName, searchTerm)) ||
        (timesheet.projectName && vietnameseIncludes(timesheet.projectName, searchTerm)) ||
        (timesheet.notes && vietnameseIncludes(timesheet.notes, searchTerm));

      const matchesMonth = selectedMonth === "all" || timesheet.date.startsWith(selectedMonth);
      const matchesProject = selectedProject === "all" || timesheet.projectName === selectedProject;
      const matchesStatus = statusFilter === 'all' ||
        timesheet.status === statusFilter ||
        timesheet.payment_status === statusFilter;

      return matchesSearch && matchesMonth && matchesProject && matchesStatus;
    });
  }, [timesheets, searchTerm, selectedMonth, selectedProject, statusFilter]);

  const projects = useMemo(() => {
    return Array.from(new Set(timesheets.map(t => t.projectName).filter(Boolean)));
  }, [timesheets]);

  const hasFilters = searchTerm !== "" || selectedProject !== 'all' || statusFilter !== 'all';

  const clearFilters = () => {
    setSearchTerm("");
    setSelectedProject('all');
    setStatusFilter('all');
  };

  return {
    searchTerm,
    setSearchTerm,
    selectedMonth,
    setSelectedMonth,
    selectedProject,
    setSelectedProject,
    statusFilter,
    setStatusFilter,
    filteredTimesheets,
    projects,
    hasFilters,
    clearFilters,
  };
};