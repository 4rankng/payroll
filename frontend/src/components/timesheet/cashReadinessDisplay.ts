import type {
  CashReadinessReliabilityState,
  CashReadinessResponse,
} from '@/types/api/cash-readiness.types';

export interface CashReadinessDriver {
  label: string;
  amount: number;
  description?: string;
}

export interface CashReadinessAccuracyMetric {
  label: string;
  value: string;
}

export interface CashReadinessDisplay {
  expectedPayout: number;
  recommendedReserve: number;
  intervalLower: number;
  intervalUpper: number;
  drivers: CashReadinessDriver[];
  reliabilityState: CashReadinessReliabilityState;
  reliabilityLabel: string;
  reliabilityDescription: string;
  accuracyMetrics: CashReadinessAccuracyMetric[];
  dataWarning?: string;
}

const nonNegative = (value: number): number => Math.max(value, 0);

const formatRate = (value: number): string => {
  const percent = Math.abs(value) <= 1 ? value * 100 : value;
  return `${new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 1 }).format(percent)}%`;
};

const getReliabilityState = (
  state: CashReadinessResponse['reliability_state'],
): CashReadinessReliabilityState => {
  if (state === 'learning' || state === 'measured') return state;
  return 'uncalibrated';
};

const getReliabilityCopy = (
  state: CashReadinessReliabilityState,
  calibrationSamples: number,
): Pick<CashReadinessDisplay, 'reliabilityLabel' | 'reliabilityDescription'> => {
  if (state === 'measured') {
    return {
      reliabilityLabel: 'Đã đo bằng kết quả thực tế',
      reliabilityDescription: `${calibrationSamples} dự báo cùng thời hạn đã được đối chiếu.`,
    };
  }

  if (state === 'learning') {
    return {
      reliabilityLabel: 'Đang học từ kết quả thực tế',
      reliabilityDescription: `${calibrationSamples} dự báo đã được đối chiếu; chưa đủ để kết luận độ chính xác.`,
    };
  }

  return {
    reliabilityLabel: 'Chưa hiệu chỉnh',
    reliabilityDescription:
      calibrationSamples > 0
        ? `${calibrationSamples} dự báo đã được đối chiếu; chưa đủ để hiệu chỉnh độ chính xác.`
        : 'Chưa có kết quả thực tế để đo độ chính xác.',
  };
};

const getAccuracyMetrics = (data: CashReadinessResponse): CashReadinessAccuracyMetric[] => {
  const metrics: CashReadinessAccuracyMetric[] = [];

  if (data.accuracy_wape !== undefined) {
    metrics.push({ label: 'Sai số WAPE', value: formatRate(data.accuracy_wape) });
  }
  if (data.accuracy_bias !== undefined) {
    metrics.push({ label: 'Độ lệch', value: formatRate(data.accuracy_bias) });
  }
  if (data.interval_coverage !== undefined) {
    metrics.push({ label: 'Bao phủ khoảng', value: formatRate(data.interval_coverage) });
  }
  if (data.reserve_shortfall_rate !== undefined) {
    metrics.push({ label: 'Thiếu dự trữ', value: formatRate(data.reserve_shortfall_rate) });
  }

  return metrics;
};

export const getCashReadinessDisplay = (data: CashReadinessResponse): CashReadinessDisplay => {
  const expectedPayout = nonNegative(data.expected_payout ?? data.expected_total);
  const recommendedReserve = Math.max(
    nonNegative(data.recommended_reserve ?? data.cash_to_prepare),
    expectedPayout,
  );
  const intervalLower = nonNegative(data.interval_lower ?? data.band_lower);
  const intervalUpper = nonNegative(data.interval_upper ?? data.band_upper);
  const reliabilityState = getReliabilityState(data.reliability_state);
  const calibrationSamples = Math.max(data.calibration_samples ?? 0, 0);

  const drivers: CashReadinessDriver[] = [
    { label: 'Đã duyệt', amount: nonNegative(data.observed_approved), description: 'chắc chắn' },
  ];

  if (data.pending_target_amount !== undefined) {
    drivers.push({
      label: 'Đang chờ duyệt',
      amount: nonNegative(data.pending_target_amount),
      description:
        data.expected_pending_amount === undefined
          ? undefined
          : `dự kiến vào chi trả ${new Intl.NumberFormat('vi-VN', {
              style: 'currency',
              currency: 'VND',
              maximumFractionDigits: 0,
            }).format(nonNegative(data.expected_pending_amount))}`,
    });
  }

  if (data.expected_future_amount !== undefined) {
    drivers.push({ label: 'Chưa phát sinh', amount: nonNegative(data.expected_future_amount), description: 'ước tính' });
  } else if (data.pending_target_amount === undefined) {
    drivers.push({ label: 'Phần còn lại', amount: nonNegative(data.projected_expected), description: 'ước tính' });
  }

  return {
    expectedPayout,
    recommendedReserve,
    intervalLower,
    intervalUpper,
    drivers,
    reliabilityState,
    ...getReliabilityCopy(reliabilityState, calibrationSamples),
    accuracyMetrics: reliabilityState === 'measured' ? getAccuracyMetrics(data) : [],
    dataWarning:
      data.method === 'no-history'
        ? 'Chưa đủ lịch sử để ước tính phần chưa phát sinh; số tiền hiện tại chủ yếu dựa trên hàng chờ hiện có.'
        : data.method === 'completed-cycle-bootstrap'
          ? 'Kỳ này chưa có dữ liệu nhập; ước tính dựa trên tổng chi trả của các kỳ đã hoàn tất và chưa được hiệu chỉnh.'
          : undefined,
  };
};
