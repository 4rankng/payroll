import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '@/services/api/dashboard.service';
import type {
  HistoricalResponse,
  HistoricalResponseData,
  HistoricalData
} from '@/types/api/dashboard.types';
import { useMemo } from 'react';

// Types for chart data
interface ChartDataPoint {
  date: string;
  fullDate: string;
  historical?: number;
}

interface RevenueChartDataPoint {
  date: string;
  fullDate: string;
  historicalRevenue?: number;
  historicalProfit?: number;
}

// Query key for historical data
export const historicalQueryKey = ['dashboard', 'historical'];

// Hook to fetch historical data
export const useDashboardHistorical = () => {
  const {
    data: rawData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: historicalQueryKey,
    queryFn: async () => {
      const response = await dashboardService.getHistorical();
      return response;
    },
  });

  // Process data for charts
  const processedData = useMemo(() => {
    if (!rawData || !rawData.data || !rawData.data.historical) return null;

    const { historical, date } = rawData.data;

    // Helper function to format date
    const formatDate = (dateStr: string): string => {
      const date = new Date(dateStr);
      return `${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
    };

    // Prepare weekly pay data
    const prepareWeeklyPayData = (): ChartDataPoint[] => {
      return historical.weekly_pay.map((d) => ({
        date: formatDate(d.date),
        fullDate: d.date,
        historical: d.paid_amount,
      }));
    };

    // Prepare employees data (based on paid_employees from weekly pay)
    const prepareEmployeesData = (): ChartDataPoint[] => {
      return historical.weekly_pay.map((d) => ({
        date: formatDate(d.date),
        fullDate: d.date,
        historical: d.paid_employees,
      }));
    };

    // Prepare capital data
    const prepareCapitalData = (): ChartDataPoint[] => {
      return historical.capital.map((d) => ({
        date: formatDate(d.date),
        fullDate: d.date,
        historical: d.amount,
      }));
    };

    // Prepare revenue data
    const prepareRevenueData = (): RevenueChartDataPoint[] => {
      return historical.revenue.map((d) => ({
        date: formatDate(d.date),
        fullDate: d.date,
        historicalRevenue: d.total_revenue,
        historicalProfit: d.profit,
      }));
    };

    // Calculate current KPI values from historical data only
    const lastWeeklyPay = historical.weekly_pay[historical.weekly_pay.length - 1];
    const previousWeeklyPay = historical.weekly_pay[historical.weekly_pay.length - 2];

    const currentEmployees = lastWeeklyPay?.paid_employees || 0;
    const previousEmployees = previousWeeklyPay?.paid_employees || 0;
    const employeeGrowth = previousEmployees > 0 ? ((currentEmployees - previousEmployees) / previousEmployees) * 100 : 0;

    const currentWeeklyPayAmount = lastWeeklyPay?.paid_amount || 0;
    const previousWeeklyPayAmount = previousWeeklyPay?.paid_amount || 0;
    const weeklyPayGrowth = previousWeeklyPayAmount > 0 ? ((currentWeeklyPayAmount - previousWeeklyPayAmount) / previousWeeklyPayAmount) * 100 : 0;

    const lastRevenue = historical.revenue[historical.revenue.length - 1];
    const previousRevenue = historical.revenue[historical.revenue.length - 2];

    const currentRevenue = lastRevenue?.total_revenue || 0;
    const previousRevenueAmt = previousRevenue?.total_revenue || 0;
    const revenueGrowth = previousRevenueAmt > 0 ? ((currentRevenue - previousRevenueAmt) / previousRevenueAmt) * 100 : 0;

    const currentProfit = lastRevenue?.profit || 0;
    const currentProfitMargin = currentRevenue > 0 ? (currentProfit / currentRevenue) * 100 : 0;
    const previousProfit = previousRevenue?.profit || 0;
    const previousProfitMargin = previousRevenueAmt > 0 ? (previousProfit / previousRevenueAmt) * 100 : 0;
    const profitMarginGrowth = previousProfitMargin !== 0 ? ((currentProfitMargin - previousProfitMargin) / Math.abs(previousProfitMargin)) * 100 : 0;

    return {
      employeesData: prepareEmployeesData(),
      weeklyPayData: prepareWeeklyPayData(),
      capitalData: prepareCapitalData(),
      revenueData: prepareRevenueData(),
      kpiData: {
        employees: {
          value: currentEmployees,
          growth: employeeGrowth,
        },
        weeklyPay: {
          value: currentWeeklyPayAmount,
          growth: weeklyPayGrowth,
        },
        revenue: {
          value: currentRevenue,
          growth: revenueGrowth,
        },
        profitMargin: {
          value: currentProfitMargin,
          growth: profitMarginGrowth,
        },
      },
      asOf: date,
    };
  }, [rawData]);

  return {
    data: processedData,
    isLoading,
    error,
    refetch,
  };
};