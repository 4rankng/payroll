import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { payRateService } from '@/services/api/payrate.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';
import type {
  PayRateListParams,
  CreatePayRateRequest,
  UpdatePayRateRequest,
  RejectPayRateRequest,
  RateCategory
} from '@/types/api/payrate.types';

// Type guard for checking if error has 404 status
function isNotFoundError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false;

  // Check axios error format
  if ('response' in error) {
    const axiosError = error as { response?: { status?: number } };
    return axiosError.response?.status === 404;
  }

  // Check API error format
  if ('http_status' in error) {
    const apiError = error as { http_status?: number };
    return apiError.http_status === 404;
  }

  return false;
}

// ========== PAY RATE QUERIES ==========

/**
 * Get current active or nearest upcoming payrate for a project
 */
export function useCurrentPayRate(projectId: number, enabled = true) {
  return useQuery({
    queryKey: ['projects', projectId, 'payrate'],
    queryFn: async () => {
      const response = await payRateService.getCurrentProjectPayRate(projectId);

      // The API returns the current payrate directly, not an array
      // So we can return the response as-is since it's already in the correct format
      return response.data;
    },
    enabled: enabled && !!projectId,
    gcTime: 15 * 60 * 1000, // 15 minutes
    retry: (failureCount, error: unknown) => {
      // Don't retry on 404 - it means no payrate exists for this project
      if (isNotFoundError(error)) {
        return false;
      }
      // Retry other errors up to 3 times
      return failureCount < 3;
    },
  });
}

/**
 * Get pay rates for a specific project
 */
export function useProjectPayRates(
  projectId: number,
  params: PayRateListParams = {},
  enabled = true
) {
  return useQuery({
    queryKey: ['projects', projectId, 'payrates', params],
    queryFn: () => payRateService.getProjectPayRates(projectId, params),
    enabled,
    gcTime: 15 * 60 * 1000, // 15 minutes
    retry: (failureCount, error: unknown) => {
      // Don't retry on 404 - it means no payrates exist for this project
      if (isNotFoundError(error)) {
        return false;
      }
      // Retry other errors up to 3 times
      return failureCount < 3;
    },
  });
}


/**
 * Get specific pay rate by ID
 */
export function usePayRate(payRateId: number, enabled = true) {
  return useQuery({
    queryKey: ['payrates', payRateId],
    queryFn: () => payRateService.getPayRate(payRateId),
    enabled,
    gcTime: 30 * 60 * 1000, // 30 minutes
  });
}

/**
 * List pay rates with optional filtering
 */
export function usePayRates(params: PayRateListParams = {}, enabled = true) {
  return useQuery({
    queryKey: ['payrates', 'list', params],
    queryFn: () => payRateService.listPayRates(params),
    enabled,
    placeholderData: (previousData) => previousData,
  });
}

// Removed pending pay rate approvals function - no approval workflow needed

// ========== PAY RATE MUTATIONS ==========

/**
 * Create new pay rate configuration
 */
export function useCreatePayRate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreatePayRateRequest) =>
      payRateService.createPayRate(data),
    onSuccess: (result) => {
      const projectId = result.data.project_id;

      // Invalidate and refetch all payrate queries to get fresh data from server
      // This ensures current/upcoming payrates are correctly determined
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrates'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrate'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['payrates', 'list'],
        refetchType: 'active'
      });

      if (result.message) {
        showSuccessNotification(result.message);
      }
    },
  });
}

/**
 * Create pay rate for specific project
 */
export function useCreateProjectPayRate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ projectId, data }: { projectId: number; data: CreatePayRateRequest }) =>
      payRateService.createProjectPayRate(projectId, data),
    onSuccess: (result) => {
      const projectId = result.data.project_id;

      // Invalidate and refetch all payrate queries to get fresh data from server
      // This ensures current/upcoming payrates are correctly determined
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrates'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrate'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['payrates', 'list'],
        refetchType: 'active'
      });

      if (result.message) {
        showSuccessNotification(result.message);
      }
    },
  });
}

/**
 * Update existing pay rate configuration
 */
export function useUpdatePayRate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ payRateId, data }: { payRateId: number; data: UpdatePayRateRequest }) =>
      payRateService.updatePayRate(payRateId, data),
    onSuccess: (result) => {
      const payRateId = result.data.id;
      const projectId = result.data.project_id;

      // Invalidate and refetch all payrate queries to get fresh data from server
      // This ensures current/upcoming payrates are correctly determined after update
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrates'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['projects', projectId, 'payrate'],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['payrates', payRateId],
        refetchType: 'active'
      });
      queryClient.invalidateQueries({
        queryKey: ['payrates', 'list'],
        refetchType: 'active'
      });

      if (result.message) {
        showSuccessNotification(result.message);
      }
    },
  });
}

