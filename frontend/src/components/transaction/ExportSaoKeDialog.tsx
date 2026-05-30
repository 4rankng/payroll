import { useState, useMemo, useCallback } from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Download, X, Calendar, Loader2, Search } from 'lucide-react';
import { cn } from '@/lib/utils';
import { dateToString } from '@/utils/dateHelpers';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';
import { apiClient, buildQueryString } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';
import { useProjects } from '@/hooks/api/useProjects';
import { getErrorMessage } from '@/utils/error-handler';

interface ExportSaoKeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ExportSaoKeDialog({ open, onOpenChange }: ExportSaoKeDialogProps) {
  const { data: projectsData } = useProjects({ status: 'active', pageSize: 100 });
  const projects = useMemo(() => projectsData?.data || [], [projectsData]);

  const defaultDateRange = useMemo(() => {
    const now = new Date();
    return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
    };
  }, []);

  const [selectedProjectIds, setSelectedProjectIds] = useState<number[]>([]);
  const [isAllProjectsSelected, setIsAllProjectsSelected] = useState(true);
  const [projectSearch, setProjectSearch] = useState('');
  const [fromDate, setFromDate] = useState(defaultDateRange.fromDate);
  const [toDate, setToDate] = useState(defaultDateRange.toDate);
  const [isExporting, setIsExporting] = useState(false);
  const [error, setError] = useState('');

  const filteredProjects = useMemo(() => {
    if (!projectSearch.trim()) return projects;
    return projects.filter(
      (p) => vietnameseIncludes(p.name, projectSearch) || vietnameseIncludes(p.code, projectSearch),
    );
  }, [projects, projectSearch]);

  const handleSelectAll = useCallback(() => {
    setIsAllProjectsSelected(true);
    setSelectedProjectIds([]);
  }, []);

  const handleToggleProject = useCallback((id: number) => {
    setIsAllProjectsSelected(false);
    setSelectedProjectIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id],
    );
  }, []);

  const handleExport = async () => {
    if (!fromDate || !toDate) return;

    setIsExporting(true);
    setError('');
    try {
      const params: Record<string, string> = { fromDate, toDate };
      if (!isAllProjectsSelected && selectedProjectIds.length > 0) {
        params.projectIds = selectedProjectIds.join(',');
      }
      const qs = buildQueryString(params);
      await apiClient.download(
        `${API_ENDPOINTS.timesheets.payrollReport}${qs}`,
        `sao_ke_${fromDate}_${toDate}.xlsx`,
      );
      onOpenChange(false);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setIsExporting(false);
    }
  };

  const handlePresetClick = (preset: string) => {
    const now = new Date();
    let f = '';
    let t = '';
    switch (preset) {
      case 'this-month':
        f = dateToString(new Date(now.getFullYear(), now.getMonth(), 1));
        t = dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0));
        break;
      case 'last-month':
        f = dateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1));
        t = dateToString(new Date(now.getFullYear(), now.getMonth(), 0));
        break;
      case 'this-quarter': {
        const q = Math.floor(now.getMonth() / 3);
        f = dateToString(new Date(now.getFullYear(), q * 3, 1));
        t = dateToString(new Date(now.getFullYear(), q * 3 + 3, 0));
        break;
      }
      case 'last-3-months':
        f = dateToString(new Date(now.getFullYear(), now.getMonth() - 3, 1));
        t = dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0));
        break;
    }
    if (f && t) {
      setFromDate(f);
      setToDate(t);
    }
  };

  const canExport = fromDate && toDate && !isExporting;

  return (
    <Dialog open={open} onOpenChange={(v) => !isExporting && onOpenChange(v)}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Xuất sao kê
          </DialogTitle>
          <DialogDescription>Chọn dự án và khoảng thời gian để xuất sao kê</DialogDescription>
        </DialogHeader>

        <div className="flex gap-6 py-2 min-h-0 flex-1 overflow-hidden">
          {/* Left — Date Presets + Date Inputs */}
          <div className="space-y-3 w-44 shrink-0">
            <Label className="text-sm font-medium">Khoảng thời gian</Label>
            <div className="flex flex-col gap-2">
              {(['this-month', 'last-month', 'this-quarter', 'last-3-months'] as const).map((p) => (
                <Button
                  key={p}
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => handlePresetClick(p)}
                  className="justify-start w-full"
                >
                  <Calendar className="h-3.5 w-3.5 mr-2" />
                  {p === 'this-month' && 'Tháng này'}
                  {p === 'last-month' && 'Tháng trước'}
                  {p === 'this-quarter' && 'Quý này'}
                  {p === 'last-3-months' && '3 tháng gần đây'}
                </Button>
              ))}
            </div>
            <div className="space-y-2 pt-2">
              <div className="space-y-1">
                <Label htmlFor="fromDate" className="text-xs text-muted-foreground">Từ ngày</Label>
                <Input id="fromDate" type="date" value={fromDate} onChange={(e) => setFromDate(e.target.value)} className="h-9 text-sm" />
              </div>
              <div className="space-y-1">
                <Label htmlFor="toDate" className="text-xs text-muted-foreground">Đến ngày</Label>
                <Input id="toDate" type="date" value={toDate} onChange={(e) => setToDate(e.target.value)} className="h-9 text-sm" />
              </div>
            </div>
          </div>

          {/* Right — Project Grid */}
          <div className="flex-1 flex flex-col min-h-0 space-y-2">
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">Dự án</Label>
              <Button
                type="button"
                variant={isAllProjectsSelected ? 'default' : 'outline'}
                size="sm"
                onClick={handleSelectAll}
                className="h-7 text-xs"
              >
                Tất cả
              </Button>
            </div>

            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="Tìm dự án..."
                value={projectSearch}
                onChange={(e) => setProjectSearch(e.target.value)}
                className="pl-8 h-9 text-sm"
              />
            </div>

            <div className="flex-1 overflow-y-auto -mx-1 px-1 min-h-0" style={{ maxHeight: '340px' }}>
              <div className="grid grid-cols-2 gap-1.5">
                {filteredProjects.length === 0 ? (
                  projectSearch && (
                    <div className="col-span-2 text-center text-muted-foreground text-xs py-6">
                      Không tìm thấy dự án
                    </div>
                  )
                ) : (
                  filteredProjects.map((project) => {
                    const isSelected = !isAllProjectsSelected && selectedProjectIds.includes(project.id);
                    return (
                      <button
                        key={project.id}
                        type="button"
                        onClick={() => handleToggleProject(project.id)}
                        className={cn(
                          'text-left px-2.5 py-1.5 rounded-md border transition-colors min-h-[44px]',
                          'hover:bg-muted/80 focus:outline-none focus:ring-2 focus:ring-ring',
                          isSelected
                            ? 'bg-primary/10 border-primary/30'
                            : 'border-transparent',
                        )}
                      >
                        <div className="font-medium text-xs truncate">{project.name}</div>
                        <div className="text-[10px] text-muted-foreground mt-0.5">{project.code}</div>
                      </button>
                    );
                  })
                )}
              </div>
            </div>

            {!isAllProjectsSelected && selectedProjectIds.length > 0 && (
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground pt-1 shrink-0">
                <span>Đã chọn {selectedProjectIds.length}/{projects.length}</span>
              </div>
            )}
            {isAllProjectsSelected && (
              <div className="text-xs text-muted-foreground pt-1 shrink-0">
                Tất cả dự án ({projects.length})
              </div>
            )}
          </div>
        </div>

        {error && <p className="text-xs text-destructive px-4">{error}</p>}

        <DialogFooter className="grid grid-cols-2 gap-2 shrink-0">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isExporting} className="w-full">
            <X className="h-4 w-4 mr-2" />
            Hủy
          </Button>
          <Button type="button" onClick={handleExport} disabled={!canExport} className="w-full">
            {isExporting ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Download className="h-4 w-4 mr-2" />}
            {isExporting ? 'Đang xuất...' : 'Xuất file'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
