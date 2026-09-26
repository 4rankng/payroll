import { useState, useEffect } from 'react';
import { PayratePreview } from './PayratePreview';
import { ExistingEntriesDisplay } from './ExistingEntriesDisplay';
import { useProjectPayRates, useCurrentPayRate, isForbiddenError } from '@/hooks/api/usePayRates';
import { useProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { useTimesheetsByProjectAndDate } from '@/hooks/api/useTimesheets';
import { timesheetService } from '@/services/api/timesheet.service';
import { getDayTypeOrNull, isSaturdayRequiringDayType } from '@/utils/dateHelpers';
import { DayType } from '@/types/api/payrate.types';
import { NewTimesheetEntry } from '@/types/api/timesheet.types';
import { toast } from '@/components/ui/sonner';

// Mobile components
import { MobileFormHeader } from './components/MobileFormHeader';
import { MobileEmployeeSelector } from './components/MobileEmployeeSelector';
import { MobileProjectSelector } from './components/MobileProjectSelector';
import { MobileDateTimeSelector } from './components/MobileDateTimeSelector';
import { MobileSubmitButton } from './components/MobileSubmitButton';

interface Employee {
  id: number;
  fullname: string;
  employee_code: string;
  position: string;
  avatar?: string;
}

interface Project {
  id: number;
  name: string;
  code: string;
}

interface TimesheetEntryMobileFormProps {
  onClose?: () => void;
}

export function TimesheetEntryMobileForm({ onClose }: TimesheetEntryMobileFormProps) {
  
  // Form state
  const [selectedEmployee, setSelectedEmployee] = useState<Employee | null>(null);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [position, setPosition] = useState<string>('phổ thông');
  const [dayType, setDayType] = useState<DayType | null>(null);
  const [hourType, setHourType] = useState<string>('');
  const [hoursWorked, setHoursWorked] = useState<number>(0);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Fetch data
  const { data: employeesData } = useProjectEmployees(
    selectedProject?.id || 0, 
    {}, 
    !!selectedProject
  );
  const { data: payRateData, error: payRateError } = useCurrentPayRate(
    selectedProject?.id || 0,
    !!selectedProject
  );
  // 403 = the current user cannot access this project's payrate — show a
  // permission notice instead of letting the form read as "no payrate config".
  const payRateForbidden = isForbiddenError(payRateError);
  
  const formattedDate = selectedDate.toISOString().split('T')[0];
  const { data: existingTimesheets, refetch: refetchTimesheets } = useTimesheetsByProjectAndDate(
    selectedProject?.id || 0,
    formattedDate,
    !!selectedProject
  );

  // Transform employees data
  const employees: Employee[] = (employeesData?.data || []).map(assignment => ({
    id: assignment.employee_id,
    fullname: assignment.employee_name,
    employee_code: assignment.employee_code,
    position: assignment.position
  }));

  const payrateConfig = payRateData?.rates;

  // Auto-determine day type when date changes
  useEffect(() => {
    const dateStr = selectedDate.toISOString().split('T')[0];
    
    if (isSaturdayRequiringDayType(dateStr)) {
      setDayType(null); // Requires user input
    } else {
      setDayType(getDayTypeOrNull(dateStr));
    }
  }, [selectedDate]);

  // Calculate payrate and amount
  const getPayrate = (): number => {
    if (!payrateConfig || !dayType || !hourType) return 0;
    
    const positionRates = payrateConfig[position];
    if (!positionRates) return 0;
    
    const dayTypeRates = positionRates[dayType];
    if (!dayTypeRates) return 0;
    
    return dayTypeRates[hourType] || 0;
  };

  const payrate = getPayrate();
  const calculatedAmount = payrate * hoursWorked;

  // Form validation
  const isFormValid = selectedEmployee && selectedProject && dayType && hourType && hoursWorked > 0;

  const handleSubmit = async () => {
    if (!isFormValid) {
      toast({
        title: "Lỗi",
        description: "Vui lòng điền đầy đủ thông tin",
        variant: "destructive"
      });
      return;
    }

    setIsSubmitting(true);
    try {
      const entry: NewTimesheetEntry = {
        projectId: selectedProject!.id,
        employeeId: selectedEmployee!.id,
        date: formattedDate,
        hoursWorked,
        hourType
      };

      // Add dayType for Saturdays
      if (isSaturdayRequiringDayType(formattedDate)) {
        entry.dayType = dayType === 'ngày lễ' ? 'Ngày lễ'
          : dayType === 'ngày thường' ? 'Ngày thường' : 'Ngày nghỉ';
      }

      await timesheetService.createTimesheets([entry]);
      
      toast({
        title: "Thành công",
        description: "Đã lưu bảng công thành công",
        variant: "default"
      });

      // Reset form
      setHourType('');
      setHoursWorked(0);
      
      // Refresh existing entries
      refetchTimesheets();
      
    } catch (error) {
      toast({
        title: "Lỗi",
        description: "Không thể lưu bảng công. Vui lòng thử lại.",
        variant: "destructive"
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-[100dvh] overflow-x-clip bg-card p-3 pb-[calc(6rem+env(safe-area-inset-bottom))] sm:p-4">
      <div className="max-w-md mx-auto space-y-3 sm:space-y-4">
        {/* Header */}
        <MobileFormHeader />
        {payRateForbidden && (
          <p className="text-xs text-red-600 bg-red-50 rounded-xl px-3 py-2">
            Bạn không có quyền truy cập bảng lương của dự án này.
          </p>
        )}

        {/* Employee Selection */}
        <MobileEmployeeSelector
          employees={employees}
          selectedEmployee={selectedEmployee}
          onEmployeeChange={setSelectedEmployee}
          onPositionChange={setPosition}
        />

        {/* Project Selection */}
        <MobileProjectSelector
          selectedProject={selectedProject}
          onProjectChange={setSelectedProject}
        />

        {/* Date, Position, Hour Type, and Hours */}
        <MobileDateTimeSelector
          selectedDate={selectedDate}
          onDateChange={setSelectedDate}
          dayType={dayType}
          onDayTypeChange={setDayType}
          position={position}
          onPositionChange={setPosition}
          hourType={hourType}
          onHourTypeChange={setHourType}
          hoursWorked={hoursWorked}
          onHoursChange={setHoursWorked}
          payrateConfig={payrateConfig}
        />

        {/* Payrate Preview */}
        {hourType && hoursWorked > 0 && (
          <PayratePreview
            payrate={payrate}
            hours={hoursWorked}
            amount={calculatedAmount}
          />
        )}

        {/* Existing Entries */}
        {selectedProject && (
          <ExistingEntriesDisplay
            projectId={selectedProject.id}
            date={formattedDate}
            existingTimesheets={existingTimesheets?.data || []}
          />
        )}

        {/* Submit Button */}
        <MobileSubmitButton
          isFormValid={!!isFormValid}
          isSubmitting={isSubmitting}
          onSubmit={handleSubmit}
        />
      </div>
    </div>
  );
}
