import { Button } from "@/components/ui/button";
import { ArrowRightLeft, FileUp, FileText, History, Mail, Plus } from "lucide-react";

interface TimesheetActionButtonsProps {
  onAddTimesheet?: () => void;
  onApprovedTimesheetsExport?: () => void;
  onPayrollReportEmailSend?: () => void;
  onBulkTransferExport?: () => void;
  onBulkTransferResultUpload?: () => void;
  onBulkTransferHistory?: () => void;
  onPaymentHistory?: () => void;
  isApprovedExportPending?: boolean;
  isPayrollReportEmailPending?: boolean;
}

export const TimesheetActionButtons = ({
  onAddTimesheet,
  onApprovedTimesheetsExport,
  onPayrollReportEmailSend,
  onBulkTransferExport,
  onBulkTransferResultUpload,
  onBulkTransferHistory,
  onPaymentHistory,
  isApprovedExportPending = false,
  isPayrollReportEmailPending = false,
}: TimesheetActionButtonsProps) => {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {onAddTimesheet && (
        <Button onClick={onAddTimesheet}>
          <Plus />
          Nhập công
        </Button>
      )}

      {onApprovedTimesheetsExport && (
        <Button
          variant="ghost"
          onClick={onApprovedTimesheetsExport}
          disabled={isApprovedExportPending}
        >
          <FileText />
          {isApprovedExportPending ? 'Đang xuất...' : 'Xuất bảng công'}
        </Button>
      )}

      {onPaymentHistory && (
        <Button variant="ghost" onClick={onPaymentHistory}>
          <History />
          Lịch sử trả lương
        </Button>
      )}

      {onBulkTransferExport && (
        <Button variant="ghost" onClick={onBulkTransferExport}>
          <ArrowRightLeft />
          Xuất chuyển lô
        </Button>
      )}

      {onBulkTransferResultUpload && (
        <Button variant="ghost" onClick={onBulkTransferResultUpload}>
          <FileUp />
          Nhập KQ chuyển lô
        </Button>
      )}

      {onBulkTransferHistory && (
        <Button variant="ghost" onClick={onBulkTransferHistory}>
          <History />
          Lịch sử chuyển lô
        </Button>
      )}

      {onPayrollReportEmailSend && (
        <Button
          variant="ghost"
          onClick={onPayrollReportEmailSend}
          disabled={isPayrollReportEmailPending}
        >
          <Mail />
          {isPayrollReportEmailPending ? 'Đang gửi...' : 'Email thanh toán'}
        </Button>
      )}
    </div>
  );
};
