import { format } from "date-fns";
import { useMemo } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar } from 'recharts';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp, TrendingDown } from 'lucide-react';
import { ledgerService } from '@/services/api/ledger.service';
import { dateToString } from '@/utils/dateHelpers';
import { useCashFlowSummary } from '@/hooks/ledger/useLedgerBalance';

interface CashFlowChartProps {
  className?: string;
}

export function CashFlowChart({ className }: CashFlowChartProps) {
  // Get last 6 months data
  const monthsData = useMemo(() => {
    const months = [];
    const now = new Date();

    for (let i = 5; i >= 0; i--) {
      const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
      const fromDate = dateToString(new Date(date.getFullYear(), date.getMonth(), 1));
      const toDate = dateToString(new Date(date.getFullYear(), date.getMonth() + 1, 0));

      months.push({
        month: format(date, 'dd/MM/yyyy'),
        fromDate,
        toDate,
      });
    }
    return months;
  }, []);

  // Fetch cash flow data for each month (using fixed calls since we know there are 6 months)
  const month0Query = useCashFlowSummary(monthsData[0]?.fromDate || '', monthsData[0]?.toDate || '');
  const month1Query = useCashFlowSummary(monthsData[1]?.fromDate || '', monthsData[1]?.toDate || '');
  const month2Query = useCashFlowSummary(monthsData[2]?.fromDate || '', monthsData[2]?.toDate || '');
  const month3Query = useCashFlowSummary(monthsData[3]?.fromDate || '', monthsData[3]?.toDate || '');
  const month4Query = useCashFlowSummary(monthsData[4]?.fromDate || '', monthsData[4]?.toDate || '');
  const month5Query = useCashFlowSummary(monthsData[5]?.fromDate || '', monthsData[5]?.toDate || '');

  const cashFlowQueries = useMemo(() => [month0Query, month1Query, month2Query, month3Query, month4Query, month5Query], [month0Query, month1Query, month2Query, month3Query, month4Query, month5Query]);

  const isLoading = cashFlowQueries.some(query => query.isLoading);
  const hasError = cashFlowQueries.some(query => query.isError);

  // Process data for chart
  const chartData = useMemo(() => {
    return monthsData.map((month, index) => {
      const data = cashFlowQueries[index]?.data;
      return {
        month: month.month,
        inflow: data?.total_inflow || 0,
        outflow: data?.total_outflow || 0,
        netFlow: data?.net_cash_flow || 0,
      };
    });
  }, [monthsData, cashFlowQueries]);

  // Calculate trend
  const trend = useMemo(() => {
    if (chartData.length < 2) return 'stable';
    const recent = chartData[chartData.length - 1]?.netFlow || 0;
    const previous = chartData[chartData.length - 2]?.netFlow || 0;
    return recent > previous ? 'up' : recent < previous ? 'down' : 'stable';
  }, [chartData]);

  const totalInflow = chartData.reduce((sum, item) => sum + item.inflow, 0);
  const totalOutflow = chartData.reduce((sum, item) => sum + item.outflow, 0);
  const totalNetFlow = totalInflow - totalOutflow;

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-48" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (hasError) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle className="text-base">Biểu Đồ Dòng Tiền</CardTitle>
          <CardDescription>Không thể tải dữ liệu</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="h-64 flex items-center justify-center text-muted-foreground">
            Không có dữ liệu
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base flex items-center gap-2">
              Dòng Tiền 6 Tháng Gần Đây
              {trend === 'up' && <TrendingUp className="h-4 w-4 text-green-500" />}
              {trend === 'down' && <TrendingDown className="h-4 w-4 text-red-500" />}
            </CardTitle>
            <CardDescription>
              Tổng ròng: {totalNetFlow >= 0 ? '+' : ''}{ledgerService.formatCurrency(totalNetFlow)}
            </CardDescription>
          </div>
          <div className="text-right text-xs text-muted-foreground">
            <div className="text-green-600">
              ↗ Vào: {ledgerService.formatCurrency(totalInflow)}
            </div>
            <div className="text-red-600">
              ↘ Ra: {ledgerService.formatCurrency(totalOutflow)}
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Net Cash Flow Line Chart */}
          <div>
            <h4 className="text-sm font-medium mb-2 text-muted-foreground">Dòng Tiền Ròng</h4>
            <ResponsiveContainer width="100%" height={200} minWidth={0}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
                <XAxis
                  dataKey="month"
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => ledgerService.formatCurrencyShort(value)}
                />
                <Tooltip
                  formatter={(value: number) => [ledgerService.formatCurrency(value), 'Dòng tiền ròng']}
                  labelFormatter={(label) => `Tháng: ${label}`}
                  contentStyle={{
                    backgroundColor: 'hsl(var(--background))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    fontSize: '12px',
                  }}
                />
                <Line
                  type="monotone"
                  dataKey="netFlow"
                  stroke="hsl(var(--primary))"
                  strokeWidth={2}
                  dot={{ fill: 'hsl(var(--primary))', strokeWidth: 0, r: 3 }}
                  activeDot={{ r: 4, stroke: 'hsl(var(--primary))', strokeWidth: 2 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Inflow vs Outflow Bar Chart */}
          <div>
            <h4 className="text-sm font-medium mb-2 text-muted-foreground">Thu Chi So Sánh</h4>
            <ResponsiveContainer width="100%" height={200} minWidth={0}>
              <BarChart data={chartData} barGap={10}>
                <CartesianGrid strokeDasharray="3 3" className="opacity-30" />
                <XAxis
                  dataKey="month"
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => ledgerService.formatCurrencyShort(value)}
                />
                <Tooltip
                  formatter={(value: number, name) => [
                    ledgerService.formatCurrency(value),
                    name === 'inflow' ? 'Thu vào' : 'Chi ra'
                  ]}
                  labelFormatter={(label) => `Tháng: ${label}`}
                  contentStyle={{
                    backgroundColor: 'hsl(var(--background))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    fontSize: '12px',
                  }}
                />
                <Bar
                  dataKey="inflow"
                  fill="hsl(142 76% 36%)"
                  radius={[2, 2, 0, 0]}
                  name="inflow"
                />
                <Bar
                  dataKey="outflow"
                  fill="hsl(0 84% 60%)"
                  radius={[2, 2, 0, 0]}
                  name="outflow"
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
