import { useState, useMemo, useRef, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp, TrendingDown, DollarSign } from 'lucide-react';
import { useFinancialChart } from '@/hooks/admin-dashboard/useFinancialChart';
import { formatCurrency } from '@/utils/formatters';
import type { FinancialChartParams } from '@/types/api/dashboard.types';

const PERIOD_OPTIONS = [
  { value: 'day', label: 'Ngày', description: '180 ngày gần đây' },
  { value: 'month', label: 'Tháng', description: '2 năm gần đây' },
] as const;

// All possible line names used by the chart. Defined as a single union so
// downstream switches can be exhaustive and don't get narrowed to a single
// group's first-element literal type by the `as const` tuple below.
type LineName =
  | 'Doanh thu'
  | 'Chi phí'
  | 'Lợi nhuận'
  | 'Tiền mặt'
  | 'Công nợ phải thu'
  | 'Công nợ phải trả'
  | 'NV làm việc';

const CHART_GROUPS: ReadonlyArray<{
  value: string;
  label: string;
  description: string;
  lines: ReadonlyArray<LineName>;
  color: string;
}> = [
  {
    value: 'revenue',
    label: 'Lợi nhuận',
    description: 'Doanh thu, Chi phí, Lợi nhuận',
    lines: ['Doanh thu', 'Chi phí', 'Lợi nhuận'],
    color: 'hsl(142 76% 36%)'
  },
  {
    value: 'cashflow',
    label: 'Dòng tiền',
    description: 'Tiền mặt, Công nợ',
    lines: ['Tiền mặt', 'Công nợ phải thu', 'Công nợ phải trả'],
    color: 'hsl(217 91% 60%)'
  },
  {
    value: 'operations',
    label: 'Hoạt động',
    description: 'NV làm việc, Doanh thu, Phải thu',
    lines: ['NV làm việc', 'Doanh thu', 'Công nợ phải thu'],
    color: 'hsl(174 72% 31%)'
  },
];

interface FinancialChartProps {
  className?: string;
}

