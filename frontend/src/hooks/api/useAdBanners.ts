import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { adminAdBannersKey, employeeAdBannerKey } from '@/lib/queryKeys';
import { adBannerService } from '@/services/api/ad-banner.service';
import type { AdBannerPayload } from '@/types/api/ad-banner.types';

/** Admin campaign list with click stats. */
export const useAdBanners = () =>
  useQuery({
    queryKey: adminAdBannersKey(),
    queryFn: () => adBannerService.getAdBanners(),
  });

const useInvalidateAdBanners = () => {
  const queryClient = useQueryClient();
  return () => {
    // The employee resolve cache mirrors the backend microcache; dropping it
    // keeps a just-published campaign instantly visible to the composer's
    // own verification.
    void queryClient.invalidateQueries({ queryKey: adminAdBannersKey() });
    void queryClient.invalidateQueries({ queryKey: employeeAdBannerKey() });
  };
};

export const useCreateAdBanner = () => {
  const invalidate = useInvalidateAdBanners();
  return useMutation({
    mutationFn: (payload: AdBannerPayload) => adBannerService.createAdBanner(payload),
    onSuccess: invalidate,
  });
};

export const useUpdateAdBanner = () => {
  const invalidate = useInvalidateAdBanners();
  return useMutation({
    mutationFn: (params: { id: number; payload: AdBannerPayload }) =>
      adBannerService.updateAdBanner(params.id, params.payload),
    onSuccess: invalidate,
  });
};

export const useDeleteAdBanner = () => {
  const invalidate = useInvalidateAdBanners();
  return useMutation({
    mutationFn: (id: number) => adBannerService.deleteAdBanner(id),
    onSuccess: invalidate,
  });
};
