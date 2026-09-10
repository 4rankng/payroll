import { format } from "date-fns";
import {
  Calendar,
  Phone,
  MapPin,
  Hash,
  Clock,
} from "lucide-react";
import { toast } from "@/components/ui/sonner";
import { useUpdateEmployeeProjectAssignment } from "@/hooks/api/useProjectEmployees";
import { getErrorMessage } from "@/utils/error-handler";
import type { EmployeeDetailsProps } from "../types";
import { EmployeeProjectSection } from "./EmployeeProjectSection";
import { EmployeeBankInfo } from "./EmployeeBankInfo";

export function EmployeeViewDetails({ employee }: EmployeeDetailsProps) {
  const updateAssignment = useUpdateEmployeeProjectAssignment();

  const handleApplyProjectChanges = async (projectId: number, changes) => {
    if (!employee?.id) {
      return;
    }
    const data: {
      project_id: number;
      assignment_id?: number;
      position?: string;
      start_date?: string;
      end_date?: string;
    } = { project_id: projectId };
    if (changes.assignmentId) {
      data.assignment_id = changes.assignmentId;
    }
    if (changes.position !== undefined) {
      data.position = changes.position;
    }
    if (changes.start_date !== undefined) {
      data.start_date = changes.start_date;
    }
    // An empty string clears the end date (assignment stays open-ended).
    if (changes.end_date !== undefined) {
      data.end_date = changes.end_date;
    }
    try {
      await updateAssignment.mutateAsync({ employeeId: employee.id, data });
      toast({
        title: "Thành công",
        description: "Đã cập nhật phân công dự án.",
      });
    } catch (error) {
      toast({
        title: "Lỗi",
        description: getErrorMessage(error),
        variant: "destructive",
      });
    }
  };
  return (
    <div className="space-y-6 p-1">
      {/* Personal Information */}
      <div className="space-y-4">
        <h3 className="typography-title-large">Thông tin cá nhân</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              <Phone className="w-4 h-4 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <dt className="typography-body-medium typography-label-medium text-muted-foreground">Số điện thoại</dt>
                <dd className="typography-body-medium">{employee.mobile || '-'}</dd>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <Calendar className="w-4 h-4 text-muted-foreground mt-0.5" />
              <div className="min-w-0 flex-1">
                <dt className="typography-body-medium typography-label-medium text-muted-foreground">Ngày sinh</dt>
                <dd className="typography-body-medium">
                  {employee.date_of_birth ?
                    format(new Date(employee.date_of_birth), 'dd/MM/yyyy') :
                    '-'
                  }
                </dd>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <Hash className="w-4 h-4 text-muted-foreground mt-0.5" />
              <div className="min-w-0 flex-1">
                <dt className="typography-body-medium typography-label-medium text-muted-foreground">CCCD</dt>
                <dd className="typography-body-medium font-mono bg-muted px-2 py-1 rounded">{employee.cccd}</dd>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <Clock className="w-4 h-4 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <dt className="typography-body-medium typography-label-medium text-muted-foreground">Ngày tham gia</dt>
                <dd className="typography-body-medium">
                  {employee.created_at ?
                    format(new Date(employee.created_at), 'dd/MM/yyyy') :
                    '-'
                  }
                </dd>
              </div>
            </div>
          </div>
        </div>

        <EmployeeProjectSection
          employee={employee}
          onProjectApply={handleApplyProjectChanges}
        />
      </div>

      {/* Address and Bank Info */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
        {/* Address */}
        <div className="space-y-4">
          <h3 className="typography-title-large">Địa chỉ</h3>
          <div className="bg-muted/50 rounded-xl p-3">
            <div className="flex items-start gap-2">
              <MapPin className="w-4 h-4 text-muted-foreground mt-0.5" />
              <p className="typography-body-medium leading-relaxed flex-1">{employee.address || '-'}</p>
            </div>
          </div>
        </div>

        {/* Bank Info */}
        <div className="space-y-4">
          <h3 className="typography-title-large">Thông tin ngân hàng</h3>
          <EmployeeBankInfo employee={employee} />
        </div>
      </div>
    </div>
  );
}
