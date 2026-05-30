import { Card, CardContent } from "@/components/ui/card";
import { Calendar, Clock, Users, TrendingUp } from "lucide-react";

interface TimesheetRecord {
  id: number;
  employeeName: string;
  workDays: number;
  actualDays: number;
  overtime: number;
  late: number;
  absent: number;
  totalHours: number;
}

interface PartnerTimesheetStatsProps {
  timesheetData: TimesheetRecord[];
}

export const PartnerTimesheetStats = ({ timesheetData }: PartnerTimesheetStatsProps) => {
  const totalActualDays = timesheetData.reduce((sum, item) => sum + item.actualDays, 0);
  const totalOvertime = timesheetData.reduce((sum, item) => sum + item.overtime, 0);

  const stats = [
    {
      title: "Tổng ngày làm",
      value: totalActualDays,
      icon: Calendar,
      gradient: "from-blue-50 to-indigo-100",
      iconBg: "from-blue-500 to-indigo-600",
      textColor: "text-foreground"
    },
    {
      title: "Tổng OT",
      value: `${totalOvertime}h`,
      icon: Clock,
      gradient: "from-orange-50 to-red-100",
      iconBg: "from-orange-500 to-red-600",
      textColor: "text-foreground"
    }
  ];

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 lg:gap-6">
      {stats.map((stat, index) => (
        <Card key={index} className={`bg-gradient-to-br ${stat.gradient} border-0 shadow-sm hover:shadow-xl transition-all duration-300`}>
          <CardContent className="p-4 sm:p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="typography-body-medium typography-label-medium text-muted-foreground">{stat.title}</p>
                <p className={`typography-display-small ${stat.textColor}`}>{stat.value}</p>
              </div>
              <div className={`p-3 bg-gradient-to-br ${stat.iconBg} rounded-xl shadow-sm`}>
                <stat.icon className="w-6 h-6 text-white" />
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
};