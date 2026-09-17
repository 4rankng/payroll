import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";
import { CreateProjectData } from "@/types/api/project.types";
import { X } from "lucide-react";
import { SalaryPeriodFields } from "@/components/projects/SalaryPeriodFields";
import type { ModalConfig } from '@/types/modal-config.types';

export const modalConfig: ModalConfig = {
  id: 'project-create',
  name: 'Create Project',
  description: 'Create a new project',
  category: 'project',
  permissions: {
    action: 'create',
    subject: 'Project',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: false,
  },
  requiresAuth: true,
  encryptData: false,
};

interface ProjectCreateModalProps {
  onProjectCreate: (projectData: CreateProjectData) => void;
  isOpen?: boolean;
  onClose?: () => void;
}

export function ProjectCreateModal({ onProjectCreate, isOpen = false, onClose }: ProjectCreateModalProps) {
  const [internalOpen, setInternalOpen] = useState(false);
  const modalOpen = onClose ? isOpen : internalOpen;
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formData, setFormData] = useState<CreateProjectData>({
    name: "",
    client_name: "",
    code: "",
    description: "",
    start_date: "",
    end_date: "",
    salary_period_from: null,
    salary_period_to: null,
  });

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    // Required fields
    if (!formData.name.trim()) {
      newErrors.name = "Tên dự án là bắt buộc";
    }
    if (!formData.client_name.trim()) {
      newErrors.client_name = "Tên khách hàng là bắt buộc";
    }

    // Date validation
    if (formData.start_date && formData.end_date) {
      const startDate = new Date(formData.start_date);
      const endDate = new Date(formData.end_date);
      if (endDate <= startDate) {
        newErrors.end_date = "Ngày kết thúc phải sau ngày bắt đầu";
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = () => {
    if (!validateForm()) {
      return;
    }

    // Clean up the data before submitting - remove empty strings and undefined values
    const cleanedData: CreateProjectData = {
      name: formData.name.trim(),
      client_name: formData.client_name.trim(),
      start_date: formData.start_date || new Date().toISOString().split('T')[0],
      end_date: formData.end_date || new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    };

    // Only add optional fields if they have values
    if (formData.code && formData.code.trim()) {
      cleanedData.code = formData.code.trim();
    }
    if (formData.description && formData.description.trim()) {
      cleanedData.description = formData.description.trim();
    }
    if (formData.start_date && formData.start_date.trim()) {
      cleanedData.start_date = formData.start_date.trim();
    }
    if (formData.end_date && formData.end_date.trim()) {
      cleanedData.end_date = formData.end_date.trim();
    }

    // Add salary period fields (can be null)
    cleanedData.salary_period_from = formData.salary_period_from;
    cleanedData.salary_period_to = formData.salary_period_to;

    onProjectCreate(cleanedData);
    if (onClose) {
      onClose();
    } else {
      setInternalOpen(false);
    }
    // Reset form
    setFormData({
      name: "",
      client_name: "",
      code: "",
      description: "",
      start_date: "",
      end_date: "",
      salary_period_from: null,
      salary_period_to: null,
    });
    setErrors({});
  };

  const handleInputChange = (field: keyof CreateProjectData, value: string | number | null) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({
        ...prev,
        [field]: ""
      }));
    }
  };

  return (
    <Dialog open={modalOpen} onOpenChange={(open) => {
      if (onClose && !open) {
        onClose();
      } else if (!onClose) {
        setInternalOpen(open);
      }
    }}>
      <DialogContent className="max-w-2xl">
        <DialogClose asChild>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-3 right-3 h-11 w-11 rounded-xl sm:top-4 sm:right-4 sm:h-8 sm:w-8 hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Đóng</span>
          </Button>
        </DialogClose>
        <DialogHeader>
          <DialogTitle>Tạo dự án mới</DialogTitle>
          <DialogDescription>
            Nhập thông tin chi tiết cho dự án mới
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-4 py-4">
          <div className="col-span-2 space-y-2">
            <Label htmlFor="projectName">Tên dự án *</Label>
            <Input
              id="projectName"
              placeholder=""
              value={formData.name}
              onChange={(e) => handleInputChange("name", e.target.value)}
              className={errors.name ? "border-red-500" : ""}
            />
            {errors.name && <p className="typography-body-medium text-red-500">{errors.name}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="client">Khách hàng *</Label>
            <Input
              id="client"
              placeholder=""
              value={formData.client_name}
              onChange={(e) => handleInputChange("client_name", e.target.value)}
              className={errors.client_name ? "border-red-500" : ""}
              required
            />
            {errors.client_name && <p className="typography-body-medium text-red-500">{errors.client_name}</p>}
          </div>
          <div className="space-y-2">
            <Label htmlFor="code">Mã dự án</Label>
            <Input
              id="code"
              placeholder="Để trống để tự động tạo"
              value={formData.code}
              onChange={(e) => handleInputChange("code", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="startDate">Ngày bắt đầu</Label>
            <Input
              id="startDate"
              type="date"
              value={formData.start_date}
              onChange={(e) => handleInputChange("start_date", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="endDate">Ngày kết thúc</Label>
            <Input
              id="endDate"
              type="date"
              value={formData.end_date}
              onChange={(e) => handleInputChange("end_date", e.target.value)}
              className={errors.end_date ? "border-red-500" : ""}
            />
            {errors.end_date && <p className="typography-body-medium text-red-500">{errors.end_date}</p>}
          </div>
          <div className="col-span-2 space-y-2">
            <Label htmlFor="description">Mô tả</Label>
            <Textarea
              id="description"
              placeholder="Mô tả chi tiết về dự án..."
              value={formData.description}
              onChange={(e) => handleInputChange("description", e.target.value)}
            />
          </div>

          <div className="col-span-2">
            <SalaryPeriodFields
              salaryPeriodFrom={formData.salary_period_from}
              salaryPeriodTo={formData.salary_period_to}
              onSalaryPeriodFromChange={(value) => handleInputChange("salary_period_from", value)}
              onSalaryPeriodToChange={(value) => handleInputChange("salary_period_to", value)}
              variant="detailed"
            />
          </div>
        </div>

        <div className="flex justify-end space-x-2">
          <Button variant="outline" onClick={() => {
            if (onClose) {
              onClose();
            } else {
              setInternalOpen(false);
            }
          }}>
            Đóng
          </Button>
          <Button onClick={handleSubmit} variant="default">
            Tạo dự án
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