/**
 * Delete pay rate configuration (Admin only)
 */
export function useDeletePayRate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ payRateId }: { payRateId: number; projectId?: number }) => payRateService.deletePayRate(payRateId),
    onSuccess: (_, variables) => {
      const { payRateId, projectId } = variables ?? { payRateId: undefined, projectId: undefined };
      // Get pay rate data from cache to invalidate related queries
      const payRateData = payRateId ? queryClient.getQueryData(['payrates', payRateId]) : undefined;

      queryClient.removeQueries({ queryKey: ['payrates', payRateId] });
      queryClient.invalidateQueries({ queryKey: ['payrates', 'list'] });

      const targetProjectId = projectId ?? (payRateData ? (payRateData as { project_id: number }).project_id : undefined);
      if (targetProjectId) {
        queryClient.invalidateQueries({ queryKey: ['projects', targetProjectId, 'payrates'] });
        queryClient.invalidateQueries({ queryKey: ['projects', targetProjectId, 'payrate'] });
      }
    },
  });
}

// ========== APPROVAL WORKFLOW REMOVED ==========
// No approval process needed - payrates are immediately active when created

// ========== UTILITY MUTATIONS ==========

/**
 * Validate pay rate configuration
 */
export function useValidatePayRate() {
  return useMutation({
    mutationFn: (data: {
      project_id: number;
      rates: Record<string, RateCategory>;
      effective_from: string;
      effective_to: string;
      exclude_rate_id?: number;
    }) => payRateService.validatePayRate(data),
  });
}

/**
 * Copy pay rate to another project
 */
export function useCopyPayRate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      sourceRateId,
      targetProjectId,
      options
    }: {
      sourceRateId: number;
      targetProjectId: number;
      options: {
        effective_from: string;
        effective_to: string;
        modify_rates?: {
          adjustment_type: 'percentage' | 'fixed';
          adjustment_value: number;
        };
      };
    }) => payRateService.copyPayRate(sourceRateId, targetProjectId, options),
    onSuccess: (result) => {
      const projectId = result.data.project_id;
      queryClient.invalidateQueries({ queryKey: ['projects', projectId, 'payrates'] });
    },
  });
}

/**
 * Create standard Vietnamese pay rate structure
 */

// ========== CONVENIENCE HOOKS ==========

/**
 * Hook for comprehensive pay rate management
 */
export function usePayRateManager(projectId: number) {
  const currentRate = useCurrentPayRate(projectId);
  const projectRates = useProjectPayRates(projectId);
  const createMutation = useCreateProjectPayRate();
  const updateMutation = useUpdatePayRate();
  const validateMutation = useValidatePayRate();

  const createWithValidation = async (data: CreatePayRateRequest) => {
    // Validate first
    const validation = await validateMutation.mutateAsync({
      project_id: projectId,
      rates: data.rates,
      effective_from: data.effective_from,
      effective_to: data.effective_to || null,
    });

    if (!validation.data.valid) {
      throw new Error(`Validation failed: ${validation.data.errors.join(', ')}`);
    }

    if (validation.data.warnings.length > 0) {
      console.warn('Pay rate warnings:', validation.data.warnings);
    }

    return createMutation.mutateAsync({ projectId, data });
  };

  return {
    currentRate,
    projectRates,
    createWithValidation,
    isCreating: createMutation.isPending,
    isValidating: validateMutation.isPending,
    isLoading: currentRate.isLoading || projectRates.isLoading,
    error: createMutation.error || updateMutation.error || validateMutation.error,
  };
}

/**
 * Hook for pay rate validation with real-time feedback
 */
export function usePayRateValidation() {
  const validateMutation = useValidatePayRate();

  const validateInRealTime = async (data: {
    project_id: number;
    rates: Record<string, RateCategory>;
    effective_from: string;
    effective_to: string;
    exclude_rate_id?: number;
  }) => {
    try {
      const result = await validateMutation.mutateAsync(data);
      return result.data;
    } catch (error) {
      return {
        valid: false,
        errors: ['Validation request failed'],
        warnings: [],
        conflicts: []
      };
    }
  };

  return {
    validateInRealTime,
    isValidating: validateMutation.isPending,
    lastValidation: validateMutation.data,
  };
}
