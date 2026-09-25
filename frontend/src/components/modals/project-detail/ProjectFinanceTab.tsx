import { Card, CardContent } from '@/components/ui/card';
import { TabsContent } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { Project } from '@/types/api/project.types';
import { DollarSign } from 'lucide-react';

interface ProjectFinanceTabProps {
  formData: Partial<Project>;
  setFormData: (data: Partial<Project>) => void;
  errors: Record<string, string>;
  isEditing: boolean;
  formatCurrency: (amount: number) => string;
  calculateBudgetUsage: () => number;
}

export function ProjectFinanceTab({
  formData,
  setFormData,
  errors,
  isEditing,
  formatCurrency,
  calculateBudgetUsage
}: ProjectFinanceTabProps) {
  return (
    <TabsContent value="finance" className="space-y-6">
      <Card>
        <CardContent className="pt-4 space-y-6">
          <div className="flex items-center gap-1.5">
            <DollarSign className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm font-medium">Thông tin tài chính</span>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="budget">Ngân sách *</Label>
              <Input
                id="budget"
                type="number"
                value={formData.budget || ''}
                onChange={(e) => setFormData({ ...formData, budget: parseInt(e.target.value) || 0 })}
                disabled={!isEditing}
                placeholder="100000000"
                className={errors.budget ? 'border-red-500' : ''}
              />
              {errors.budget && <p className="typography-body-medium text-red-600 mt-1">{errors.budget}</p>}
              {!isEditing && formData.budget && (
                <p className="typography-body-medium text-muted-foreground mt-1">
                  {formatCurrency(formData.budget)}
                </p>
              )}
            </div>

            <div>
              <Label htmlFor="total_payout_vnd">Đã chi tiêu</Label>
              <Input
                id="total_payout_vnd"
                type="number"
                value={formData.total_payout_vnd || ''}
                onChange={(e) => setFormData({ ...formData, total_payout_vnd: parseInt(e.target.value) || 0 })}
                disabled={!isEditing}
              />
              {!isEditing && formData.total_payout_vnd && (
                <p className="typography-body-medium text-muted-foreground mt-1">
                  {formatCurrency(formData.total_payout_vnd)}
                </p>
              )}
            </div>
          </div>

          {!isEditing && formData.budget && formData.total_payout_vnd !== undefined && (
            <div className="space-y-4">
              <Separator />
              <div>
                <div className="flex justify-between items-center mb-2">
                  <span className="typography-body-medium">Sử dụng ngân sách</span>
                  <span className="typography-body-medium text-muted-foreground">
                    {Math.round(calculateBudgetUsage())}%
                  </span>
                </div>
                <Progress value={calculateBudgetUsage()} className="w-full" />
              </div>
              
              <div className="grid grid-cols-1 gap-3 typography-body-medium sm:grid-cols-3 sm:gap-4">
                <div className="text-center">
                  <p className="typography-title-large text-blue-600">
                    {formatCurrency(formData.budget)}
                  </p>
                  <p className="text-muted-foreground">Tổng ngân sách</p>
                </div>
                <div className="text-center">
                  <p className="typography-title-large text-red-600">
                    {formatCurrency(formData.total_payout_vnd || 0)}
                  </p>
                  <p className="text-muted-foreground">Đã chi tiêu</p>
                </div>
                <div className="text-center">
                  <p className="typography-title-large text-green-700">
                    {formatCurrency((formData.budget || 0) - (formData.total_payout_vnd || 0))}
                  </p>
                  <p className="text-muted-foreground">Còn lại</p>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}
