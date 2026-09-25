import { Card, CardContent } from '@/components/ui/card';
import { TabsContent } from '@/components/ui/tabs';
import { Project } from '@/types/api/project.types';
import {
  Users,
  DollarSign,
  Calendar,
  FileText,
  Target
} from 'lucide-react';
import { ProjectStatusCard } from './ProjectStatusCard';
import {
  ProjectNameField,
  ProjectClientField,
  ProjectDescriptionField,
  ProjectStatusField,
  ProjectEmployeeCountField,
  ProjectBudgetField
} from './ProjectFormFields';

interface ProjectOverviewTabProps {
  formData: Partial<Project>;
  setFormData: (data: Partial<Project>) => void;
  errors: Record<string, string>;
  isEditing: boolean;
  project?: Project;
  formatCurrency: (amount: number) => string;
  calculateBudgetUsage: () => number;
}

export function ProjectOverviewTab({
  formData,
  setFormData,
  errors,
  isEditing,
  project,
  formatCurrency,
  calculateBudgetUsage
}: ProjectOverviewTabProps) {
  return (
    <TabsContent value="overview" className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardContent className="pt-4 space-y-4">
            <div className="flex items-center gap-1.5 mb-2">
              <FileText className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-sm font-medium">Thông tin cơ bản</span>
            </div>
            <ProjectNameField
              formData={formData}
              setFormData={setFormData}
              errors={errors}
              isEditing={isEditing}
            />
            <ProjectClientField
              formData={formData}
              setFormData={setFormData}
              errors={errors}
              isEditing={isEditing}
            />
            <ProjectDescriptionField
              formData={formData}
              setFormData={setFormData}
              isEditing={isEditing}
            />
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-4 space-y-4">
            <div className="flex items-center gap-1.5 mb-2">
              <Target className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-sm font-medium">Trạng thái & Cấu hình</span>
            </div>
            <ProjectStatusField
              formData={formData}
              setFormData={setFormData}
              isEditing={isEditing}
            />
            <ProjectEmployeeCountField
              formData={formData}
              setFormData={setFormData}
              errors={errors}
              isEditing={isEditing}
            />
            <ProjectBudgetField
              formData={formData}
              setFormData={setFormData}
              errors={errors}
              isEditing={isEditing}
            />
          </CardContent>
        </Card>
      </div>

      {!isEditing && project && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <ProjectStatusCard
            icon={Users}
            value={formData.employee_count || 0}
            label="Nhân viên"
            color="text-blue-600"
          />

          <ProjectStatusCard
            icon={DollarSign}
            value={`${Math.round(calculateBudgetUsage())}%`}
            label="Ngân sách đã sử dụng"
            color="text-green-700"
          />

          <ProjectStatusCard
            icon={Target}
            value="-"
            label="Tiến độ"
            color="text-emerald-700"
          />

          <ProjectStatusCard
            icon={Calendar}
            value={(() => {
              try {
                if (formData.end_date) {
                  const endDate = new Date(formData.end_date);
                  const today = new Date();
                  const diffTime = endDate.getTime() - today.getTime();
                  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                  return diffDays > 0 ? diffDays : 0;
                }
                return '-';
              } catch {
                return '-';
              }
            })()}
            label="Ngày còn lại"
            color="text-orange-700"
          />
        </div>
      )}
    </TabsContent>
  );
}
