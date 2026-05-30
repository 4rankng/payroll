import type { CreatePayRateRequest, PayRate } from '@/types/api/payrate.types';

// Test Vietnamese payrate structure matching your API
export const createVietnamesePayrateTest = (): CreatePayRateRequest => {
  return {
    rates: {
      "tất cả": {
        "ngày thường": {
          "bình thường": 30000,
          "tăng ca": 45000,
          "ca đêm": 39000
        },
        "cuối tuần": {
          "bình thường": 60000,
          "tăng ca": 90000,
          "ca đêm": 78000
        },
        "ngày lễ": {
          "bình thường": 90000,
          "tăng ca": 135000,
          "ca đêm": 117000
        }
      }
    },
    effective_from: "2025-01-01"
    // effective_to is optional - omit for open-ended rates
  };
};

// Test experienced/unskilled structure from API docs
export const createSkillBasedPayrateTest = (): CreatePayRateRequest => {
  return {
    rates: {
      "experienced": {
        "weekday": {
          "08:00-17:00": 32500,
          "17:00-20:00": 48750,
          "20:00-22:00": 32500
        },
        "weekend": {
          "08:00-17:00": 65000,
          "17:00-20:00": 65000
        },
        "holiday": {
          "08:00-17:00": 97500,
          "17:00-20:00": 97500
        }
      },
      "unskilled": {
        "weekday": {
          "normal": 25000,
          "overtime": 30000
        },
        "weekend": {
          "normal": 40000
        }
      }
    },
    effective_from: "2025-01-01",
    effective_to: "2025-12-31" // Optional - can specify end date
  };
};

// Test API response parsing
export const testApiResponseParsing = (apiResponse: unknown): PayRate => {
  // This should match your actual API response structure
  const expectedResponse = {
    "success": true, // API docs show boolean success field
    "message": "Payrate created successfully",
    "data": {
      "id": 7,
      "project_id": 2,
      "rates": {
        "tất cả": {
          "ngày thường": {
            "bình thường": 30000,
            "tăng ca": 45000,
            "ca đêm": 39000
          },
          "cuối tuần": {
            "bình thường": 60000,
            "tăng ca": 90000,
            "ca đêm": 78000
          },
          "ngày lễ": {
            "bình thường": 90000,
            "tăng ca": 135000,
            "ca đêm": 117000
          }
        }
      },
      "from_date": "2025-01-01",
      "to_date": null, // Open-ended rate
      "status": "active",
      "last_updated_by": 1,
      "last_approved_by": null,
      "approved_at": null,
      "created_at": "2025-01-16T10:30:00Z", // ISO format per docs
      "updated_at": "2025-01-16T10:30:00Z"
    }
  };

  return expectedResponse.data;
};

// Helper function to extract rates for specific time periods
export const extractRatesForPeriod = (
  payRate: PayRate,
  skillLevel: string = "tất cả",
  period: "ngày thường" | "cuối tuần" | "ngày lễ"
): Record<string, number> | null => {
  const skillRates = payRate.rates[skillLevel];
  if (!skillRates || typeof skillRates !== 'object') return null;

  const periodRates = skillRates[period];
  if (!periodRates || typeof periodRates !== 'object') return null;

  // Extract numeric values only
  const result: Record<string, number> = {};
  Object.entries(periodRates).forEach(([key, value]) => {
    if (typeof value === 'number') {
      result[key] = value;
    }
  });

  return result;
};

// Usage examples:
// const request = createVietnamesePayrateTest();
// const normalDayRates = extractRatesForPeriod(payRate, "tất cả", "ngày thường");
//  // { "bình thường": 30000, "tăng ca": 45000, "ca đêm": 39000 }
