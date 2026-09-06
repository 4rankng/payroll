import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type { AdBanner, AdBannerPayload } from '@/types/api/ad-banner.types';

type AdBannerEnvelope<T> = {
  status: string;
  data: T;
  message?: string;
};

type AdBannerListEnvelope = {
  status: string;
  data: { banners: AdBanner[] };
  message?: string;
};

class AdBannerService {
  /**
   * Resolve the single campaign addressing the authenticated employee.
   * Returns null when no campaign targets them (data is null in the envelope).
   */
  async getMyAdBanner(): Promise<AdBanner | null> {
    const response = await apiClient.get<AdBannerEnvelope<AdBanner | null>>(
      API_ENDPOINTS.adBanners.myBanner,
    );
    return response.data?.data ?? null;
  }

  /**
   * Record a CTA tap. Fire-and-forget: a tracking outage must never block
   * the worker from reaching the hotline.
   */
  async recordCTAClick(bannerId: number, ctaIndex: number): Promise<void> {
    await apiClient.post(API_ENDPOINTS.adBanners.click(bannerId), {
      cta_index: ctaIndex,
    });
  }

  /** Admin: every campaign, newest first, with per-CTA click counts. */
  async getAdBanners(): Promise<AdBanner[]> {
    const response = await apiClient.get<AdBannerListEnvelope>(API_ENDPOINTS.adBanners.base);
    return response.data?.data?.banners ?? [];
  }

  /** Admin: publish a new campaign. */
  async createAdBanner(payload: AdBannerPayload): Promise<AdBanner> {
    const response = await apiClient.post<AdBannerEnvelope<AdBanner>>(
      API_ENDPOINTS.adBanners.base,
      payload,
    );
    if (!response.data?.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /** Admin: replace a campaign's content and window. */
  async updateAdBanner(id: number, payload: AdBannerPayload): Promise<AdBanner> {
    const response = await apiClient.put<AdBannerEnvelope<AdBanner>>(
      API_ENDPOINTS.adBanners.byId(id),
      payload,
    );
    if (!response.data?.data) {
      throw new Error('API response missing expected data');
    }
    return response.data.data;
  }

  /** Admin: soft-delete a campaign. */
  async deleteAdBanner(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.adBanners.byId(id));
  }
}

export const adBannerService = new AdBannerService();
