import { useState, useEffect, useMemo } from "react";
import { Project, UpdateProjectData } from "@/types/api/project.types";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useUpdateProject } from "@/hooks/api/useProjects";
import { Edit3, AlertTriangle, CheckCircle, Edit, XCircle, Pause } from "lucide-react";
import type { ModalConfig } from "@/types/modal-config.types";
import { ProjectStatus } from "@/components/projects/ProjectStatusBadge";
import { getVietnameseProjectStatus } from "@/utils/vietnamese";
import { OffDaysPicker } from "@/components/projects/OffDaysPicker";
import { authManager } from "@/lib/auth";

interface ProjectEditSheetProps {
  project: Project | null;
  isOpen: boolean;
  onClose: () => void;
  onProjectUpdated?: (updatedProject: Project) => void;
}

interface ProjectFormData {
  name: string;
  code: string;
  description: string;
  client_name: string;
  start_date: string;
  end_date: string;
  status: ProjectStatus;
  is_weekly: boolean;
  is_monthly: boolean;
  salary_period_from: number | null;
  salary_period_to: number | null;
  off_days: number;
}

// Helper function to format date for input[type="date"]
const formatDateForInput = (dateString: string | null | undefined): string => {
  if (!dateString) return "";
  return dateString ? dateString.split('T')[0] : "";
};

