import { useQuery } from '@tanstack/react-query';
import { systemHealthService } from '@/services/api/system-health.service';
import { QueryKeys } from '@/lib/queryKeys';

export const useAPISummary = (days = 1, sortBy?: string) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.apiSummary(days, sortBy),
    queryFn: () => systemHealthService.getAPISummary({ days, sortBy }),
  });

export const useErrorsByUser = (days = 14) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.errorsByUser(days),
    queryFn: () => systemHealthService.getErrorsByUser({ days }),
  });

export const useLatencyTrend = (days = 7, groupBy = 'day') =>
  useQuery({
    queryKey: QueryKeys.systemHealth.latencyTrend(days, groupBy),
    queryFn: () => systemHealthService.getLatencyTrend({ days, groupBy }),
  });

export const useSlowestEndpoints = (days = 7) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.slowestEndpoints(days),
    queryFn: () => systemHealthService.getSlowestEndpoints({ days }),
  });

export const useRecentErrors = (days = 1) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.recentErrors(days),
    queryFn: () => systemHealthService.getRecentErrors({ days }),
  });

export const useErrorCount = (days = 1) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.errorCount(days),
    queryFn: () => systemHealthService.getErrorCount({ days }),
  });

export const useEventBusMetrics = () =>
  useQuery({
    queryKey: QueryKeys.systemHealth.eventBus(),
    queryFn: () => systemHealthService.getEventBusMetrics(),
  });

export const useCacheMetrics = () =>
  useQuery({
    queryKey: QueryKeys.systemHealth.cache(),
    queryFn: () => systemHealthService.getCacheMetrics(),
  });

export const useTopEndpoints = (days = 7) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.topEndpoints(days),
    queryFn: () => systemHealthService.getTopEndpoints({ days }),
  });

export const useEndpointLatencyTrend = (endpointId: number, days = 7, groupBy = 'day') =>
  useQuery({
    queryKey: QueryKeys.systemHealth.endpointTrend(endpointId, days, groupBy),
    queryFn: () => systemHealthService.getEndpointLatencyTrend({ endpointId, days, groupBy }),
    enabled: endpointId > 0,
  });

export const useFailedLogins = (days = 7, limit = 100) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.failedLogins(days),
    queryFn: () => systemHealthService.getFailedLogins({ days, limit }),
  });

export const useFailedLoginsByIdentifier = (identifier: string, days = 7, enabled = false) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.failedLoginsByIdentifier(identifier, days),
    queryFn: () => systemHealthService.getFailedLoginsByIdentifier(identifier, { days }),
    enabled: enabled && identifier !== '',
  });

export const useBrowserPlatformStats = (days = 30) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.browserPlatformStats(days),
    queryFn: () => systemHealthService.getBrowserPlatformStats({ days }),
  });

export const useBrowserPlatformUsers = (browser: string, platform: string, days = 30, enabled = false) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.browserPlatformUsers(browser, platform, days),
    queryFn: () => systemHealthService.getBrowserPlatformUsers({ browser, platform, days }),
    enabled: enabled && browser !== '' && platform !== '',
  });

export const useOSStats = (days = 30) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.osStats(days),
    queryFn: () => systemHealthService.getOSStats({ days }),
  });

export const useOSUsers = (osFamily: string, days = 30, enabled = false) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.osUsers(osFamily, days),
    queryFn: () => systemHealthService.getOSUsers({ osFamily, days }),
    enabled: enabled && osFamily !== '',
  });

export const useBrowserStats = (days = 30) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.browserStats(days),
    queryFn: () => systemHealthService.getBrowserStats({ days }),
  });

export const useBrowserUsers = (browserFamily: string, days = 30, enabled = false) =>
  useQuery({
    queryKey: QueryKeys.systemHealth.browserUsers(browserFamily, days),
    queryFn: () => systemHealthService.getBrowserUsers({ browserFamily, days }),
    enabled: enabled && browserFamily !== '',
  });
