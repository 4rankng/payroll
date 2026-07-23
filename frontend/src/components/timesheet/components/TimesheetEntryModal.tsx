import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import {
  X, Calendar, Clock, Trash2, Check, XCircle, RotateCcw, FileEdit,
  AlertTriangle, Info, Lock, Ban, User, Building2,
} from "lucide-react";
import { Timesheet } from "@/types/api/timesheet.types";
import { InlineAlert } from "@/components/shared/InlineAlert";
import { ContextStrip } from "@/components/shared/ContextStrip";
import { useTimesheetEntryForm } from "./useTimesheetEntryForm";

interface TimesheetEntryModalProps {
  isOpen: boolean;
  onClose: () => void;
  projectId: number;
  projectName: string;
  employeeId: number;
  employeeName: string;
  date: Date;
  existingEntry?: Timesheet | null;
  onSuccess?: () => void;
  onDelete?: (timesheet: Timesheet) => void;
  canDelete?: (timesheet: Timesheet) => boolean;
  onRequestEdit?: (timesheet: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void;
  onRequestEditSuccess?: () => void;
  requestingTimesheetId?: number | null;
}

export function TimesheetEntryModal(props: TimesheetEntryModalProps) {
  const {
    isOpen, onClose, existingEntry, date, canDelete,
  } = props;

  const f = useTimesheetEntryForm(props);

  return (
    <Dialog open={isOpen} onOpenChange={f.isLoading ? undefined : onClose}>
      <DialogContent className="sm:max-w-[480px] gap-0 overflow-hidden rounded-2xl shadow-2xl" contentPadding="none" hideCloseButton>
        {/* Header */}
        <div className="bg-emerald-950 px-5 pt-5 pb-4 text-white flex-shrink-0">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-lg font-semibold tracking-tight text-white leading-tight">
                {f.isReadOnly ? "Chi tiết bảng công" : f.isEditing ? "Chỉnh sửa bảng công" : "Nhập bảng công"}
              </p>
              <div className="flex items-center gap-1.5 mt-1">
                <Calendar className="h-3 w-3 text-emerald-200/75" />
                <p className="text-xs text-emerald-200/75">{format(date, "EEEE, dd/MM/yyyy", { locale: vi })}</p>
              </div>
            </div>
            <div className="flex items-center gap-1.5 shrink-0">
              {f.statusBadge && (
                <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${f.statusBadge.cls}`}>
                  {f.statusBadge.label}
                </span>
              )}
              {existingEntry?.force_payroll && <Badge variant="info" className="text-xs">Kỳ tiếp theo</Badge>}
              <button onClick={onClose} disabled={f.isLoading}
                className="w-8 h-8 rounded-full flex items-center justify-center bg-white/10 hover:bg-white/20 transition-colors disabled:opacity-40" aria-label="Đóng">
                <X className="h-4 w-4 text-white" />
              </button>
            </div>
          </div>
        </div>

        <div className="p-5 space-y-3 max-h-[calc(90vh-80px)] overflow-y-auto">
          {/* Context strip */}
          <ContextStrip items={[
            { label: "Dự án", value: f.displayProjectName, icon: Building2, iconBg: "bg-blue-100", iconColor: "text-blue-600" },
            { label: "Nhân viên", value: f.displayEmployeeName, icon: User, iconBg: "bg-slate-100", iconColor: "text-slate-500", subtitle: f.displayEmployeeCode || undefined },
          ]} />

          {/* Inline alerts */}
          {existingEntry?.payment_status === "paid" && (
            <InlineAlert severity="error" icon={Lock} message="Đã thanh toán — không thể chỉnh sửa." />
          )}
          {existingEntry?.status === "approved" && !f.isReadOnly && f.user?.role === "admin" && (
            <InlineAlert severity="warning" icon={AlertTriangle} message="Bản ghi đã duyệt — hãy cẩn thận khi chỉnh sửa." />
          )}
          {!f.dateValidation.isValid && (
            <InlineAlert severity="error" icon={AlertTriangle} message={f.dateValidation.errorMessage ?? ""} />
          )}
          {f.isPayrateError && (
            <InlineAlert severity="error" icon={AlertTriangle} message="Không thể tải cấu hình bảng lương. Vui lòng thử lại." />
          )}
          {!f.isPayrateLoading && !f.isPayrateError && f.availableHourTypes.length === 0 && (
            <InlineAlert severity="warning" icon={AlertTriangle} message="Dự án chưa có cấu hình loại giờ." />
          )}
          {existingEntry?.request_edit_id != null && (
            <InlineAlert severity="info" icon={Info} message="Yêu cầu chỉnh sửa đang chờ duyệt."
              action={f.user?.role === "admin" ? (
                <Button type="button" variant="success" size="sm" onClick={f.handleApproveEditRequest} disabled={f.isLoading} className="h-7 text-xs px-2.5 shrink-0">
                  {f.approveEditRequestMutation.isPending ? <div className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <><Check className="w-3 h-3 mr-1" />Duyệt</>}
                </Button>
              ) : undefined}
            />
          )}

          {/* Form fields */}
          <div className="space-y-3">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-bold text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
                  <Clock className="h-3.5 w-3.5 text-blue-500" />Thời gian làm việc
                </span>
                {existingEntry && existingEntry.status === "pending_approval" && (
                  <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 text-xs font-semibold px-2 py-0.5 rounded-full">
                    <Check className="h-2.5 w-2.5" />Đang làm
                  </span>
                )}
              </div>

              <div className="flex items-center gap-4 bg-slate-50 p-4 rounded-xl border border-dashed border-slate-300">
                <div className="flex-1">
                  <div className="relative">
                    <Input id="hoursWorked" type="number" min="0" max="24" step="0.5"
                      value={f.formData.hoursWorked} onChange={f.handleHoursChange} disabled={f.isReadOnly}
                      className="w-full border-2 border-slate-200 rounded-lg py-3 px-4 text-2xl font-bold text-slate-900 focus:border-blue-500 tabular-nums pr-14 h-auto" />
                    <span className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 font-medium text-sm pointer-events-none">giờ</span>
                  </div>
                </div>
                <div className="text-right shrink-0">
                  {f.isPayrateLoading ? (
                    <div className="flex flex-col items-end gap-1">
                      <div className="h-6 w-24 bg-slate-200 rounded animate-pulse" />
                      <div className="h-3 w-16 bg-slate-200 rounded animate-pulse" />
                    </div>
                  ) : f.displayAmount != null ? (
                    <>
                      <div className="text-[10px] text-slate-500 font-medium mb-0.5">Tạm tính:</div>
                      <div className="text-xl font-bold text-blue-600 tabular-nums leading-none">
                        {f.displayAmount.toLocaleString("vi-VN")}đ
                      </div>
                      {f.displayPayrate != null && (
                        <div className="text-[10px] text-slate-400 mt-0.5">Đơn giá: {f.displayPayrate.toLocaleString("vi-VN")}đ/h</div>
                      )}
                    </>
                  ) : null}
                </div>
              </div>
            </div>

            {/* Hour type + Day type */}
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-xs font-bold text-slate-500 uppercase tracking-wider px-0.5">Loại ca</Label>
                <Select value={f.formData.hourType} onValueChange={(v) => f.handleFormChange("hourType", v)}
                  disabled={f.isReadOnly || f.isPayrateLoading || f.isPayrateError}>
                  <SelectTrigger className="h-10 bg-slate-50 border-slate-200 focus:border-blue-500 focus:ring-blue-500/20">
                    {f.isPayrateLoading ? (
                      <span className="flex items-center gap-1.5 text-muted-foreground text-sm">
                        <div className="w-3 h-3 border-2 border-muted-foreground border-t-transparent rounded-full animate-spin" />Đang tải...
                      </span>
                    ) : f.isPayrateError ? (
                      <span className="text-destructive text-sm">Lỗi</span>
                    ) : <SelectValue placeholder="Chọn loại ca" />}
                  </SelectTrigger>
                  <SelectContent>
                    {f.availableHourTypes.map((ht) => <SelectItem key={ht} value={ht}>{ht}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-bold text-slate-500 uppercase tracking-wider px-0.5">Loại ngày</Label>
                <Select value={f.formData.dayType} onValueChange={(v) => f.handleFormChange("dayType", v as "Ngày thường" | "Ngày nghỉ" | "Ngày lễ")}
                  disabled={f.isReadOnly}>
                  <SelectTrigger className="h-10 bg-slate-50 border-slate-200 focus:border-blue-500 focus:ring-blue-500/20">
                    <SelectValue placeholder="Chọn loại ngày" />
                  </SelectTrigger>
                  <SelectContent>
                    {f.availableDayTypes.map((dt) => <SelectItem key={dt} value={dt}>{dt}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="border-t border-slate-100 pt-4 -mx-5 px-5">
            {f.isReadOnly ? (
              <div className="space-y-2">
                {f.requestEditError && <p className="text-xs text-destructive" role="alert">{f.requestEditError}</p>}
                <div className="flex items-center justify-between">
                  <div>
                    {f.user?.role === "partner" && existingEntry?.request_edit_id != null && (
                      <button type="button" onClick={f.handleCancelEditRequest} disabled={f.isLoading}
                        className="flex items-center gap-1.5 text-red-500 text-sm font-semibold hover:bg-red-50 px-3 py-2 rounded-lg transition-colors disabled:opacity-40">
                        {f.cancelEditRequestMutation.isPending ? <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <Ban className="w-3.5 h-3.5" />}
                        Hủy yêu cầu
                      </button>
                    )}
                  </div>
                  <div className="flex flex-wrap justify-end gap-2">
                    {f.canRequestEdit && (
                      <Button variant="outline" size="sm" onClick={f.handleRequestEditClick} disabled={f.isRequestEditPending || f.isLoading} className="min-h-11">
                        {f.isRequestEditPending ? <><div className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin mr-1.5" />Đang gửi...</> : <><FileEdit className="w-3.5 h-3.5 mr-1.5" />Yêu cầu chỉnh sửa</>}
                      </Button>
                    )}
                    <Button variant="ghost" size="sm" onClick={onClose} disabled={f.isLoading} className="min-h-11 text-slate-600">Đóng</Button>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-between">
                <div>
                  {existingEntry && props.onDelete && canDelete && canDelete(existingEntry) && (
                    <button type="button" onClick={f.handleDelete} disabled={f.isLoading}
                      className="flex items-center gap-1.5 text-red-500 text-sm font-semibold hover:bg-red-50 px-3 py-2 rounded-lg transition-colors disabled:opacity-40">
                      <Trash2 className="w-3.5 h-3.5" />Xóa công
                    </button>
                  )}
                </div>
                <div className="flex flex-wrap items-center justify-end gap-2">
                  {f.user?.role === "admin" && existingEntry?.status === "pending_approval" && (
                    <Button variant="outline" size="sm" onClick={f.handleReject} disabled={f.isLoading} className="min-h-11 text-slate-600">
                      {f.rejectTimesheetMutation.isPending ? <div className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <><XCircle className="w-3.5 h-3.5 mr-1.5" />Loại</>}
                    </Button>
                  )}
                  {f.user?.role === "admin" && existingEntry &&
                    (existingEntry.status === "approved" || existingEntry.status === "rejected") &&
                    existingEntry.payment_status !== "paid" && (
                    <Button variant="outline" size="sm" onClick={f.handleReset} disabled={f.isLoading} className="min-h-11 text-slate-600">
                      {f.resetTimesheetMutation.isPending ? <div className="w-3 h-3 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <><RotateCcw className="w-3.5 h-3.5 mr-1.5" />Reset</>}
                    </Button>
                  )}
                  <button type="button" onClick={onClose} disabled={f.isLoading}
                    className="min-h-11 rounded-lg px-4 py-2 text-sm font-semibold text-slate-600 transition-colors hover:bg-slate-100 disabled:opacity-40">
                    Bỏ qua
                  </button>
                  <Button className="flex min-h-11 items-center gap-2 bg-primary px-5 text-primary-foreground hover:bg-primary/90"
                    onClick={f.canSaveAndApprove ? f.handleSaveAndApprove : f.handleSave}
                    disabled={f.isLoading || f.isPayrateLoading || f.formData.hoursWorked <= 0 || !f.formData.hourType || !f.isHourTypeValid || f.availableHourTypes.length === 0 || !f.dateValidation.isValid}>
                    {f.isLoading ? <div className="w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <Check className="w-3.5 h-3.5" />}
                    {f.isLoading ? (f.canSaveAndApprove ? "Đang xử lý..." : f.isEditing ? "Đang lưu..." : "Đang tạo...") : (f.canSaveAndApprove ? "Duyệt & Lưu" : f.isEditing ? "Lưu thay đổi" : "Tạo bảng công")}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
