// Common fee-schedule presets shown as one-click buttons in the create form.
// Lets admins skip building common configurations from scratch.

import type { FeeScheduleFormState } from "./types";
import { DEFAULT_MIN_FEE_VND, tomorrowISO } from "./form-helpers";

export interface FeeSchedulePreset {
  key: string;
  label: string;
  description: string;
  build: () => FeeScheduleFormState;
}

const flat = (percentage: number, label: string, description: string): FeeSchedulePreset => ({
  key: `flat-${percentage}`,
  label,
  description,
  build: () => ({
    effectiveDate: tomorrowISO(),
    structure: "flat",
    tiers: [{ minAmount: 0, percentage }],
    minFeeVnd: DEFAULT_MIN_FEE_VND,
    notes: "",
  }),
});

export const FEE_SCHEDULE_PRESETS: FeeSchedulePreset[] = [
  flat(2, "Phí cố định 2%", "Mọi khoản ứng đều tính 2%"),
  flat(1.5, "Phí cố định 1,5%", "Mọi khoản ứng đều tính 1,5%"),
  {
    key: "tiered-2-1.3",
    label: "Phân tầng 2% / 1,3%",
    description: "≥ 3,5 triệu xuống 1,3%",
    build: () => ({
      effectiveDate: tomorrowISO(),
      structure: "tiered",
      tiers: [
        { minAmount: 0, percentage: 2 },
        { minAmount: 3_500_000, percentage: 1.3 },
      ],
      minFeeVnd: DEFAULT_MIN_FEE_VND,
      notes: "",
    }),
  },
  {
    key: "tiered-2.5-2-1.5",
    label: "Phân tầng 2,5% / 2% / 1,5%",
    description: "≥ 3 triệu, ≥ 10 triệu",
    build: () => ({
      effectiveDate: tomorrowISO(),
      structure: "tiered",
      tiers: [
        { minAmount: 0, percentage: 2.5 },
        { minAmount: 3_000_000, percentage: 2 },
        { minAmount: 10_000_000, percentage: 1.5 },
      ],
      minFeeVnd: DEFAULT_MIN_FEE_VND,
      notes: "",
    }),
  },
];
