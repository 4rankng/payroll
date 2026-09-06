import { useMutation, useQuery } from '@tanstack/react-query';

import { adBannerService } from '@/services/api/ad-banner.service';
import { employeeAdBannerKey } from '@/lib/queryKeys';

/**
 * Resolves the ad campaign addressing the authenticated employee. Mirrors the
 * backend's 60s microcache with staleTime so a settled page doesn't re-ask.
 */
export const useEmployeeAdBanner = () =>
  useQuery({
    queryKey: employeeAdBannerKey(),
    queryFn: () => adBannerService.getMyAdBanner(),
    staleTime: 60_000,
    refetchOnWindowFocus: false,
    retry: 1,
  });

/**
 * Fire-and-forget CTA tap recording. The caller navigates (tel:/window.open)
 * without waiting for this mutation — tracking must never gate the hotline.
 */
export const useRecordAdBannerClick = () =>
  useMutation({
    mutationFn: (params: { bannerId: number; ctaIndex: number }) =>
      adBannerService.recordCTAClick(params.bannerId, params.ctaIndex),
  });
