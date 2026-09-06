export type AdBannerCTAType = 'phone' | 'url' | 'zalo';

export interface AdBannerCTA {
  label: string;
  type: AdBannerCTAType;
  value: string;
}

/**
 * Wire shape shared by the admin campaign list and the employee resolve
 * endpoint. `updatedAt` doubles as the campaign version the portal keys its
 * dismissal state on — an admin edit re-shows the sheet once.
 */
export interface AdBanner {
  id: number;
  title: string;
  body: string;
  bullets: string[];
  ctas: AdBannerCTA[];
  footer: string;
  targetProjectIds: number[];
  priority: number;
  startsAt: string;
  endsAt: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  /** CTA index → tap count. Admin list only. */
  clickCounts?: Record<string, number>;
}

/** Admin create/update payload. Times are RFC3339 strings. */
export interface AdBannerPayload {
  title: string;
  body: string;
  bullets: string[];
  ctas: AdBannerCTA[];
  footer: string;
  targetProjectIds: number[];
  priority: number;
  startsAt: string;
  endsAt: string;
  isActive: boolean;
}
