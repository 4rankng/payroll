import { format } from "date-fns";
import { useState, useEffect } from "react";
import { useSecureModal } from "@/hooks/useSecureModal";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "@/components/ui/sonner";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { timesheetService } from "@/services/api/timesheet.service";
import { Timesheet } from "@/types/api/timesheet.types";
import { X, Clock, User, Calendar, FileText, CheckCircle, XCircle } from "lucide-react";

/**
 * Route-based Approval Details Modal
 * Shows detailed information about an approval item
 */
export function ApprovalDetailsModal() {
  const navigate = useNavigate();
  const { id, type } = useParams<{ id: string; type?: string }>();
  const [approvalData, setApprovalData] = useState<Timesheet | null>(null);
  const [loading, setLoading] = useState(true);
  const [approving, setApproving] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [rejectionReason, setRejectionReason] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);

  const modal = useSecureModal('approval_details', {
    requiresAuth: true,
    validateParams: (params) => Boolean(params.id),
    onError: (error) => {
      console.error('ApprovalDetailsModal error:', error);
      toast({
        title: "Lỗi",
        description: "Có lỗi xảy ra khi tải thông tin phê duyệt.",
        variant: "destructive"
      });
    }
  });

  const handleClose = () => {
    modal.close();
    navigate(-1); // Go back in history
  };

  // Fetch approval details based on type and id
  useEffect(() => {
    const fetchApprovalDetails = async () => {
      if (!id) return;

      setLoading(true);
      try {
        // For now, we'll focus on timesheet approvals as they're the most common
        // In the future, this can be extended for payrate, payroll, and employee approvals
        if (type === 'timesheet' || !type) {
          const timesheetId = parseInt(id, 10);
          const timesheet = await timesheetService.getTimesheetById(timesheetId);
          setApprovalData(timesheet);
        }
      } catch (error) {
        console.error('Failed to fetch approval details:', error);
        toast({
          title: "Lỗi",
          description: "Không thể tải thông tin phê duyệt.",
          variant: "destructive"
        });
      } finally {
        setLoading(false);
      }
    };

    fetchApprovalDetails();
  }, [id, type]);

  const handleApprove = async () => {
    if (!id || !approvalData) return;

    setApproving(true);
    try {
      await timesheetService.approveTimesheet(approvalData.id);
      toast({
        title: "Thành công",
        description: "Đã phê duyệt timesheet.",
      });

      // Refresh data
      const updatedTimesheet = await timesheetService.getTimesheetById(approvalData.id);
      setApprovalData(updatedTimesheet);
    } catch (error) {
      console.error('Approval error:', error);
      toast({
        title: "Lỗi",
        description: "Không thể phê duyệt timesheet.",
        variant: "destructive"
      });
    } finally {
      setApproving(false);
    }
  };

  const handleReject = async () => {
    if (!id || !approvalData || !rejectionReason.trim()) return;

    setRejecting(true);
    try {
      await timesheetService.rejectTimesheet(approvalData.id, {
        rejection_reason: rejectionReason
      });
      toast({
        title: "Thành công",
        description: "Đã loại timesheet.",
      });

      // Refresh data
      const updatedTimesheet = await timesheetService.getTimesheetById(approvalData.id);
      setApprovalData(updatedTimesheet);
      setShowRejectForm(false);
      setRejectionReason('');
    } catch (error) {
      console.error('Rejection error:', error);
      toast({
        title: "Lỗi",
        description: "Không thể loại timesheet.",
        variant: "destructive"
      });
    } finally {
      setRejecting(false);
    }
  };

  const getStatusBadge = (status: string) => {
    const statusMap = {
      pending_approval: { variant: 'outline' as const, label: 'Chờ phê duyệt', icon: Clock },
      approved: { variant: 'default' as const, label: 'Đã phê duyệt', icon: CheckCircle },
      rejected: { variant: 'destructive' as const, label: 'Đã loại', icon: XCircle },
      draft: { variant: 'secondary' as const, label: 'Bản nháp', icon: FileText }
    };

    const statusInfo = statusMap[status as keyof typeof statusMap] || statusMap.draft;
    const IconComponent = statusInfo.icon;

    return (
      <Badge variant={statusInfo.variant} className="gap-1">
        <IconComponent className="h-3 w-3" />
        {statusInfo.label}
      </Badge>
    );
  };

  return (
    <Sheet open={modal.isOpen} onOpenChange={handleClose}>
      <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto">
        <SheetHeader className="flex flex-row items-center justify-between space-y-0 pb-6" style={{ paddingTop: "max(0px, env(safe-area-inset-top))" }}>
          <SheetTitle>Chi tiết phê duyệt #{id}</SheetTitle>
          <Button variant="ghost" size="icon" onClick={handleClose}>
            <X className="h-4 w-4" />
          </Button>
        </SheetHeader>

        <div className="space-y-6">
          {loading ? (
            <div className="space-y-4">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-16 w-full" />
            </div>
          ) : approvalData ? (
            <>
              {/* Status and Basic Info */}
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="typography-title-medium">
                      Timesheet #{approvalData.id}
                    </CardTitle>
                    {getStatusBadge(approvalData.status)}
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div className="flex items-center gap-2">
                      <User className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">Nhân viên:</span>
                      <span>{approvalData.employeeName || '-'}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">Ngày:</span>
                      <span>{format(new Date(approvalData.date), 'dd/MM/yyyy')}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">Ca làm việc:</span>
                      <span>{approvalData.hours_worked} giờ</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">Loại công việc:</span>
                      <span>{approvalData.paytype || '-'}</span>
                    </div>
                  </div>

                  {approvalData.hour_type && (
                    <div>
                      <span className="font-medium typography-body-medium">Loại giờ làm:</span>
                      <p className="mt-1 text-sm text-muted-foreground">{approvalData.hour_type}</p>
                    </div>
                  )}

                  {approvalData.rejection_reason && (
                    <div className="p-3 bg-destructive/10 rounded-xl">
                      <span className="font-medium typography-body-medium text-destructive">Lý do loại:</span>
                      <p className="mt-1 text-sm text-destructive">{approvalData.rejection_reason}</p>
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Actions */}
              {approvalData.status === 'pending_approval' && (
                <Card>
                  <CardHeader>
                    <CardTitle className="typography-title-medium">Thao tác phê duyệt</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {!showRejectForm ? (
                      <div className="flex gap-3">
                        <Button
                          onClick={handleApprove}
                          disabled={approving}
                          className="flex-1"
                        >
                          {approving ? (
                            <>
                              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                              Đang phê duyệt...
                            </>
                          ) : (
                            <>
                              <CheckCircle className="h-4 w-4 mr-2" />
                              Phê duyệt
                            </>
                          )}
                        </Button>
                        <Button
                          variant="destructive"
                          onClick={() => setShowRejectForm(true)}
                          className="flex-1"
                        >
                          <XCircle className="h-4 w-4 mr-2" />
                          Loại
                        </Button>
                      </div>
                    ) : (
                      <div className="space-y-3">
                        <Label htmlFor="rejection-reason">Lý do loại *</Label>
                        <Textarea
                          id="rejection-reason"
                          value={rejectionReason}
                          onChange={(e) => setRejectionReason(e.target.value)}
                          placeholder="Nhập lý do loại..."
                          className="min-h-20"
                        />
                        <div className="flex gap-2">
                          <Button
                            variant="destructive"
                            onClick={handleReject}
                            disabled={rejecting || !rejectionReason.trim()}
                            className="flex-1"
                          >
                            {rejecting ? (
                              <>
                                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                                Đang loại...
                              </>
                            ) : (
                              'Xác nhận loại'
                            )}
                          </Button>
                          <Button
                            variant="outline"
                            onClick={() => {
                              setShowRejectForm(false);
                              setRejectionReason('');
                            }}
                            className="flex-1"
                          >
                            Đóng
                          </Button>
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}
            </>
          ) : (
            <div className="p-4 bg-muted rounded-xl text-center">
              <p className="typography-body-medium text-muted-foreground">
                Không tìm thấy thông tin phê duyệt.
              </p>
            </div>
          )}

          <div className="flex justify-end">
            <Button variant="outline" onClick={handleClose}>
              Đóng
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
