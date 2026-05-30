import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

/**
 * Navigation hook for dashboard stat cards
 * Provides consistent deeplink navigation for all dashboard statistics
 */
export function useDashboardNavigation() {
  const navigate = useNavigate();

  const navigateToEmployees = useCallback((filter?: string) => {
    const params = new URLSearchParams();
    if (filter) {
      params.set('filter', filter);
    }
    navigate(`/admin/employees?${params.toString()}`);
  }, [navigate]);

  const navigateToProjects = useCallback((filter?: string) => {
    const params = new URLSearchParams();
    if (filter) {
      params.set('status', filter);
    }
    navigate(`/admin/projects?${params.toString()}`);
  }, [navigate]);

  const navigateToLedger = useCallback((filter?: string) => {
    const params = new URLSearchParams();
    if (filter) {
      params.set('period', filter);
    }
    navigate(`/admin/transactions?${params.toString()}`);
  }, [navigate]);

  return {
    // Employee-related navigation
    navigateToActiveEmployees: useCallback(() => {
      const params = new URLSearchParams();
      params.set('status', 'working');
      navigate(`/admin/employees?${params.toString()}`);
    }, [navigate]),

    navigateToNewEmployees: useCallback(() => {
      const now = new Date();
      const year = now.getFullYear();
      const month = String(now.getMonth() + 1).padStart(2, '0');
      const day = String(now.getDate()).padStart(2, '0');

      const fromDate = `${year}-${month}-01`;
      const toDate = `${year}-${month}-${day}`;

      const params = new URLSearchParams();
      params.set('fromDate', fromDate);
      params.set('toDate', toDate);
      navigate(`/admin/employees?${params.toString()}`);
    }, [navigate]),

    navigateToAllEmployees: useCallback(() => {
      navigate('/admin/employees');
    }, [navigate]),

    // Project-related navigation
    navigateToActiveProjects: useCallback(() => {
      navigateToProjects('active');
    }, [navigateToProjects]),

    // Salary/Ledger navigation
    navigateToSalaryLedger: useCallback(() => {
      navigateToLedger('salary');
    }, [navigateToLedger]),

    navigateToCurrentMonthSalary: useCallback(() => {
      navigateToLedger('current-month');
    }, [navigateToLedger]),

    // Timesheet-related navigation
    navigateToPendingApprovals: useCallback(() => {
      const params = new URLSearchParams();
      params.set('status', 'pending_approval');
      navigate(`/admin/timesheet?${params.toString()}`);
    }, [navigate])
  };
}
