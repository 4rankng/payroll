import { useState, useEffect, useMemo, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "@/components/ui/sonner";
import { Plus, Calendar, FileText, AlertTriangle, DollarSign, Trash2 } from "lucide-react";
import { CreateProjectData } from "@/types/api/project.types";
import { PayrateMatrixEditor } from "@/components/payrates/components/PayrateMatrixEditor";
import { PayrateStructure } from "@/components/payrates/types";
import { SlideSheetTemplate } from "./templates/SlideSheetTemplate";
import { SalaryPeriodFields } from "@/components/projects/SalaryPeriodFields";
import { OffDaysPicker } from "@/components/projects/OffDaysPicker";
import type { ModalConfig } from "@/types/modal-config.types";

export const modalConfig: ModalConfig = {
  id: 'project_create',
  name: 'Tạo dự án mới',
  description: 'Tạo dự án mới trong hệ thống',
  category: 'project',
  permissions: {
    action: 'create',
    subject: 'Project',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: [],
    example: '?modal=project_create',
    validateParams: () => true
  },
  requiresAuth: true,
  encryptData: false,
};

interface AddProjectSheetProps {
  isOpen: boolean;
  onClose: () => void;
  onProjectCreate?: (projectData: CreateProjectData, payrates?: PayrateStructure) => Promise<void>;
}

export function AddProjectSheet({
  isOpen,
  onClose,
  onProjectCreate
}: AddProjectSheetProps) {
  const [isSaving, setIsSaving] = useState(false);

  // Get today's date in YYYY-MM-DD format
  const todayDate = useMemo(() => {
    const today = new Date();
    return today.toISOString().split('T')[0];
  }, []);

  const [formData, setFormData] = useState<CreateProjectData>({
    name: "",
    client_name: "",
    code: "",
    start_date: todayDate,
    end_date: "",
    is_weekly: true,
    is_monthly: false,
    salary_period_from: null,
    salary_period_to: null,
    off_days: 0, // default: no fixed off days
  });

  const [payrates, setPayrates] = useState<PayrateStructure>({});

  const [errors, setErrors] = useState<Record<string, string>>({});

  // Helper function to check if payrates are configured
  const hasPayratesConfigured = (rates: PayrateStructure): boolean => {
    if (!rates || typeof rates !== 'object') return false;

    const positions = Object.keys(rates);
    if (positions.length === 0) return false;

    // Check if any position has configured rates
    return positions.some(position => {
      const dayConfig = rates[position];
      if (!dayConfig || typeof dayConfig !== 'object') return false;

      return Object.values(dayConfig).some(hourConfig => {
        if (!hourConfig || typeof hourConfig !== 'object') return false;
        return Object.values(hourConfig).some(rate =>
          typeof rate === 'number' && rate > 0
        );
      });
    });
  };

  useEffect(() => {
    if (isOpen) {
      // Reset form when sheet opens with default values
      setFormData({
        name: "",
        client_name: "",
        code: "",
        start_date: todayDate,
        end_date: "",
        is_weekly: true,
        is_monthly: false,
        salary_period_from: null,
        salary_period_to: null,
        off_days: 0, // default: no fixed off days
      });
      setPayrates({});
      setErrors({});
    }
  }, [isOpen, todayDate]);

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    // Required fields - only name and client_name are truly required
    if (!formData.name.trim()) {
      newErrors.name = "Tên dự án là bắt buộc";
    }
    if (!formData.client_name.trim()) {
      newErrors.client_name = "Tên khách hàng là bắt buộc";
    }

    // Date validation - only validate end_date if it's provided
    if (formData.start_date && formData.end_date && formData.end_date.trim()) {
      const startDate = new Date(formData.start_date);
      const endDate = new Date(formData.end_date);
      if (endDate <= startDate) {
        newErrors.end_date = "Ngày kết thúc phải sau ngày bắt đầu";
      }
    }


    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      toast({
        title: "Lỗi xác thực",
        description: "Vui lòng kiểm tra và điền đầy đủ thông tin.",
        variant: "destructive"
      });
      return;
    }

    setIsSaving(true);

    try {
      // Determine status based on start date
      const today = new Date();
      today.setHours(0, 0, 0, 0); // Reset time to start of day for accurate comparison
      const startDate = new Date(formData.start_date);
      startDate.setHours(0, 0, 0, 0);

      const status: 'draft' | 'active' = startDate <= today ? 'active' : 'draft';

      // Clean up the data before submitting - remove empty strings and undefined values
      const cleanedData: CreateProjectData = {
        name: formData.name.trim(),
        client_name: formData.client_name.trim(),
        start_date: formData.start_date.trim(),
        end_date: formData.end_date.trim() || "", // Keep empty string for optional field
        status,
        is_weekly: formData.is_weekly,
        is_monthly: formData.is_monthly,
        salary_period_from: formData.salary_period_from,
        salary_period_to: formData.salary_period_to,
        off_days: formData.off_days ?? 0, // default: no fixed off days
      };

      // Only add optional fields if they have values
      if (formData.code && formData.code.trim()) {
        cleanedData.code = formData.code.trim();
      }

      if (onProjectCreate) {
        // Only pass payrates if user has configured them
        const configuredPayrates = hasPayratesConfigured(payrates) ? payrates : undefined;

        // Wait for the backend response before closing the sheet
        await onProjectCreate(cleanedData, configuredPayrates);
        // Only close after successful creation
        onClose();
      }
    } catch (error) {
      // Error handling is now done in the mutation hook
      console.error('Project creation failed:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleInputChange = useCallback((field: keyof CreateProjectData, value: string | number) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  }, [errors]);



  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-3 sm:gap-4 flex-1 min-w-0">
            <div className="h-10 w-10 rounded-full bg-primary/10 flex items-center justify-center">
              <Plus className="h-5 w-5 text-primary" />
            </div>
            <div className="space-y-1 flex-1 min-w-0">
              <h2 className="typography-headline-medium text-base sm:text-lg font-semibold">
                Tạo dự án mới
              </h2>
              <p className="typography-body-medium text-muted-foreground text-sm">
                Tạo dự án mới để quản lý nhân viên và timesheet
              </p>
            </div>
          </div>
        )
      }}
      size="large"
      footer={
        <div className="w-full space-y-3">
          <div className="flex gap-3 w-full">
            <Button
              variant="outline"
              onClick={onClose}
              disabled={isSaving}
              className="flex-1 h-11"
            >
              Đóng
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={isSaving}
              className="flex-1 h-11"
            >
              {isSaving ? "Đang tạo..." : "Tạo dự án"}
            </Button>
          </div>
        </div>
      }
    >
      <div className="space-y-6">
        {/* Two-column layout for basic info and timeline */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 p-6 pb-0">
          {/* Left Column - Basic Information */}
          <div className="space-y-6">
            <div className="space-y-4">
              <div className="flex items-center gap-2 pb-2 border-b">
                <FileText className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Thông tin cơ bản</h3>
              </div>

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="name" className="text-sm font-medium">
                    Tên dự án <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => handleInputChange('name', e.target.value)}
                    className={`transition-colors ${errors.name ? "border-red-500 focus-visible:ring-red-500" : ""}`}
                    placeholder="Nhập tên dự án"
                    disabled={isSaving}
                  />
                  {errors.name && (
                    <p className="text-xs text-red-500 flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      {errors.name}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="client_name" className="text-sm font-medium">
                    Tên khách hàng <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="client_name"
                    value={formData.client_name}
                    onChange={(e) => handleInputChange('client_name', e.target.value)}
                    className={`transition-colors ${errors.client_name ? "border-red-500 focus-visible:ring-red-500" : ""}`}
                    placeholder="Nhập tên khách hàng"
                    disabled={isSaving}
                  />
                  {errors.client_name && (
                    <p className="text-xs text-red-500 flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      {errors.client_name}
                    </p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="code" className="text-sm font-medium">
                    Mã dự án
                  </Label>
                  <Input
                    id="code"
                    value={formData.code}
                    onChange={(e) => handleInputChange('code', e.target.value)}
                    className={`transition-colors font-mono ${errors.code ? "border-red-500 focus-visible:ring-red-500" : ""}`}
                    placeholder="VD: PRJ001"
                    disabled={isSaving}
                  />
                  {errors.code && (
                    <p className="text-xs text-red-500 flex items-center gap-1">
                      <AlertTriangle className="h-3 w-3" />
                      {errors.code}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Right Column - Timeline */}
          <div className="space-y-6">
            <div className="space-y-4">
              <div className="flex items-center gap-2 pb-2 border-b">
                <Calendar className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Thời gian dự án</h3>
              </div>

              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="start_date" className="text-sm font-medium">
                      Ngày bắt đầu
                    </Label>
                    <Input
                      id="start_date"
                      type="date"
                      value={formData.start_date}
                      onChange={(e) => handleInputChange('start_date', e.target.value)}
                      className={`transition-colors ${errors.start_date ? "border-red-500 focus-visible:ring-red-500" : ""}`}
                      disabled={isSaving}
                    />
                    {errors.start_date && (
                      <p className="text-xs text-red-500 flex items-center gap-1">
                        <AlertTriangle className="h-3 w-3" />
                        {errors.start_date}
                      </p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="end_date" className="text-sm font-medium">
                      Ngày kết thúc
                    </Label>
                    <Input
                      id="end_date"
                      type="date"
                      value={formData.end_date}
                      onChange={(e) => handleInputChange('end_date', e.target.value)}
                      className={`transition-colors ${errors.end_date ? "border-red-500 focus-visible:ring-red-500" : ""}`}
                      disabled={isSaving}
                    />
                    {errors.end_date && (
                      <p className="text-xs text-red-500 flex items-center gap-1">
                        <AlertTriangle className="h-3 w-3" />
                        {errors.end_date}
                      </p>
                    )}
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  Ngày kết thúc là tùy chọn
                </p>
              </div>
            </div>

            <div className="pt-2">
              <SalaryPeriodFields
                salaryPeriodFrom={formData.salary_period_from}
                salaryPeriodTo={formData.salary_period_to}
                onSalaryPeriodFromChange={(value) => handleInputChange('salary_period_from', value)}
                onSalaryPeriodToChange={(value) => handleInputChange('salary_period_to', value)}
                disabled={isSaving}
              />
            </div>

            <div className="pt-2 space-y-2">
              <Label className="text-sm font-medium">Ngày nghỉ trong tuần</Label>
              <OffDaysPicker
                value={formData.off_days ?? 0}
                onChange={(v) => handleInputChange('off_days', v)}
                disabled={isSaving}
              />
              <p className="text-xs text-muted-foreground">Chọn các ngày nghỉ cố định trong tuần của dự án</p>
            </div>
          </div>
        </div>

        {/* Full-width Payrate Configuration Section */}
        <div className="px-6 pb-6 pt-2">
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-2 border-b">
              <div className="flex items-center gap-2">
                <DollarSign className="h-5 w-5 text-primary" />
                <h3 className="text-lg font-semibold">Cấu hình lương giờ</h3>
              </div>
              {Object.keys(payrates).length > 0 && (
                <Button
                  onClick={() => setPayrates({})}
                  variant="outline"
                  size="sm"
                  disabled={isSaving}
                  className="text-destructive hover:text-destructive-foreground hover:bg-destructive"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  Xóa tất cả
                </Button>
              )}
            </div>

            <p className="text-sm text-muted-foreground">
              Thiết lập mức lương theo giờ cho dự án (tùy chọn)
            </p>

            <PayrateMatrixEditor
              rates={payrates}
              onChange={setPayrates}
              readOnly={isSaving}
            />
          </div>
        </div>
      </div>
    </SlideSheetTemplate>
  );
}

export default AddProjectSheet;
