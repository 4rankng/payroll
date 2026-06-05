import { memo, useCallback } from "react";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { PayRate } from "@/types/api/payrate.types";
import type {
  TimesheetFormData,
  TimesheetFormOptions,
} from "@/types/timesheet";
import { requiresDayType } from "@/utils/timesheetHelpers";

interface DateTimeSectionProps {
  formData: TimesheetFormData;
  formOptions: TimesheetFormOptions;
  saturdayDayType: "Ngày thường" | "Ngày nghỉ" | "Ngày lễ";
  isLoadingPayRate: boolean;
  payRateData?: PayRate;
  onInputChange: (
    field: keyof TimesheetFormData,
    value: string | number,
  ) => void;
  onSaturdayDayTypeChange: (
    value: "Ngày thường" | "Ngày nghỉ" | "Ngày lễ",
  ) => void;
}

export const DateTimeSection = memo(
  ({
    formData,
    formOptions,
    saturdayDayType,
    isLoadingPayRate,
    payRateData,
    onInputChange,
    onSaturdayDayTypeChange,
  }: DateTimeSectionProps) => {
    const handleDateChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        onInputChange("date", e.target.value);
      },
      [onInputChange],
    );

    const handleHourTypeChange = useCallback(
      (value: string) => {
        onInputChange("hour_type", value);
      },
      [onInputChange],
    );

    const handleHoursChange = useCallback(
      (e: React.ChangeEvent<HTMLInputElement>) => {
        onInputChange("hours_worked", parseFloat(e.target.value) || 0);
      },
      [onInputChange],
    );

    return (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <Label htmlFor="date">Ngày làm việc *</Label>
          <Input
            id="date"
            type="date"
            value={formData.date}
            onChange={handleDateChange}
          />

          {requiresDayType(formData.date) && (
            <div className="mt-2">
              <Label className="typography-body-medium">
                Phân loại thứ 7 *
              </Label>
              <Select
                value={saturdayDayType}
                onValueChange={onSaturdayDayTypeChange}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Ngày thường">Ngày thường</SelectItem>
                  <SelectItem value="Ngày nghỉ">Ngày nghỉ</SelectItem>
                  <SelectItem value="Ngày lễ">Ngày lễ</SelectItem>
                </SelectContent>
              </Select>
              <div className="mt-1 typography-body-small text-muted-foreground">
                Chọn cách tính lương cho ngày thứ 7
              </div>
            </div>
          )}
        </div>

        <div>
          <Label htmlFor="hourType">Khung giờ *</Label>
          {isLoadingPayRate ? (
            <div className="flex items-center justify-center p-2 border rounded-xl">
              <div className="animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent mr-2" />
              Đang tải...
            </div>
          ) : !payRateData?.rates ? (
            <div className="p-2 border rounded-xl text-muted-foreground text-sm">
              Chưa có cấu hình lương
            </div>
          ) : (
            <Select
              value={formData.hour_type}
              onValueChange={handleHourTypeChange}
            >
              <SelectTrigger>
                <SelectValue placeholder="Chọn khung giờ" />
              </SelectTrigger>
              <SelectContent>
                {formOptions.hourTypes.map((hourType) => (
                  <SelectItem key={hourType} value={hourType}>
                    {hourType}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        <div>
          <Label htmlFor="hours">Số giờ làm việc *</Label>
          <Input
            id="hours"
            type="number"
            min="0"
            max="24"
            step="0.5"
            value={formData.hours_worked}
            onChange={handleHoursChange}
          />
        </div>
      </div>
    );
  },
);

DateTimeSection.displayName = "DateTimeSection";
