import { useState, useRef, useEffect, useCallback } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogClose } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ScrollArea } from '@/components/ui/scroll-area';
import { toast } from '@/components/ui/sonner';
import {
  Upload,
  FileSpreadsheet,
  CheckCircle,
  XCircle,
  Loader2,
  X,
  Download,
  Info,
  AlertCircle
} from 'lucide-react';
import { employeeService } from '@/services/api/employee.service';

interface EmployeeImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  onImportSuccess?: () => void;
}

interface ImportStatus {
  import_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  total_rows: number;
  processed_rows: number;
  created_count: number;
  updated_count: number;
  error_count: number;
  errors: Array<{ row_number: number; cccd?: string; message: string }>;
  started_at: string;
  completed_at?: string;
}

const POLLING_INTERVAL = 1000; // 1 second

export function EmployeeImportModal({
  isOpen,
  onClose,
  onImportSuccess
}: EmployeeImportModalProps) {
  const [uploading, setUploading] = useState(false);
  const [importId, setImportId] = useState<string | null>(null);
  const [importStatus, setImportStatus] = useState<ImportStatus | null>(null);
  const [uploadedFileName, setUploadedFileName] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, []);

  // Reset state when modal opens/closes
  useEffect(() => {
    if (!isOpen) {
      // Reset state when modal closes
      setUploading(false);
      setImportId(null);
      setImportStatus(null);
      setUploadedFileName(null);
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
    }
  }, [isOpen]);

  // Poll for import status
  const pollImportStatus = useCallback(async (id: string) => {
    try {
      const status = await employeeService.getImportStatus(id);
      setImportStatus(status);

      // Stop polling if import is completed or failed
      if (status.status === 'completed' || status.status === 'failed') {
        if (pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
          pollingIntervalRef.current = null;
        }

        if (status.status === 'completed') {
          toast({
            title: "Import thành công",
            description: `Đã import ${status.created_count} nhân viên mới, cập nhật ${status.updated_count} nhân viên.`,
          });
          if (onImportSuccess) {
            onImportSuccess();
          }
        } else {
          toast({
            title: "Import thất bại",
            description: "Có lỗi xảy ra trong quá trình import.",
            variant: "destructive"
          });
        }
      }
    } catch (error) {
      console.error('Error polling import status:', error);
      // Don't stop polling on error, might be a temporary network issue
    }
  }, [onImportSuccess]);

  // Start polling when we have an import ID
  useEffect(() => {
    if (importId && !pollingIntervalRef.current) {
      pollingIntervalRef.current = setInterval(() => {
        pollImportStatus(importId);
      }, POLLING_INTERVAL);

      // Initial status check
      pollImportStatus(importId);
    }
  }, [importId, pollImportStatus]);

  const handleFileUpload = async (file: File) => {
    // Validate file extension
    const ext = file.name.toLowerCase();
    if (!ext.endsWith('.xlsx')) {
      toast({
        title: "Sai định dạng file",
        description: "Vui lòng tải lên file Excel (.xlsx).",
        variant: "destructive"
      });
      return;
    }

    // Validate file size (max 10MB)
    if (file.size > 10 * 1024 * 1024) {
      toast({
        title: "File quá lớn",
        description: "Kích thước file không được vượt quá 10MB.",
        variant: "destructive"
      });
      return;
    }

    setUploading(true);
    setUploadedFileName(file.name);

    try {
      const response = await employeeService.importEmployees(file);
      setImportId(response.import_id);

      toast({
        title: "Đã gửi file import",
        description: response.message || "File đang được xử lý.",
      });
    } catch (error) {
      console.error('Import error:', error);
      toast({
        title: "Lỗi import",
        description: error instanceof Error ? error.message : "Không thể import file. Vui lòng thử lại.",
        variant: "destructive"
      });
      setUploading(false);
      setUploadedFileName(null);
    }
  };

  const downloadTemplate = async () => {
    try {
      // Create template data with expected columns
      const templateData = [
        // Headers row - matching backend expected format
        [
          'Fullname',
          'CCCD',
          'Email',
          'Mobile',
          'Address',
          'Date of Birth',
          'Bank Account',
          'Bank Name',
          'Project Code',
          'Position',
          'Start Date',
          'Payment Schedule'
        ],
        // Sample data row
        [
          'Nguyễn Văn A',
          '123456789012',
          'nguyenvana@example.com',
          '0123456789',
          'Hà Nội, Việt Nam',
          '1990-01-01',
          '1234567890',
          'Vietcombank',
          'PROJ001',
          'Nhân viên',
          '2024-01-01',
          'flexible'
        ]
      ];

      // Create workbook and worksheet
      const XLSX = await import('xlsx');
      const workbook = XLSX.utils.book_new();
      const worksheet = XLSX.utils.aoa_to_sheet(templateData);

      // Set column widths
      worksheet['!cols'] = [
        { wch: 25 }, // Fullname
        { wch: 15 }, // CCCD
        { wch: 25 }, // Email
        { wch: 12 }, // Mobile
        { wch: 30 }, // Address
        { wch: 12 }, // Date of Birth
        { wch: 15 }, // Bank Account
        { wch: 20 }, // Bank Name
        { wch: 15 }, // Project Code
        { wch: 15 }, // Position
        { wch: 12 }, // Start Date
        { wch: 15 }, // Payment Schedule
      ];

      // Add worksheet to workbook
      XLSX.utils.book_append_sheet(workbook, worksheet, 'Nhân viên');

      // Generate and download file
      XLSX.writeFile(workbook, 'template_import_nhan_vien.xlsx');

      toast({
        title: "Tải template thành công",
        description: "File template đã được tải xuống.",
      });
    } catch (error) {
      console.error('Template generation error:', error);
      toast({
        title: "Lỗi tạo template",
        description: "Không thể tạo file template. Vui lòng thử lại.",
        variant: "destructive"
      });
    }
  };

  const getProgressPercentage = () => {
    if (!importStatus) return 0;
    if (importStatus.total_rows === 0) return 0;
    return Math.round((importStatus.processed_rows / importStatus.total_rows) * 100);
  };

  const getStatusBadge = () => {
    if (!importStatus) return null;

    switch (importStatus.status) {
      case 'pending':
        return <Badge variant="secondary">Chờ xử lý</Badge>;
      case 'processing':
        return <Badge variant="default" className="bg-blue-500">Đang xử lý</Badge>;
      case 'completed':
        return <Badge variant="default" className="bg-green-500">Hoàn thành</Badge>;
      case 'failed':
        return <Badge variant="destructive">Thất bại</Badge>;
      default:
        return null;
    }
  };

  const isProcessing = uploading || importStatus?.status === 'pending' || importStatus?.status === 'processing';
  const isCompleted = importStatus?.status === 'completed' || importStatus?.status === 'failed';

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogClose asChild>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-4 right-4 h-8 w-8 rounded-xl hover:bg-muted/50"
            disabled={isProcessing}
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Đóng</span>
          </Button>
        </DialogClose>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Upload className="h-5 w-5" />
            Import danh sách nhân viên
          </DialogTitle>
          <DialogDescription>
            Import nhân viên từ file Excel. Quá trình xử lý sẽ diễn ra ngầm (async).
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Step 1: Upload */}
          {!importId && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileSpreadsheet className="h-5 w-5" />
                  Tải lên file Excel
                </CardTitle>
                <CardDescription>
                  Hỗ trợ file .xlsx với kích thước tối đa 10MB.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <Alert>
                  <Info className="h-4 w-4" />
                  <AlertDescription>
                    <strong>Định dạng file mong đợi:</strong>
                    <ul className="list-disc list-inside mt-2 text-sm">
                      <li>Dòng đầu tiên: Tiêu đề cột (Fullname, CCCD, Email, ...)</li>
                      <li>Dòng thứ 2 trở đi: Dữ liệu nhân viên</li>
                      <li>Trường bắt buộc: Fullname, CCCD</li>
                    </ul>
                  </AlertDescription>
                </Alert>

                <div className="flex items-center justify-center gap-4">
                  <Button variant="outline" onClick={downloadTemplate} disabled={uploading}>
                    <Download className="h-4 w-4 mr-2" />
                    Tải template mẫu
                  </Button>

                  <Button
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploading}
                    variant="default"
                  >
                    {uploading ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Đang tải lên...
                      </>
                    ) : (
                      <>
                        <Upload className="h-4 w-4 mr-2" />
                        Chọn file Excel
                      </>
                    )}
                  </Button>
                </div>

                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".xlsx"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) handleFileUpload(file);
                  }}
                  className="hidden"
                />

                {uploadedFileName && !importId && (
                  <div className="flex items-center gap-2 p-3 bg-muted rounded-xl">
                    <FileSpreadsheet className="h-4 w-4 text-green-600" />
                    <span className="text-sm">{uploadedFileName}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Step 2: Progress */}
          {importId && importStatus && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="flex items-center gap-2">
                    {importStatus.status === 'processing' && <Loader2 className="h-5 w-5 animate-spin" />}
                    {importStatus.status === 'completed' && <CheckCircle className="h-5 w-5 text-green-600" />}
                    {importStatus.status === 'failed' && <XCircle className="h-5 w-5 text-red-600" />}
                    {importStatus.status === 'pending' && <AlertCircle className="h-5 w-5 text-yellow-600" />}
                    Trạng thái import
                  </CardTitle>
                  {getStatusBadge()}
                </div>
                <CardDescription>
                  Import ID: {importId}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {/* Progress bar */}
                {importStatus.status === 'pending' || importStatus.status === 'processing' ? (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span>Đang xử lý...</span>
                      <span>{importStatus.processed_rows} / {importStatus.total_rows} dòng</span>
                    </div>
                    <Progress value={getProgressPercentage()} className="w-full" />
                  </div>
                ) : (
                  <Progress value={100} className="w-full" />
                )}

                {/* Stats */}
                <div className="grid grid-cols-4 gap-4 text-center">
                  <div className="p-3 bg-blue-50 rounded-xl">
                    <div className="text-2xl font-bold text-blue-600">{importStatus.total_rows}</div>
                    <div className="text-sm text-blue-600">Tổng dòng</div>
                  </div>
                  <div className="p-3 bg-green-50 rounded-xl">
                    <div className="text-2xl font-bold text-green-600">{importStatus.created_count}</div>
                    <div className="text-sm text-green-600">Mới</div>
                  </div>
                  <div className="p-3 bg-yellow-50 rounded-xl">
                    <div className="text-2xl font-bold text-yellow-600">{importStatus.updated_count}</div>
                    <div className="text-sm text-yellow-600">Cập nhật</div>
                  </div>
                  <div className="p-3 bg-red-50 rounded-xl">
                    <div className="text-2xl font-bold text-red-600">{importStatus.error_count}</div>
                    <div className="text-sm text-red-600">Lỗi</div>
                  </div>
                </div>

                {/* Errors */}
                {importStatus.errors.length > 0 && (
                  <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                      <div className="font-medium mb-2">Phát hiện {importStatus.errors.length} lỗi:</div>
                      <ScrollArea className="h-32">
                        {importStatus.errors.slice(0, 20).map((error, index) => (
                          <div key={index} className="text-sm py-1 border-b last:border-0">
                            Dòng {error.row_number}{error.cccd ? ` (${error.cccd})` : ''}: {error.message}
                          </div>
                        ))}
                        {importStatus.errors.length > 20 && (
                          <div className="text-sm py-1 font-medium">
                            ... và {importStatus.errors.length - 20} lỗi khác
                          </div>
                        )}
                      </ScrollArea>
                    </AlertDescription>
                  </Alert>
                )}

                {/* Info message while processing */}
                {(importStatus.status === 'pending' || importStatus.status === 'processing') && (
                  <Alert>
                    <Info className="h-4 w-4" />
                    <AlertDescription>
                      Quá trình import đang diễn ra ngầm. Bạn có thể đóng modal này và quay lại sau.
                      Tiến độ sẽ được tiếp tục xử lý.
                    </AlertDescription>
                  </Alert>
                )}
              </CardContent>
            </Card>
          )}

          {/* Actions */}
          <div className="flex gap-2 justify-end">
            <Button
              variant="outline"
              onClick={onClose}
              disabled={isProcessing && !isCompleted}
            >
              {isCompleted ? 'Đóng' : 'Hủy'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
