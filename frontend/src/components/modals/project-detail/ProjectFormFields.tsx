import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Project } from '@/types/api/project.types';
import { getVietnameseProjectStatus } from '@/utils/vietnamese';
import { SalaryPeriodFields } from '@/components/projects/SalaryPeriodFields';
import {
  Clock,
  TrendingUp,
  AlertCircle,
  CheckCircle,
  XCircle,
  Pause
} from 'lucide-react';

interface ProjectFormFieldsProps {
  formData: Partial<Project>;
  setFormData: (data: Partial<Project>) => void;
  errors: Record<string, string>;
  isEditing: boolean;
}

const projectStatuses = [
  { value: 'draft', label: getVietnameseProjectStatus('draft'), color: 'bg-muted/500', icon: Clock },
  { value: 'active', label: getVietnameseProjectStatus('active'), color: 'bg-green-500', icon: TrendingUp },
  { value: 'paused', label: getVietnameseProjectStatus('paused'), color: 'bg-orange-500', icon: Pause },
  { value: 'completed', label: getVietnameseProjectStatus('completed'), color: 'bg-blue-600', icon: CheckCircle },
  { value: 'cancelled', label: getVietnameseProjectStatus('cancelled'), color: 'bg-red-500', icon: XCircle }
];

export function ProjectNameField({ formData, setFormData, errors, isEditing }: ProjectFormFieldsProps) {
  return (
    <div>
      <Label htmlFor="name">Tên dự án *</Label>
      <Input
        id="name"
        value={formData.name || ''}
        onChange={(e) => setFormData({ ...formData, name: e.target.value })}
        disabled={!isEditing}
        className={errors.name ? 'border-red-500' : ''}
      />
      {errors.name && <p className="typography-body-medium text-red-500 mt-1">{errors.name}</p>}
    </div>
  );
}

export function ProjectClientField({ formData, setFormData, errors, isEditing }: ProjectFormFieldsProps) {
  return (
    <div>
      <Label htmlFor="client_name">Khách hàng *</Label>
      <Input
        id="client_name"
        value={formData.client_name || ''}
        onChange={(e) => setFormData({ ...formData, client_name: e.target.value })}
        disabled={!isEditing}
        className={errors.client_name ? 'border-red-500' : ''}
      />
      {errors.client_name && <p className="typography-body-medium text-red-500 mt-1">{errors.client_name}</p>}
    </div>
  );
}

export function ProjectDescriptionField({ formData, setFormData, isEditing }: Omit<ProjectFormFieldsProps, 'errors'>) {
  return (
    <div>
      <Label htmlFor="description">Mô tả dự án</Label>
      <Input
        id="description"
        value={formData.description || ''}
        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        disabled={!isEditing}
      />
    </div>
  );
}

export function ProjectStatusField({ formData, setFormData, isEditing }: Omit<ProjectFormFieldsProps, 'errors'>) {
  return (
    <div>
      <Label htmlFor="status">Trạng thái</Label>
      <Select
        value={formData.status}
        onValueChange={(value) => setFormData({ ...formData, status: value as Project['status'] })}
        disabled={!isEditing}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {projectStatuses.map((status) => (
            <SelectItem key={status.value} value={status.value}>
              <div className="flex items-center gap-2">
                <status.icon className="h-4 w-4" />
                {status.label}
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function ProjectEmployeeCountField({ formData, setFormData, errors, isEditing }: ProjectFormFieldsProps) {
  return (
    <div>
      <Label htmlFor="employee_count">Số nhân viên</Label>
      <Input
        id="employee_count"
        type="number"
        min="1"
        value={formData.employee_count || 1}
        onChange={(e) => setFormData({ ...formData, employee_count: parseInt(e.target.value) || 1 })}
        disabled={!isEditing}
        className={errors.employee_count ? 'border-red-500' : ''}
      />
      {errors.employee_count && <p className="typography-body-medium text-red-500 mt-1">{errors.employee_count}</p>}
    </div>
  );
}

export function ProjectBudgetField({ formData, setFormData, errors, isEditing }: ProjectFormFieldsProps) {
  return (
    <div>
      <Label htmlFor="budget">Ngân sách</Label>
      <Input
        id="budget"
        type="number"
        value={formData.budget || ''}
        onChange={(e) => setFormData({ ...formData, budget: parseInt(e.target.value) || 0 })}
        disabled={!isEditing}
        placeholder="100000000"
        className={errors.budget ? 'border-red-500' : ''}
      />
      {errors.budget && <p className="typography-body-medium text-red-500 mt-1">{errors.budget}</p>}
    </div>
  );
}

export function ProjectDateFields({ formData, setFormData, errors, isEditing }: ProjectFormFieldsProps) {
  return (
    <div className="grid grid-cols-2 gap-4">
      <div>
        <Label htmlFor="start_date">Ngày bắt đầu</Label>
        <Input
          id="start_date"
          type="date"
          value={formData.start_date || ''}
          onChange={(e) => {
            setFormData({
              ...formData,
              start_date: e.target.value
            });
          }}
          disabled={!isEditing}
        />
      </div>

      <div>
        <Label htmlFor="end_date">Ngày kết thúc</Label>
        <Input
          id="end_date"
          type="date"
          value={formData.end_date || ''}
          onChange={(e) => {
            setFormData({
              ...formData,
              end_date: e.target.value
            });
          }}
          disabled={!isEditing}
          className={errors.end_date ? 'border-red-500' : ''}
        />
        {errors.end_date && <p className="typography-body-medium text-red-500 mt-1">{errors.end_date}</p>}
      </div>
    </div>
  );
}

export function ProjectDescriptionTextarea({ formData, setFormData, isEditing }: Omit<ProjectFormFieldsProps, 'errors'>) {
  return (
    <div>
      <Label htmlFor="description">Mô tả dự án</Label>
      <Textarea
        id="description"
        value={formData.description || ''}
        onChange={(e) => setFormData({ ...formData, description: e.target.value })}
        disabled={!isEditing}
        rows={4}
        placeholder="Mô tả chi tiết về dự án..."
      />
    </div>
  );
}

export function ProjectSalaryPeriodFields({ formData, setFormData, isEditing }: Omit<ProjectFormFieldsProps, 'errors'>) {
  const handleSalaryPeriodFromChange = (value: number | null) => {
    setFormData({
      ...formData,
      salary_period_from: value
    });
  };

  const handleSalaryPeriodToChange = (value: number | null) => {
    setFormData({
      ...formData,
      salary_period_to: value
    });
  };

  return (
    <SalaryPeriodFields
      salaryPeriodFrom={formData.salary_period_from}
      salaryPeriodTo={formData.salary_period_to}
      onSalaryPeriodFromChange={handleSalaryPeriodFromChange}
      onSalaryPeriodToChange={handleSalaryPeriodToChange}
      disabled={!isEditing}
    />
  );
}
