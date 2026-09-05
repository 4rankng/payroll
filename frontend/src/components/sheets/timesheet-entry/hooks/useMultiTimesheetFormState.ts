import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { timesheetService } from '@/services/api/timesheet.service';
import { PREFERRED_DAY_TYPES } from '@/constants/timesheetConstants';
import {
  mergeAPIEntriesToUI,
} from '@/utils/timesheetTransformers';
import type {
  TimesheetEntry,
  MultiTimesheetFormData,
  MultiTimesheetFormOptions,
  TimesheetValidation
} from '../types/multi-timesheet.types';
import { createEntryFactory } from './utils/entryFactory';
import { validateTimesheetEntries } from './utils/entryValidation';
import { mapApiEntriesToTimesheetEntries, mergeEntriesWithGaps } from './utils/apiEntryMapper';
import { hasEntryChangedFromOriginal } from './utils/entryChange';
import { transformEntriesToAPI } from '@/utils/timesheetTransformers';

// Imported Hooks
import { usePayRateData } from './usePayRateData';
import { useEmployeeData } from './useEmployeeData';
import { useTimesheetFormActions } from './useTimesheetFormActions';
import { usePreviewValidation } from './usePreviewValidation';
import { useFormInitialization } from './useFormInitialization';

interface UseMultiTimesheetFormStateProps {
  employeeId?: number;
  projectId?: number;
  isOpen: boolean;
  isClosing?: boolean;
  projectOffDays?: number;
}

