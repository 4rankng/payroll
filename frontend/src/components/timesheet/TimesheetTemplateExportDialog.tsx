import { useState, memo, useCallback, useMemo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Download, Loader2, AlertCircle, Calendar as CalendarIcon } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';
import { formatDateForAPI } from '@/utils/formatters';
import { ProjectSelector } from '@/components/ui/project-selector';
import type { Project } from '@/types/api/project.types';

export interface TimesheetTemplateExportParams {
  fromDate: string;
  toDate: string;
  projectId: number;
}

interface TimesheetTemplateExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onExport: (params: TimesheetTemplateExportParams) => void;
  isLoading?: boolean;
}

export const TimesheetTemplateExportDialog = memo(function TimesheetTemplateExportDialog({
  open,
  onOpenChange,
  onExport,
  isLoading = false
}: TimesheetTemplateExportDialogProps) {
  // Calculate initial dates - default to last 10 days
  const initialDates = useMemo(() => {
    const now = new Date();
    const fromDate = new Date(now);
    fromDate.setDate(now.getDate() - 9); // 10 days including today
    return { fromDate, toDate: now };
  }, []);

  const [fromDate, setFromDate] = useState<Date | undefined>(initialDates.fromDate);
  const [toDate, setToDate] = useState<Date | undefined>(initialDates.toDate);
  const [fromDateOpen, setFromDateOpen] = useState(false);
  const [toDateOpen, setToDateOpen] = useState(false);
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);

  // Initialize dates on dialog open - set default to last 10 days
  const handleDialogOpen = useCallback((isOpen: boolean) => {
    if (isOpen) {
      const now = new Date();
      const fromDate = new Date(now);
      fromDate.setDate(now.getDate() - 9); // 10 days including today
      setFromDate(fromDate);
      setToDate(now);
      setSelectedProject(null);
    }
    onOpenChange(isOpen);
  }, [onOpenChange]);

  // Validation
  const canExport = useMemo(() => {
    return fromDate && toDate && fromDate <= toDate && selectedProject !== null;
  }, [fromDate, toDate, selectedProject]);

  const hasDateError = useMemo(() => {
    return fromDate && toDate && fromDate > toDate;
  }, [fromDate, toDate]);

  // Handle export
  const handleExport = useCallback(() => {
    if (!canExport || !fromDate || !toDate || !selectedProject) return;

    const params: TimesheetTemplateExportParams = {
      fromDate: formatDateForAPI(fromDate),
      toDate: formatDateForAPI(toDate),
      projectId: selectedProject.id,
    };

    onExport(params);
  }, [canExport, fromDate, toDate, selectedProject, onExport]);

  // Reset form when dialog closes
  const handleClose = useCallback(() => {
    if (!isLoading) {
      setFromDate(undefined);
      setToDate(undefined);
      setSelectedProject(null);
      onOpenChange(false);
    }
  }, [isLoading, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleDialogOpen}>
      <DialogContent className="max-w-[95vw] sm:max-w-2xl shadow-none p-4 sm:p-5">
        <DialogHeader className="space-y-1 pb-3">
          <DialogTitle className="text-lg">Mẫu nhập công</DialogTitle>
          <DialogDescription className="text-sm">
            Chọn dự án và khoảng thời gian để tải mẫu nhập công.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 sm:space-y-6">
          {/* Project Selection */}
          <div className="space-y-1.5">
            <Label htmlFor="project" className="text-xs text-muted-foreground">
              Dự án <span className="text-red-600">*</span>
            </Label>
            <ProjectSelector
              value={selectedProject}
              onSelect={setSelectedProject}
              placeholder="Chọn dự án..."
              activeOnly={true}
            />
          </div>

          {/* Date Range Selection */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="from-date" className="text-xs text-muted-foreground">
                Từ ngày <span className="text-red-600">*</span>
              </Label>
              <Popover open={fromDateOpen} onOpenChange={setFromDateOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      "w-full justify-start text-left font-normal h-10 px-3 text-sm min-h-[44px]",
                      !fromDate && "text-muted-foreground",
                      hasDateError && 'border-red-500 focus:ring-red-500'
                    )}
                  >
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {fromDate ? (
                      format(fromDate, "dd/MM/yyyy", { locale: vi })
                    ) : (
                      <span>Chọn ngày bắt đầu</span>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start" side="bottom" sideOffset={4}>
                  <Calendar
                    mode="single"
                    selected={fromDate}
                    onSelect={(date) => {
                      setFromDate(date);
                      setFromDateOpen(false);
                    }}
                    initialFocus
                    locale={vi}
                  />
                </PopoverContent>
              </Popover>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="to-date" className="text-xs text-muted-foreground">
                Đến ngày <span className="text-red-600">*</span>
              </Label>
              <Popover open={toDateOpen} onOpenChange={setToDateOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      "w-full justify-start text-left font-normal h-10 px-3 text-sm min-h-[44px]",
                      !toDate && "text-muted-foreground",
                      hasDateError && 'border-red-500 focus:ring-red-500'
                    )}
                  >
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {toDate ? (
                      format(toDate, "dd/MM/yyyy", { locale: vi })
                    ) : (
                      <span>Chọn ngày kết thúc</span>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0" align="start" side="bottom" sideOffset={4}>
                  <Calendar
                    mode="single"
                    selected={toDate}
                    onSelect={(date) => {
                      setToDate(date);
                      setToDateOpen(false);
                    }}
                    initialFocus
                    locale={vi}
                    disabled={(date) => fromDate ? date < fromDate : false}
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>

          {hasDateError && (
            <div className="flex items-center gap-1.5 text-xs text-red-600">
              <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
              <span>Ngày kết thúc phải sau ngày bắt đầu</span>
            </div>
          )}
        </div>

        <DialogFooter className="flex flex-row gap-2.5 sm:gap-3 pt-3">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isLoading}
            className="w-full h-10 min-h-[44px] text-sm order-2 sm:order-1"
          >
            Đóng
          </Button>
          <Button
            onClick={handleExport}
            disabled={!canExport || isLoading}
            className="w-full h-10 min-h-[44px] text-sm order-1 sm:order-2"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang tải...
              </>
            ) : (
              <>
                <Download className="w-4 h-4 mr-2" />
                Tải mẫu nhập công
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
