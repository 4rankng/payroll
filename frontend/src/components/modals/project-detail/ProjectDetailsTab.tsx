import { Card, CardContent } from '@/components/ui/card';
import { TabsContent } from '@/components/ui/tabs';
import { Project } from '@/types/api/project.types';
import { Calendar, CalendarRange, FileText } from 'lucide-react';
import {
  ProjectDescriptionTextarea,
  ProjectDateFields,
  ProjectSalaryPeriodFields
} from './ProjectFormFields';

interface ProjectDetailsTabProps {
  formData: Partial<Project>;
  setFormData: (data: Partial<Project>) => void;
  errors: Record<string, string>;
  isEditing: boolean;
}

export function ProjectDetailsTab({
  formData,
  setFormData,
  errors,
  isEditing
}: ProjectDetailsTabProps) {
  return (
    <TabsContent value="details" className="space-y-6">
      <Card>
        <CardContent className="pt-4">
          <div className="flex items-center gap-1.5 mb-3">
            <FileText className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm font-medium">Mô tả dự án</span>
          </div>
          <ProjectDescriptionTextarea 
            formData={formData} 
            setFormData={setFormData} 
            isEditing={isEditing} 
          />
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-4 space-y-4">
          <div className="flex items-center gap-1.5 mb-2">
            <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm font-medium">Thời gian thực hiện</span>
          </div>
          <ProjectDateFields 
            formData={formData} 
            setFormData={setFormData} 
            errors={errors} 
            isEditing={isEditing} 
          />

          {!isEditing && formData.start_date && formData.end_date && (
            <div className="pt-4 border-t">
              <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">Thống kê thời gian</h4>
              <div className="grid grid-cols-3 gap-4 typography-body-medium">
                <div>
                  <span className="text-muted-foreground">Tổng thời gian:</span>
                  <p className="font-medium">
                    {(() => {
                      try {
                        const startDate = new Date(formData.start_date!);
                        const endDate = new Date(formData.end_date!);
                        const diffTime = endDate.getTime() - startDate.getTime();
                        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                        return `${diffDays} ngày`;
                      } catch {
                        return 'Chưa xác định';
                      }
                    })()}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Đã trôi qua:</span>
                  <p className="font-medium">
                    {(() => {
                      try {
                        const startDate = new Date(formData.start_date!);
                        const today = new Date();
                        const diffTime = Math.max(0, today.getTime() - startDate.getTime());
                        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                        return `${diffDays} ngày`;
                      } catch {
                        return 'Chưa xác định';
                      }
                    })()}
                  </p>
                </div>
                <div>
                  <span className="text-muted-foreground">Còn lại:</span>
                  <p className="font-medium">
                    {(() => {
                      try {
                        const endDate = new Date(formData.end_date!);
                        const today = new Date();
                        const diffTime = endDate.getTime() - today.getTime();
                        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
                        return diffDays > 0 ? `${diffDays} ngày` : 'Đã quá hạn';
                      } catch {
                        return 'Chưa xác định';
                      }
                    })()}
                  </p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-4 space-y-4">
          <div className="flex items-center gap-1.5 mb-2">
            <CalendarRange className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm font-medium">Kỳ lương tháng</span>
          </div>
          <ProjectSalaryPeriodFields
            formData={formData}
            setFormData={setFormData}
            isEditing={isEditing}
          />

          {!isEditing && (formData.salary_period_from || formData.salary_period_to) && (
            <div className="pt-4 border-t">
              <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">Thông tin kỳ lương</h4>
              <div className="typography-body-medium text-muted-foreground">
                <p>
                  Kỳ lương: Từ ngày{' '}
                  <span className="font-medium text-foreground">
                    {formData.salary_period_from || '01'}
                  </span>{' '}
                  tháng trước đến ngày{' '}
                  <span className="font-medium text-foreground">
                    {formData.salary_period_to || 'cuối tháng'}
                  </span>{' '}
                  tháng hiện tại
                </p>
                <p className="mt-2 text-sm">
                  * Áp dụng cho nhân viên có chu kỳ trả lương theo tháng
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}