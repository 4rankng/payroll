import { format } from "date-fns";
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Calendar, Clock, AlertCircle, X } from 'lucide-react';
import { ProjectEmployeeAssignment } from '@/types/api/project-employee.types';
import { useChangePaymentSchedule, useCancelScheduleChange } from '@/hooks/api/useProjectEmployees';
import { VIETNAMESE_ASSIGNMENT_LABELS } from '@/types/api/project-employee.types';

interface PaymentScheduleToggleProps {
  assignment: ProjectEmployeeAssignment;
  disabled?: boolean;
}

export function PaymentScheduleToggle({ assignment, disabled }: PaymentScheduleToggleProps) {
  const [showDialog, setShowDialog] = useState(false);
  const [selectedSchedule, setSelectedSchedule] = useState<'weekly' | 'monthly' | 'flexible'>(
    assignment.payment_schedule || 'weekly'
  );

  const changeScheduleMutation = useChangePaymentSchedule();
  const cancelScheduleMutation = useCancelScheduleChange();

  const currentSchedule: 'weekly' | 'monthly' | 'flexible' = assignment.payment_schedule || 'weekly';
  const hasPendingChange = !!assignment.pending_payment_schedule;
  const effectiveFrom = assignment.schedule_effective_from;

  const handleChangeSchedule = async () => {
    if (selectedSchedule === currentSchedule) {
      setShowDialog(false);
      return;
    }

    await changeScheduleMutation.mutateAsync({
      assignmentId: assignment.id,
      data: {
        new_schedule: selectedSchedule,
      },
    });

    setShowDialog(false);
  };

  const handleCancelPendingChange = async () => {
    await cancelScheduleMutation.mutateAsync({
      assignmentId: assignment.id,
    });
  };

  const getScheduleLabel = (schedule: 'weekly' | 'monthly' | 'flexible') => {
    return schedule === 'flexible' ? 'Linh hoạt' : VIETNAMESE_ASSIGNMENT_LABELS.payment_schedule[schedule];
  };

  const getScheduleBadgeVariant = (schedule: 'weekly' | 'monthly' | 'flexible') => {
    return schedule === 'monthly' ? 'default' : 'secondary';
  };

  return (
    <div className="flex items-center gap-2">
      <Badge
        variant={getScheduleBadgeVariant(currentSchedule)}
        className="cursor-pointer"
        onClick={() => !disabled && setShowDialog(true)}
      >
        {currentSchedule === 'weekly' ? (
          <Clock className="h-3 w-3 mr-1" />
        ) : (
          <Calendar className="h-3 w-3 mr-1" />
        )}
        {getScheduleLabel(currentSchedule)}
      </Badge>

      {hasPendingChange && (
        <div className="flex items-center gap-1">
          <Badge variant="outline" className="text-orange-700 border-orange-600">
            <AlertCircle className="h-3 w-3 mr-1" />
            Chờ: {getScheduleLabel(assignment.pending_payment_schedule!)}
          </Badge>
          {effectiveFrom && (
            <span className="typography-body-small text-muted-foreground">
              từ {format(new Date(effectiveFrom), 'dd/MM/yyyy')}
            </span>
          )}
          <Button
            variant="ghost"
            size="sm"
            className="h-6 w-6 p-0"
            onClick={handleCancelPendingChange}
            disabled={cancelScheduleMutation.isPending}
          >
            <X className="h-3 w-3" />
            <span className="sr-only">Hủy thay đổi</span>
          </Button>
        </div>
      )}

      <Dialog open={showDialog} onOpenChange={setShowDialog}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>Thay đổi chu kỳ thanh toán</DialogTitle>
            <DialogDescription>
              Chọn chu kỳ thanh toán mới cho{' '}
              <strong>{assignment.employee_name || assignment.employee_code}</strong>
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Chu kỳ thanh toán hiện tại</Label>
              <div className="flex items-center gap-2">
                <Badge variant={getScheduleBadgeVariant(currentSchedule)}>
                  {currentSchedule === 'weekly' ? (
                    <Clock className="h-3 w-3 mr-1" />
                  ) : (
                    <Calendar className="h-3 w-3 mr-1" />
                  )}
                  {getScheduleLabel(currentSchedule)}
                </Badge>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="new-schedule">Chu kỳ thanh toán mới</Label>
              <Select value={selectedSchedule} onValueChange={(value) => setSelectedSchedule(value as 'weekly' | 'monthly' | 'flexible')}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="weekly">
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4" />
                      {getScheduleLabel('weekly')}
                    </div>
                  </SelectItem>
                  <SelectItem value="monthly">
                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4" />
                      {getScheduleLabel('monthly')}
                    </div>
                  </SelectItem>
                  <SelectItem value="flexible">
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4" />
                      {getScheduleLabel('flexible')}
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="rounded-xl bg-muted p-3 typography-body-small">
              <p className="font-medium mb-1">Lưu ý:</p>
              <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                <li>Nếu tháng hiện tại chưa có bảng công đã trả, thay đổi có hiệu lực ngay</li>
                <li>Nếu đã có bảng công đã trả, thay đổi có hiệu lực từ đầu tháng sau</li>
              </ul>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowDialog(false)}
              disabled={changeScheduleMutation.isPending}
            >
              Đóng
            </Button>
            <Button
              type="button"
              onClick={handleChangeSchedule}
              disabled={changeScheduleMutation.isPending || selectedSchedule === currentSchedule}
            >
              {changeScheduleMutation.isPending ? 'Đang cập nhật...' : 'Cập nhật'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