export const useMultiTimesheetFormState = ({
  employeeId,
  projectId,
  isOpen,
  isClosing = false,
  projectOffDays = 0
}: UseMultiTimesheetFormStateProps) => {
  // 1. State Management
  const [formData, setFormData] = useState<MultiTimesheetFormData>(() => ({
    projectId: projectId || 0,
    entries: []
  }));

  const formDataRef = useRef(formData);
  useEffect(() => {
    formDataRef.current = formData;
  }, [formData]);

  // 2. Data Hooks
  const {
    payRateData,
    isLoadingPayRate,
    payRateForbidden,
    isPayRateReady,
    getDayTypesForPosition,
    getHourTypesForEntry,
    hasPayRateForEntry,
    getPayRateForEntry
  } = usePayRateData(formData.projectId || 0);

  const {
    projectEmployeesData,
    isLoadingProjectEmployees,
    availableEmployees,
    availableEmployeesRef,
    selectedEmployees,
    getAvailableProjectsForEmployee
  } = useEmployeeData(formData.projectId || 0, formData.entries, isOpen);

  // 3. Helper Factory
  const projectOffDaysRef = useRef(projectOffDays);
  useEffect(() => {
    projectOffDaysRef.current = projectOffDays;
  }, [projectOffDays]);

  const createNewEntry = useMemo(
    () => createEntryFactory({
      getDayTypesForPosition,
      getFallbackProjectId: () => formDataRef.current.projectId || 0,
      preferredDayTypes: [...PREFERRED_DAY_TYPES],
      getProjectOffDays: () => projectOffDaysRef.current
    }),
    [getDayTypesForPosition]
  );

  // 4. Form Actions Hook
  const {
    handleProjectChange,
    handleAddEmployee,
    handleAddAllEmployees,
    handleAddEntry,
    handleRemoveEntry,
    handleDuplicateEntry,
    handleEntryChange,
    handleDateRangeChange,
    resetForm: resetFormState
  } = useTimesheetFormActions({
    formData,
    setFormData,
    availableEmployees,
    availableEmployeesRef,
    createNewEntry,
    getDayTypesForPosition
  });

  // 5. Validation Hook
  const {
    triggerPreview,
    isPreviewLoading,
    allEntriesValid,
    getValidationErrors,
    getErrorsForEmployee,
    getErrorsForRow,
    resetValidation
  } = usePreviewValidation(formData);

  // 6. Initialization Hook
  useFormInitialization({
    isOpen,
    isClosing,
    projectId,
    employeeId,
    formData,
    setFormData,
    availableEmployees,
    isPayRateReady,
    getDayTypesForPosition,
    handleAddEmployee,
    handleProjectChange,
    projectOffDays
  });

  // 7. Derived Data
  const formOptions: MultiTimesheetFormOptions = useMemo(() => {
    const positions = payRateData?.rates ? Object.keys(payRateData.rates) : [];
    let hourTypes: string[] = [];

    if (payRateData?.rates && positions.length > 0) {
      const firstPosition = positions[0];
      const dayTypes = getDayTypesForPosition(firstPosition);
      if (dayTypes.length > 0) {
        hourTypes = getHourTypesForEntry(firstPosition, dayTypes[0]);
      }
    }

    return {
      positions,
      hourTypes,
      availableEmployees
    };
  }, [availableEmployees, payRateData, getHourTypesForEntry, getDayTypesForPosition]);

  const validation: TimesheetValidation = useMemo(() => {
    return validateTimesheetEntries({
      entries: formData.entries,
      availableEmployees,
      projectEmployeesData
    });
  }, [formData.entries, availableEmployees, projectEmployeesData]);

  // 8. API Integration (Fetch Existing Entries)
  const fetchingRef = useRef(false);

  useEffect(() => {
    if (!isOpen || isClosing) {
      fetchingRef.current = false;
    }
  }, [isOpen, isClosing]);

  const fetchAndMergeExistingEntries = useCallback(async (
    projectId: number,
    startDate: string,
    endDate: string
  ) => {
    if (!projectId || !startDate || !endDate) return;

    if (fetchingRef.current) {
      return;
    }

    fetchingRef.current = true;
    try {
      const data = await timesheetService.getTimesheetEntryTable(projectId, {
        fromDate: startDate,
        toDate: endDate
      });

      const mappedEntries = mapApiEntriesToTimesheetEntries({
        apiEntries: Array.isArray(data) ? data : [],
        projectId,
        availableEmployees: availableEmployeesRef.current,
        getDayTypesForPosition,
        getHourTypesForEntry
      });

      setFormData(prev => ({
        ...prev,
        entries: mergeEntriesWithGaps({
          existingEntries: prev.entries,
          newEntries: mappedEntries,
          startDate,
          endDate,
          projectId,
          createEntry: createNewEntry,
          allProjectEmployees: availableEmployeesRef.current.filter(e =>
            e.current_projects?.some(p => p.project_id === projectId)
          )
        })
      }));
    } catch (error) {
      console.error('Failed to fetch existing timesheet entries:', error);
    } finally {
      fetchingRef.current = false;
    }
  }, [createNewEntry, getDayTypesForPosition, getHourTypesForEntry, availableEmployeesRef]);

  const mergeAPIEntriesToUIWrapper = useCallback((apiEntries: Array<{
    projectId: number;
    employeeId: number;
    date: string;
    hoursWorked: number;
    hourType: string;
    dayType?: string;
  }>) => {
    return mergeAPIEntriesToUI(apiEntries, availableEmployees);
  }, [availableEmployees]);

  const transformChangedEntriesToAPI = useCallback((entries: TimesheetEntry[]) => {
    const changedEntries = entries.filter(hasEntryChangedFromOriginal);
    return transformEntriesToAPI(changedEntries);
  }, []);

  const resetForm = useCallback((targetProjectId?: number) => {
    fetchingRef.current = false; // allow next fetch after reset
    resetValidation();
    resetFormState(targetProjectId !== undefined ? targetProjectId : projectId);
  }, [resetValidation, resetFormState, projectId]);

  return {
    formData,
    selectedEmployees,
    availableEmployees,
    formOptions,
    validation,
    isLoadingPayRate,
    payRateForbidden,
    isPayRateReady,
    isLoadingProjectEmployees,
    getDayTypesForPosition,
    getHourTypesForEntry,
    hasPayRateForEntry,
    getPayRateForEntry,
    getAvailableProjectsForEmployee,
    transformEntriesToAPI: transformChangedEntriesToAPI,
    mergeAPIEntriesToUI: mergeAPIEntriesToUIWrapper,
    handleProjectChange,
    handleAddEmployee,
    handleAddAllEmployees,
    handleAddEntry,
    handleRemoveEntry,
    handleDuplicateEntry,
    handleEntryChange,
    handleDateRangeChange,
    resetForm,
    triggerPreview,
    isPreviewLoading,
    allEntriesValid,
    getValidationErrors,
    getErrorsForEmployee,
    getErrorsForRow,
    fetchAndMergeExistingEntries
  };
};
