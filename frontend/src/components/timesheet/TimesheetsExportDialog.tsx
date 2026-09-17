import { useState, memo, useCallback, useMemo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Download, Loader2, AlertCircle, Calendar as CalendarIcon, Search } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';
import { formatDateForAPI } from '@/utils/formatters';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';
import { useProjects } from '@/hooks/api/useProjects';

export interface TimesheetsExportParams {
  fromDate: string;
  toDate: string;
  project_ids?: string;
  status?: string;
}

interface TimesheetsExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onExport: (params: TimesheetsExportParams) => void;
  isLoading?: boolean;
}

export const TimesheetsExportDialog = memo(function TimesheetsExportDialog({
  open,
  onOpenChange,
  onExport,
  isLoading = false
}: TimesheetsExportDialogProps) {
  // Calculate initial dates with month-to-date default
  const initialDates = useMemo(() => {
    const now = new Date();
    const firstDayOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
    return { fromDate: firstDayOfMonth, toDate: now };
  }, []);

  const [fromDate, setFromDate] = useState<Date>(initialDates.fromDate);
  const [toDate, setToDate] = useState<Date>(initialDates.toDate);
  const [fromDateOpen, setFromDateOpen] = useState(false);
  const [toDateOpen, setToDateOpen] = useState(false);
  const [selectedProjectIds, setSelectedProjectIds] = useState<number[]>([]);
  const [projectSearch, setProjectSearch] = useState('');
  const [status, setStatus] = useState<string>('all');
  const [isAllProjectsSelected, setIsAllProjectsSelected] = useState<boolean>(true);

  // Fetch all projects (no status filter = all projects)
  const { data: projectsData } = useProjects({ pageSize: 100 });
  const projects = useMemo(() => projectsData?.data || [], [projectsData]);

  // Filter projects based on search
  const filteredProjects = useMemo(() => {
    if (!projectSearch.trim()) return projects;
    return projects.filter(
      (p) =>
        vietnameseIncludes(p.name, projectSearch) ||
        vietnameseIncludes(p.code, projectSearch)
    );
  }, [projects, projectSearch]);

  // Get selected projects details
  const selectedProjects = useMemo(() => {
    return projects.filter((p) => selectedProjectIds.includes(p.id));
  }, [projects, selectedProjectIds]);

  // Initialize dates on dialog open - set default to month to date
  const handleDialogOpen = useCallback((isOpen: boolean) => {
    if (isOpen) {
      const now = new Date();
      const firstDay = new Date(now.getFullYear(), now.getMonth(), 1);
      setFromDate(firstDay);
      setToDate(now);
      setSelectedProjectIds([]); // Reset project selection
      setProjectSearch(''); // Reset search
      setStatus('all'); // Reset status to all
      setIsAllProjectsSelected(true); // Reset to all projects
    }
    onOpenChange(isOpen);
  }, [onOpenChange]);

  // Validation
  const canExport = useMemo(() => {
    return fromDate && toDate && fromDate <= toDate;
  }, [fromDate, toDate]);

  const hasDateError = useMemo(() => {
    return fromDate && toDate && fromDate > toDate;
  }, [fromDate, toDate]);

  // Handle "All Projects" selection
  const handleSelectAllProjects = useCallback(() => {
    setIsAllProjectsSelected(true);
    setSelectedProjectIds([]);
  }, []);

  // Toggle individual project
  const handleToggleProject = useCallback((projectId: number) => {
    setIsAllProjectsSelected(false);
    setSelectedProjectIds((prev) =>
      prev.includes(projectId)
        ? prev.filter((id) => id !== projectId)
        : [...prev, projectId]
    );
  }, []);

  // Remove project by id
  const handleRemoveProject = useCallback((projectId: number) => {
    setSelectedProjectIds((prev) => prev.filter((id) => id !== projectId));
  }, []);

  // Handle export
  const handleExport = useCallback(() => {
    if (!canExport || !fromDate || !toDate) return;

    const params: TimesheetsExportParams = {
      fromDate: formatDateForAPI(fromDate),
      toDate: formatDateForAPI(toDate),
    };

    // Add project_ids if specific projects are selected (not "all projects")
    if (!isAllProjectsSelected && selectedProjectIds.length > 0) {
      params.project_ids = selectedProjectIds.join(',');
    }

    // Add status if not "all"
    if (status && status !== 'all') {
      params.status = status;
    }

    onExport(params);
  }, [canExport, fromDate, toDate, isAllProjectsSelected, selectedProjectIds, status, onExport]);

  // Reset form when dialog closes
  const handleClose = useCallback(() => {
    if (!isLoading) {
      setFromDate(undefined);
      setToDate(undefined);
      setSelectedProjectIds([]);
      setProjectSearch('');
      setStatus('all');
      setIsAllProjectsSelected(true);
      onOpenChange(false);
    }
  }, [isLoading, onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={handleDialogOpen}>
      <DialogContent className="max-h-[92dvh] max-w-[95vw] overflow-y-auto p-4 shadow-none sm:max-w-3xl sm:p-5">
        <DialogHeader className="space-y-1 pb-3">
          <DialogTitle className="text-lg">Xuất bảng công</DialogTitle>
          <DialogDescription className="text-sm">
            Chọn khoảng thời gian và trạng thái để xuất bảng công.
          </DialogDescription>
        </DialogHeader>

        {/* 2x2 grid layout - "Từ ngày" and "Đến ngày" on same row, "Trạng thái" and "Dự án" on the next row */}
        <div className="space-y-4 sm:space-y-6">
          {/* Row 1: "Từ ngày" and "Đến ngày" side by side */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="from-date" className="text-xs text-muted-foreground">Từ ngày</Label>
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
              <Label htmlFor="to-date" className="text-xs text-muted-foreground">Đến ngày</Label>
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

          {/* Row 2: "Trạng thái" and "Dự án" side by side */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Status Filter */}
            <div className="space-y-1.5">
              <Label htmlFor="status" className="text-xs text-muted-foreground">Trạng thái</Label>
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger id="status" className="w-full h-10 min-h-[44px]">
                  <SelectValue placeholder="Chọn trạng thái" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả trạng thái</SelectItem>
                  <SelectItem value="pending_approval">Chờ duyệt</SelectItem>
                  <SelectItem value="approved">Đã duyệt</SelectItem>
                  <SelectItem value="rejected">Bị loại</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Project Selection */}
            <div className="space-y-1.5">
              <Label htmlFor="project" className="text-xs text-muted-foreground">Dự án</Label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className="w-full justify-start text-left font-normal h-10 px-3 text-sm min-h-[44px]"
                  >
                    {isAllProjectsSelected
                      ? 'Tất cả dự án'
                      : selectedProjectIds.length > 0
                        ? `${selectedProjectIds.length} dự án`
                        : 'Chọn dự án'}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[calc(100vw-2rem)] max-w-[360px] p-0" align="start">
                  <div className="p-2 space-y-2">
                    {/* Search Input */}
                    <div className="relative">
                      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                      <Input
                        placeholder="Tìm dự án theo tên hoặc mã..."
                        value={projectSearch}
                        onChange={(e) => setProjectSearch(e.target.value)}
                        className="min-h-11 pl-8 text-sm"
                      />
                    </div>

                    {/* Project List */}
                    <ScrollArea className="h-[240px]">
                      <div className="space-y-0.5">
                        {/* "All Projects" Option */}
                        <button
                          type="button"
                          onClick={handleSelectAllProjects}
                          className={cn(
                            "min-h-11 w-full rounded px-3 py-2.5 text-left transition-colors",
                            "hover:bg-muted/80 focus:outline-none focus:ring-2 focus:ring-ring",
                            isAllProjectsSelected && "bg-primary/10 hover:bg-primary/15"
                          )}
                        >
                          <div className="font-medium text-xs">Tất cả dự án</div>
                        </button>

                        {/* Individual Projects */}
                        {filteredProjects.length === 0 ? (
                          projectSearch && (
                            <div className="text-center text-muted-foreground text-xs py-6">
                              Không tìm thấy dự án
                            </div>
                          )
                        ) : (
                          filteredProjects.map((project) => {
                            const isSelected = selectedProjectIds.includes(project.id);
                            return (
                              <button
                                key={project.id}
                                type="button"
                                onClick={() => handleToggleProject(project.id)}
                                className={cn(
                                  "min-h-11 w-full rounded px-3 py-2.5 text-left transition-colors",
                                  "hover:bg-muted/80 focus:outline-none focus:ring-2 focus:ring-ring",
                                  isSelected && "bg-primary/10 hover:bg-primary/15"
                                )}
                              >
                                <div className="break-words text-xs font-medium leading-snug">{project.name}</div>
                                <div className="text-xs text-muted-foreground mt-0.5">
                                  {project.code}
                                </div>
                              </button>
                            );
                          })
                        )}
                      </div>
                    </ScrollArea>
                  </div>
                </PopoverContent>
              </Popover>
            </div>
          </div>

          {/* Selected Projects Display */}
          {!isAllProjectsSelected && selectedProjects.length > 0 && (
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Đã chọn</Label>
              <div className="min-h-[60px] max-h-36 overflow-y-auto rounded border bg-muted/20 p-2 text-xs">
                {selectedProjects.map((project, index) => (
                  <div key={project.id} className="py-1.5">
                    <span className="break-words leading-snug">{index + 1}. {project.name}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="flex flex-row gap-2.5 pt-3 sm:gap-3">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isLoading}
            className="h-11 flex-1 basis-0 text-sm sm:flex-none"
          >
            Đóng
          </Button>
          <Button
            onClick={handleExport}
            disabled={!canExport || isLoading}
            className="h-11 flex-1 basis-0 text-sm sm:flex-none"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang xuất...
              </>
            ) : (
              <>
                <Download className="w-4 h-4 mr-2" />
                Xuất bảng công
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