export function FinancialChart({ className }: FinancialChartProps) {
  const [period, setPeriod] = useState<FinancialChartParams['period']>('day');
  const [isVisible, setIsVisible] = useState(false);
  const cardRef = useRef<HTMLDivElement>(null);
  const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set(['Tiền mặt', 'Công nợ phải thu', 'Công nợ phải trả', 'NV làm việc']));
  const [activeGroup, setActiveGroup] = useState<string>('revenue');

  // Intersection Observer for lazy loading
  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: '100px' } // Start loading 100px before visible
    );

    if (cardRef.current) {
      observer.observe(cardRef.current);
    }

    return () => observer.disconnect();
  }, []);

  const { data, isLoading, isError } = useFinancialChart({ period }, isVisible);

  // Format data for chart
  const chartData = useMemo(() => {
    if (!data) return [];
    return data.map(item => ({
      date: item.date,
      'Doanh thu': item.revenue_vnd,
      'Chi phí': item.expenses_vnd,
      'Lợi nhuận': item.profit_vnd,
      'Tiền mặt': item.cash_vnd,
      'Công nợ phải thu': item.receivable_vnd,
      'Công nợ phải trả': item.payable_vnd,
      'NV làm việc': item.active_employees,
    }));
  }, [data]);

  // Calculate group-specific stats (latest data point)
  const groupStats = useMemo(() => {
    if (!data || data.length === 0) {
      return [];
    }

    const latestPeriod = data[data.length - 1];
    const activeGroupData = CHART_GROUPS.find(g => g.value === activeGroup);

    if (!activeGroupData) return [];

    return activeGroupData.lines.map((rawLine) => {
      const lineName = rawLine as LineName;
      let value = 0;
      let color = '';

      switch (lineName) {
        case 'Doanh thu':
          value = latestPeriod.revenue_vnd;
          color = 'text-green-700';
          break;
        case 'Chi phí':
          value = latestPeriod.expenses_vnd;
          color = 'text-red-600';
          break;
        case 'Lợi nhuận':
          value = latestPeriod.profit_vnd;
          color = value >= 0 ? 'text-green-700' : 'text-red-600';
          break;
        case 'Tiền mặt':
          value = latestPeriod.cash_vnd;
          color = 'text-blue-600';
          break;
        case 'Công nợ phải thu':
          value = latestPeriod.receivable_vnd;
          color = 'text-cyan-700';
          break;
        case 'Công nợ phải trả':
          value = latestPeriod.payable_vnd;
          color = 'text-orange-700';
          break;
        case 'NV làm việc':
          value = latestPeriod.active_employees;
          color = 'text-emerald-700';
          break;
        default:
          value = 0;
          color = 'text-muted-foreground';
      }

      return {
        name: lineName,
        value,
        color,
        isEmployee: lineName === 'NV làm việc'
      };
    });
  }, [data, activeGroup]);

  // Calculate trend based on first line in active group
  const trend = useMemo(() => {
    if (!data || data.length < 2) return 'stable' as const;

    const activeGroupData = CHART_GROUPS.find(g => g.value === activeGroup);
    const firstLineName = activeGroupData?.lines[0] as LineName | undefined;
    if (!activeGroupData || !firstLineName) return 'stable';
    const recent = data[data.length - 1];
    const previous = data[data.length - 2];

    let recentValue = 0;
    let previousValue = 0;

    switch (firstLineName) {
      case 'Doanh thu':
        recentValue = recent.revenue_vnd;
        previousValue = previous.revenue_vnd;
        break;
      case 'Chi phí':
        recentValue = recent.expenses_vnd;
        previousValue = previous.expenses_vnd;
        break;
      case 'Lợi nhuận':
        recentValue = recent.profit_vnd;
        previousValue = previous.profit_vnd;
        break;
      case 'Tiền mặt':
        recentValue = recent.cash_vnd;
        previousValue = previous.cash_vnd;
        break;
      case 'Công nợ phải thu':
        recentValue = recent.receivable_vnd;
        previousValue = previous.receivable_vnd;
        break;
      case 'Công nợ phải trả':
        recentValue = recent.payable_vnd;
        previousValue = previous.payable_vnd;
        break;
      case 'NV làm việc':
        recentValue = recent.active_employees;
        previousValue = previous.active_employees;
        break;
    }

    return recentValue > previousValue ? 'up' : recentValue < previousValue ? 'down' : 'stable';
  }, [data, activeGroup]);

  const selectedPeriod = PERIOD_OPTIONS.find(p => p.value === period);

  const handleGroupToggle = (groupValue: string) => {
    if (activeGroup === groupValue) return; // Don't change if clicking the same group

    const newGroup = CHART_GROUPS.find(g => g.value === groupValue);
    if (!newGroup) return;

    // Hide all lines from the previous active group
    const previousGroup = CHART_GROUPS.find(g => g.value === activeGroup);
    if (previousGroup) {
      setHiddenSeries(prevHidden => {
        const newHidden = new Set(prevHidden);
        previousGroup.lines.forEach(line => newHidden.add(line));
        return newHidden;
      });
    }

    // Show all lines from the new active group
    setHiddenSeries(prevHidden => {
      const newHidden = new Set(prevHidden);
      newGroup.lines.forEach(line => newHidden.delete(line));
      return newHidden;
    });

    setActiveGroup(groupValue);
  };

  const handleLegendClick = (data: { dataKey?: string | number }) => {
    const dataKey = typeof data.dataKey === 'string' ? data.dataKey : String(data.dataKey);
    if (!dataKey || dataKey === 'undefined') return;
    setHiddenSeries(prev => {
      const newSet = new Set(prev);
      if (newSet.has(dataKey)) {
        newSet.delete(dataKey);
      } else {
        newSet.add(dataKey);
      }
      return newSet;
    });
  };

  if (!isVisible || isLoading) {
    return (
      <Card ref={cardRef} className={className}>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-64 mt-2" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-10 w-full mb-4" />
          <Skeleton className="h-80 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (isError) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base">Biểu Đồ Tài Chính</CardTitle>
          <CardDescription>Không thể tải dữ liệu</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-80 flex items-center justify-center text-muted-foreground">
            Không có dữ liệu
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card ref={cardRef} className={className}>
      <CardHeader className="pb-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-base flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <DollarSign className="h-5 w-5" />
                Biểu Đồ Tài Chính
                {trend === 'up' && <TrendingUp className="h-4 w-4 text-green-700" />}
                {trend === 'down' && <TrendingDown className="h-4 w-4 text-red-600" />}
              </div>

              {/* Period Toggle Group */}
              <div className="flex gap-0.5 p-0.5 bg-muted rounded-xl">
                {PERIOD_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => setPeriod(option.value as FinancialChartParams['period'])}
                    className={`px-2 py-0.5 text-xs font-medium rounded-sm transition-colors ${
                      period === option.value
                        ? 'bg-background text-foreground shadow-sm'
                        : 'text-muted-foreground hover:text-foreground'
                    }`}
                    title={option.description}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </CardTitle>
            <CardDescription className="mt-1.5">
              {selectedPeriod?.description}
            </CardDescription>
          </div>

          {/* Group Selector */}
          <div className="flex gap-0.5 p-0.5 bg-muted rounded-xl">
            {CHART_GROUPS.map((group) => (
              <button
                key={group.value}
                onClick={() => handleGroupToggle(group.value)}
                className={`px-1.5 py-0.5 text-xs font-medium rounded-sm transition-colors ${
                  activeGroup === group.value
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
                title={group.description}
              >
                {group.label}
              </button>
            ))}
          </div>
        </div>

        {/* Group Stats */}
        <div className="grid grid-cols-3 gap-3 mt-4 text-xs">
          {groupStats.slice(0, 3).map((stat, index) => (
            <div key={stat.name}>
              <div className="text-muted-foreground mb-1">{stat.name}</div>
              <div className={`font-semibold text-sm ${stat.color}`}>
                {stat.isEmployee ? stat.value.toLocaleString('vi-VN') : formatCurrency(stat.value)}
              </div>
            </div>
          ))}
        </div>
      </CardHeader>

      <CardContent>
        {/* Chart */}
        <div className="w-full -mx-2 sm:mx-0">
          <ResponsiveContainer width="100%" height={350} minWidth={0}>
            <LineChart
              data={chartData}
              margin={{ top: 5, right: 10, left: 0, bottom: 5 }}
            >
              <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
              <XAxis
                dataKey="date"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                interval="preserveStartEnd"
                minTickGap={30}
              />
              <YAxis
                yAxisId="left"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                width={80}
                domain={[0, 'dataMax']}
                tickFormatter={(value) => {
                  // Vietnamese number format with thousands separator
                  return new Intl.NumberFormat('vi-VN').format(value);
                }}
              />
              <YAxis
                yAxisId="right"
                orientation="right"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                width={40}
                tickFormatter={(value) => {
                  // Format employee count as whole number
                  return Math.round(value).toString();
                }}
              />
              <Tooltip
                formatter={(value: number, name: string) => {
                  // Format employee count differently from currency
                  if (name === 'NV làm việc') {
                    return [Math.round(value).toLocaleString('vi-VN'), name];
                  }
                  return [formatCurrency(value), name];
                }}
                labelFormatter={(label) => label}
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '8px',
                  fontSize: '11px',
                  padding: '8px',
                }}
              />
              <Legend
                wrapperStyle={{ fontSize: '11px', paddingTop: '12px', cursor: 'pointer' }}
                iconType="line"
                iconSize={14}
                onClick={(data: { value?: string | number }) => handleLegendClick({ dataKey: data.value })}
              />
              {/* Group 1: Doanh thu & Chi phí (Left Y-axis) */}
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Doanh thu"
                stroke="hsl(142 76% 36%)"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Doanh thu')}
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Chi phí"
                stroke="hsl(0 84% 60%)"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Chi phí')}
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Lợi nhuận"
                stroke="hsl(45 93% 47%)"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Lợi nhuận')}
              />

              {/* Group 2: Dòng tiền (Left Y-axis) */}
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Tiền mặt"
                stroke="hsl(217 91% 60%)"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Tiền mặt')}
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Công nợ phải thu"
                stroke="hsl(200 70% 50%)"
                strokeWidth={2}
                strokeDasharray="5 5"
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Công nợ phải thu')}
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="Công nợ phải trả"
                stroke="hsl(25 95% 53%)"
                strokeWidth={2}
                strokeDasharray="5 5"
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('Công nợ phải trả')}
              />

              {/* Group 3: Hoạt động (Right Y-axis for NV làm việc, others already on left) */}
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="NV làm việc"
                stroke="hsl(174 72% 31%)"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                hide={hiddenSeries.has('NV làm việc')}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
