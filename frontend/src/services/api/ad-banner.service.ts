import { apiClient } from './client';
import { API_ENDPOINTS } from '@/config/api.config';
import type { AdBanner, AdBannerPayload } from '@/types/api/ad-banner.types';

// apiClient.get/post/put already return the ApiResponse envelope once-unwrapped
// ({ status, data, message }), so T is the SERVER's data payload itself.

class AdBannerService {
  /**
   * Resolve the single campaign addressing the authenticated employee.
   * Returns null when no campaign targets them (data is null in the envelope).
   */
  async getMyAdBanner(): Promise<AdBanner | null> {
    const result = await apiClient.get<AdBanner | null>(API_ENDPOINTS.adBanners.myBanner);
    return result.data ?? null;
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
    const result = await apiClient.get<{ banners: AdBanner[] }>(API_ENDPOINTS.adBanners.base);
    return result.data?.banners ?? [];
  }

  /** Admin: publish a new campaign. */
  async createAdBanner(payload: AdBannerPayload): Promise<AdBanner> {
    const result = await apiClient.post<AdBanner>(API_ENDPOINTS.adBanners.base, payload);
    if (!result.data) {
      throw new Error(result.message || 'Tạo chiến dịch thất bại');
    }
    return result.data;
  }

  /** Admin: replace a campaign's content and window. */
  async updateAdBanner(id: number, payload: AdBannerPayload): Promise<AdBanner> {
    const result = await apiClient.put<AdBanner>(API_ENDPOINTS.adBanners.byId(id), payload);
    if (!result.data) {
      throw new Error(result.message || 'Cập nhật chiến dịch thất bại');
    }
    return result.data;
  }

  /** Admin: soft-delete a campaign. */
  async deleteAdBanner(id: number): Promise<void> {
    await apiClient.delete(API_ENDPOINTS.adBanners.byId(id));
  }
}

export const adBannerService = new AdBannerService();