function ProjectEditSheet({
  project,
  isOpen,
  onClose,
  onProjectUpdated
}: ProjectEditSheetProps) {
  const updateProjectMutation = useUpdateProject();
  const isAdmin = authManager.hasRole("admin");

  const [formData, setFormData] = useState<ProjectFormData>({
    name: "",
    code: "",
    description: "",
    client_name: "",
    start_date: "",
    end_date: "",
    status: "draft",
    is_weekly: true,
    is_monthly: false,
    salary_period_from: null,
    salary_period_to: null,
    off_days: 0,
  });
  const [hasChanges, setHasChanges] = useState(false);
  const [errors, setErrors] = useState<Partial<ProjectFormData>>({});

  // Memoize original data with optimized date formatting
  const originalData = useMemo(() => {
    if (!project) return null;
    return {
      name: project.name || "",
      code: project.code || "",
      description: project.description || "",
      client_name: project.client_name || "",
      start_date: formatDateForInput(project.start_date),
      end_date: formatDateForInput(project.end_date),
      status: (project.status as ProjectStatus) || "draft",
      is_weekly: project.is_weekly ?? true,
      is_monthly: project.is_monthly ?? false,
      salary_period_from: project.salary_period_from ?? null,
      salary_period_to: project.salary_period_to ?? null,
      off_days: project.off_days ?? 0,
    };
  }, [project]);

  // Reset form when project changes or sheet opens
  useEffect(() => {
    if (isOpen && originalData) {
      setFormData(originalData);
      setHasChanges(false);
      setErrors({});
    }
  }, [isOpen, originalData]);

  // Check for changes
  useEffect(() => {
    if (!originalData) return;

    const changed = Object.keys(formData).some((key) => {
      const formKey = key as keyof ProjectFormData;
      return formData[formKey] !== originalData[formKey];
    });

    setHasChanges(changed);
  }, [formData, originalData]);

  const validateForm = (): boolean => {
    const newErrors: Partial<ProjectFormData> = {};

    if (!formData.name.trim()) {
      newErrors.name = "Tên dự án là bắt buộc";
    } else if (formData.name.length > 100) {
      newErrors.name = "Tên dự án không được vượt quá 100 ký tự";
    }

    if (!formData.client_name.trim()) {
      newErrors.client_name = "Tên khách hàng là bắt buộc";
    } else if (formData.client_name.length > 100) {
      newErrors.client_name = "Tên khách hàng không được vượt quá 100 ký tự";
    }

    // Code validation - optional for both admin and partner
    if (formData.code.trim() && formData.code.length > 50) {
      newErrors.code = "Mã dự án không được vượt quá 50 ký tự";
    }

    if (formData.description && formData.description.length > 1000) {
      newErrors.description = "Mô tả không được vượt quá 1000 ký tự";
    }

    if (formData.start_date && formData.end_date) {
      if (new Date(formData.start_date) > new Date(formData.end_date)) {
        newErrors.end_date = "Ngày kết thúc phải sau ngày bắt đầu";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleInputChange = (field: keyof ProjectFormData, value: string | number | boolean | null) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({
        ...prev,
        [field]: undefined
      }));
    }
  };


  const handleSave = async () => {
    if (!project || !validateForm()) return;

    try {
      const updateData: UpdateProjectData = {
        name: formData.name,
        code: formData.code,
        description: formData.description,
        client_name: formData.client_name,
        start_date: formData.start_date || undefined,
        end_date: formData.end_date || undefined,
        is_weekly: formData.is_weekly,
        is_monthly: formData.is_monthly,
        salary_period_from: formData.salary_period_from,
        salary_period_to: formData.salary_period_to,
        off_days: formData.off_days,
      };

      const response = await updateProjectMutation.mutateAsync({
        id: project.id,
        data: updateData
      });

      if (onProjectUpdated && response?.data) {
        onProjectUpdated(response.data);
      }

      onClose();
    } catch (error) {
      console.error('Failed to update project:', error);
    }
  };

  const handleClose = () => {
    if (hasChanges) {
      // Show confirmation dialog using modal component (following CLAUDE.md guidelines)
      // For now, just close - we can enhance this later with a proper modal
      onClose();
    } else {
      onClose();
    }
  };

  if (!project) return null;

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={handleClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
              <Edit3 className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <h2 className="text-sm font-semibold leading-tight">Chỉnh sửa dự án</h2>
              <p className="text-xs text-muted-foreground truncate">{project.name}</p>
            </div>
          </div>
        )
      }}
      size="large"
      footer={
        <div className="w-full flex items-center gap-3">
          {hasChanges && (
            <p className="text-xs text-muted-foreground shrink-0">• Có thay đổi chưa lưu</p>
          )}
          <div className="flex gap-2 ml-auto">
            <Button variant="outline" onClick={handleClose} className="h-8 px-4 text-sm">
              Hủy bỏ
            </Button>
            <Button
              onClick={handleSave}
              disabled={!hasChanges || updateProjectMutation.isPending}
              className="h-8 px-4 text-sm"
              variant="default"
            >
              {updateProjectMutation.isPending ? "Đang lưu..." : "Lưu thay đổi"}
            </Button>
          </div>
        </div>
      }
    >
      <div className="p-4 space-y-4">
        {/* Row 1: name + client on top, code + status below — stacks to 1-col on mobile */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {/* Tên dự án */}
          <div className="space-y-1">
            <Label htmlFor="name" className="text-xs text-muted-foreground">Tên dự án *</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              className={`h-8 text-sm ${errors.name ? "border-red-500" : ""}`}
              placeholder="Tên dự án"
            />
            {errors.name && <p className="text-xs text-red-500 flex items-center gap-1"><AlertTriangle className="h-3 w-3" />{errors.name}</p>}
          </div>

          {/* Khách hàng */}
          <div className="space-y-1">
            <Label htmlFor="client_name" className="text-xs text-muted-foreground">Khách hàng *</Label>
            <Input
              id="client_name"
              value={formData.client_name}
              onChange={(e) => handleInputChange('client_name', e.target.value)}
              className={`h-8 text-sm ${errors.client_name ? "border-red-500" : ""}`}
              placeholder="Tên khách hàng"
            />
            {errors.client_name && <p className="text-xs text-red-500 flex items-center gap-1"><AlertTriangle className="h-3 w-3" />{errors.client_name}</p>}
          </div>

          {/* Mã dự án */}
          <div className="space-y-1">
            <Label htmlFor="code" className="text-xs text-muted-foreground">Mã dự án</Label>
            <Input
              id="code"
              value={formData.code}
              onChange={(e) => handleInputChange('code', e.target.value)}
              className={`h-8 text-sm font-mono ${errors.code ? "border-red-500" : ""}`}
              placeholder="VD: PRJ001"
            />
            {errors.code && <p className="text-xs text-red-500 flex items-center gap-1"><AlertTriangle className="h-3 w-3" />{errors.code}</p>}
          </div>

          {/* Trạng thái */}
          <div className="space-y-1">
            <Label htmlFor="status" className="text-xs text-muted-foreground">Trạng thái *</Label>
            <Select
              value={formData.status || 'draft'}
              onValueChange={(value: ProjectStatus) => handleInputChange('status', value)}
            >
              <SelectTrigger className="h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="draft"><div className="flex items-center gap-2"><Edit className="h-3.5 w-3.5 text-muted-foreground" />{getVietnameseProjectStatus('draft')}</div></SelectItem>
                <SelectItem value="active"><div className="flex items-center gap-2"><CheckCircle className="h-3.5 w-3.5 text-green-500" />{getVietnameseProjectStatus('active')}</div></SelectItem>
                <SelectItem value="completed"><div className="flex items-center gap-2"><CheckCircle className="h-3.5 w-3.5 text-blue-500" />{getVietnameseProjectStatus('completed')}</div></SelectItem>
                <SelectItem value="paused"><div className="flex items-center gap-2"><Pause className="h-3.5 w-3.5 text-orange-500" />{getVietnameseProjectStatus('paused')}</div></SelectItem>
                <SelectItem value="cancelled"><div className="flex items-center gap-2"><XCircle className="h-3.5 w-3.5 text-red-500" />{getVietnameseProjectStatus('cancelled')}</div></SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Row 2: left = dates+salary+offdays, right = description — stacks on mobile */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div className="space-y-3">
            {/* Dates */}
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1">
                <Label htmlFor="start_date" className="text-xs text-muted-foreground">Ngày bắt đầu</Label>
                <Input id="start_date" type="date" value={formData.start_date} onChange={(e) => handleInputChange('start_date', e.target.value)} className="h-8 text-sm" />
              </div>
              <div className="space-y-1">
                <Label htmlFor="end_date" className="text-xs text-muted-foreground">Ngày kết thúc</Label>
                <Input id="end_date" type="date" value={formData.end_date} onChange={(e) => handleInputChange('end_date', e.target.value)} className={`h-8 text-sm ${errors.end_date ? "border-red-500" : ""}`} />
                {errors.end_date && <p className="text-xs text-red-500">{errors.end_date}</p>}
              </div>
            </div>

            {/* Salary period */}
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">Kỳ lương tháng</Label>
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-0.5">
                  <Input
                    type="number" min={0} max={28}
                    value={formData.salary_period_from ?? ''}
                    onChange={(e) => handleInputChange('salary_period_from', e.target.value === '' ? null : Number(e.target.value))}
                    className="h-8 text-sm"
                    placeholder="Từ ngày"
                    disabled={!isAdmin}
                  />
                  <p className="text-[10px] text-muted-foreground">Tháng trước (0 = ngày 01)</p>
                </div>
                <div className="space-y-0.5">
                  <Input
                    type="number" min={0} max={28}
                    value={formData.salary_period_to ?? ''}
                    onChange={(e) => handleInputChange('salary_period_to', e.target.value === '' ? null : Number(e.target.value))}
                    className="h-8 text-sm"
                    placeholder="Đến ngày"
                    disabled={!isAdmin}
                  />
                  <p className="text-[10px] text-muted-foreground">Tháng này (0 = cuối tháng)</p>
                </div>
              </div>
            </div>

            {/* Off days */}
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">Ngày nghỉ trong tuần</Label>
              <OffDaysPicker
                value={formData.off_days}
                onChange={(v) => handleInputChange('off_days', v)}
              />
            </div>
          </div>

          {/* Description */}
          <div className="space-y-1">
            <Label htmlFor="description" className="text-xs text-muted-foreground">Mô tả dự án</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              rows={10}
              className={`resize-none text-sm ${errors.description ? "border-red-500" : ""}`}
              placeholder="Nhập mô tả chi tiết về dự án..."
            />
            {errors.description && <p className="text-xs text-red-500">{errors.description}</p>}
            <p className="text-[10px] text-muted-foreground text-right">{formData.description.length}/1000</p>
          </div>
        </div>
      </div>
    </SlideSheetTemplate>
  );
}

export const modalConfig: ModalConfig = {
  id: 'project-edit-sheet',
  name: 'Project Edit Sheet',
  category: 'project',
  permissions: {
    action: 'update',
    subject: 'Project',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=project_edit_sheet&id=123',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  requiresAuth: true,
  encryptData: false,
};

export default ProjectEditSheet;
