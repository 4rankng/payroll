import { Card, CardContent } from '@/components/ui/card';
import { TabsContent } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Project } from '@/types/api/project.types';
import { ProjectEmployee } from '@/types/api/project.types';
import { Users } from 'lucide-react';
import { ProjectEmployeeCountField } from './ProjectFormFields';

interface ProjectTeamTabProps {
  formData: Partial<Project>;
  setFormData: (data: Partial<Project>) => void;
  errors: Record<string, string>;
  isEditing: boolean;
  employees: ProjectEmployee[];
}

export function ProjectTeamTab({
  formData,
  setFormData,
  errors,
  isEditing,
  employees
}: ProjectTeamTabProps) {
  return (
    <TabsContent value="team" className="space-y-6">
      <Card>
        <CardContent className="pt-4">
          <div className="flex items-center gap-1.5 mb-3">
            <Users className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm font-medium">Đội ngũ dự án</span>
            <Badge variant="secondary" className="text-xs">{employees.length} nhân viên</Badge>
          </div>
          <ProjectEmployeeCountField 
            formData={formData} 
            setFormData={setFormData} 
            errors={errors} 
            isEditing={isEditing} 
          />

          {!isEditing && employees.length > 0 && (
            <div className="mt-6">
              <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-3">Danh sách nhân viên</h4>
              <div className="space-y-3">
                {employees.slice(0, 10).map((projectEmployee) => (
                  <div key={projectEmployee.id} className="flex items-center gap-3 p-3 border rounded-xl">
                    <Avatar className="h-8 w-8">
                      <AvatarImage src={projectEmployee.employee?.avatar} />
                      <AvatarFallback className="bg-blue-100 text-blue-700 typography-body-small">
                        {projectEmployee.employee?.fullname ? 
                          projectEmployee.employee.fullname.split(' ').slice(-2).map(n => n[0]).join('').toUpperCase() :
                          'NV'
                        }
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1">
                      <p className="typography-body-medium">
                        {projectEmployee.employee?.fullname || 'Tên không xác định'}
                      </p>
                      <p className="typography-body-small text-muted-foreground">
                        {projectEmployee.employee?.position || 'Vị trí không xác định'}
                      </p>
                    </div>
                    <Badge variant={projectEmployee.status === 'active' ? 'default' : 'secondary'}>
                      {projectEmployee.status === 'active' ? 'Đang làm việc' : 'Nghỉ việc'}
                    </Badge>
                  </div>
                ))}
                {employees.length > 10 && (
                  <p className="typography-body-medium text-muted-foreground text-center py-2">
                    và {employees.length - 10} nhân viên khác...
                  </p>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}