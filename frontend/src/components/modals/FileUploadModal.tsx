import { useState, useRef } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogClose } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { toast } from '@/components/ui/sonner';
import { 
  Upload, 
  Download, 
  FileSpreadsheet, 
  FileText, 
  File,
  CheckCircle, 
  XCircle,
  AlertTriangle,
  Loader2,
  X,
  Eye,
  ExternalLink
} from 'lucide-react';

export const modalConfig = {
  id: 'file-upload',
};

const uploadFile = async (fileName: string, fileSize: number) => {
  await new Promise(resolve => setTimeout(resolve, 1000));
  return {
    success: Math.random() > 0.2, // 80% success rate
    message: 'Upload successful',
    data: { recordCount: Math.floor(Math.random() * 100) + 10 }
  };
};

const exportData = async (formatName: string, format: string) => {
  await new Promise(resolve => setTimeout(resolve, 1000));
  return {
    success: true,
    message: 'Export successful',
    downloadUrl: `#${formatName.toLowerCase().replace(/\s+/g, '-')}`
  };
};

interface FileUploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  type: 'upload' | 'export' | 'payroll-export';
  title: string;
  description?: string;
  acceptedFiles?: string[];
  maxFileSize?: number; // in MB
  onFileProcessed?: (result: { success: boolean; data?: unknown; error?: string }) => void;
}

interface UploadResult {
  fileName: string;
  success: boolean;
  recordCount?: number;
  errors?: string[];
  warnings?: string[];
}

const fileTypeIcons = {
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': FileSpreadsheet,
  'application/vnd.ms-excel': FileSpreadsheet,
  'text/csv': FileSpreadsheet,
  'application/pdf': FileText,
  'default': File
};

const getFileIcon = (fileType: string) => {
  return fileTypeIcons[fileType as keyof typeof fileTypeIcons] || fileTypeIcons.default;
};

const payrollExportFormats = [
  {
    id: 'bank-transfer',
    name: 'Chuyển khoản ngân hàng',
    description: 'File Excel định dạng cho chuyển khoản tự động',
    icon: FileSpreadsheet,
    format: 'excel',
    fields: ['STT', 'Tên nhân viên', 'Số tài khoản', 'Ngân hàng', 'Số tiền', 'Nội dung chuyển khoản']
  },
  {
    id: 'salary-slip',
    name: 'Phiếu lương',
    description: 'Phiếu lương PDF cho từng nhân viên',
    icon: FileText,
    format: 'pdf',
    fields: ['Thông tin cá nhân', 'Chi tiết lương', 'Khấu trừ', 'Lương thực nhận']
  },
  {
    id: 'tax-report',
    name: 'Báo cáo thuế',
    description: 'Báo cáo thuế thu nhập cá nhân',
    icon: FileSpreadsheet,
    format: 'excel',
    fields: ['Mã số thuế', 'Lương brutto', 'Thuế TNCN', 'Bảo hiểm']
  },
  {
    id: 'detailed-payroll',
    name: 'Bảng lương chi tiết',
    description: 'Bảng lương chi tiết đầy đủ thông tin',
    icon: FileSpreadsheet,
    format: 'excel',
    fields: ['Thông tin NV', 'Lương CB', 'Phụ cấp', 'Thưởng', 'Khấu trừ', 'Thực lãnh']
  }
];

export function FileUploadModal({
  isOpen,
  onClose,
  type,
  title,
  description,
  acceptedFiles = ['.xls', '.xlsx', '.csv'],
  maxFileSize = 10,
  onFileProcessed
}: FileUploadModalProps) {
  const [dragOver, setDragOver] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [progress, setProgress] = useState(0);
  const [results, setResults] = useState<UploadResult[]>([]);
  const [selectedExportFormat, setSelectedExportFormat] = useState(payrollExportFormats[0]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (files: FileList | null) => {
    if (!files || files.length === 0) return;

    const fileArray = Array.from(files);
    processFiles(fileArray);
  };

  const processFiles = async (files: File[]) => {
    setUploading(true);
    setProgress(0);
    setResults([]);

    const newResults: UploadResult[] = [];

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      
      // Validate file size
      if (file.size > maxFileSize * 1024 * 1024) {
        newResults.push({
          fileName: file.name,
          success: false,
          errors: [`Kích thước file vượt quá ${maxFileSize}MB`]
        });
        continue;
      }

      // Validate file type
      const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase();
      if (!acceptedFiles.includes(fileExtension)) {
        newResults.push({
          fileName: file.name,
          success: false,
          errors: [`Định dạng file không được hỗ trợ. Chấp nhận: ${acceptedFiles.join(', ')}`]
        });
        continue;
      }

      // Simulate file processing
      try {
        const result = await uploadFile(file.name, file.size);
        newResults.push({
          fileName: file.name,
          success: result.success,
          recordCount: result.data?.recordCount,
          errors: result.success ? undefined : [result.message],
          warnings: result.success && Math.random() > 0.7 ? ['Một số bản ghi có dữ liệu không đầy đủ'] : undefined
        });
        
        if (result.success && onFileProcessed) {
          onFileProcessed({ success: true, data: result.data });
        }
      } catch (error) {
        newResults.push({
          fileName: file.name,
          success: false,
          errors: ['Lỗi xử lý file']
        });
      }

      setProgress(((i + 1) / files.length) * 100);
      setResults([...newResults]);
    }

    setUploading(false);

    // Show success toast if all files processed successfully
    const successCount = newResults.filter(r => r.success).length;
    const totalCount = newResults.length;

    if (successCount === totalCount) {
      toast({
        title: "Thành công",
        description: `Đã xử lý thành công ${successCount} file.`,
      });
    } else if (successCount > 0) {
      toast({
        title: "Hoàn thành một phần",
        description: `Xử lý thành công ${successCount}/${totalCount} file.`,
        variant: "destructive"
      });
    } else {
      toast({
        title: "Thất bại",
        description: "Không thể xử lý file nào.",
        variant: "destructive"
      });
    }
  };

  const handleExport = async () => {
    setExporting(true);
    setProgress(0);

    try {
      // Simulate export progress
      for (let i = 0; i <= 100; i += 10) {
        setProgress(i);
        await new Promise(resolve => setTimeout(resolve, 150));
      }

      const result = await exportData(selectedExportFormat.name, selectedExportFormat.format as 'excel' | 'pdf' | 'csv');
      
      if (result.success) {
        toast({
          title: "Xuất file thành công",
          description: `Đã tạo file ${selectedExportFormat.name}`,
        });

        // Simulate file download
        const link = document.createElement('a');
        link.href = result.downloadUrl || '#';
        link.download = `${selectedExportFormat.id}_${Date.now()}.${selectedExportFormat.format === 'excel' ? 'xlsx' : selectedExportFormat.format}`;
        document.body.appendChild(link);
        link.click();
        // Safely remove the link
        if (link.parentNode) {
          link.parentNode.removeChild(link);
        }

        if (onFileProcessed) {
          onFileProcessed(result);
        }
      } else {
        throw new Error(result.message);
      }
    } catch (error) {
      toast({
        title: "Lỗi xuất file",
        description: "Không thể tạo file xuất. Vui lòng thử lại.",
        variant: "destructive"
      });
    } finally {
      setExporting(false);
      setProgress(0);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    handleFileSelect(e.dataTransfer.files);
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogClose asChild>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-3 right-3 h-11 w-11 rounded-xl sm:top-4 sm:right-4 sm:h-8 sm:w-8 hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 z-10"
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Đóng</span>
          </Button>
        </DialogClose>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {type === 'upload' ? <Upload className="h-5 w-5" /> : <Download className="h-5 w-5" />}
            {title}
          </DialogTitle>
          {description && (
            <DialogDescription>{description}</DialogDescription>
          )}
        </DialogHeader>

        <div className="space-y-6">
          {/* Upload Section */}
          {type === 'upload' && (
            <Card>
              <CardHeader>
                <CardTitle>Tải lên file</CardTitle>
                <CardDescription>
                  Kéo thả file vào đây hoặc nhấn để chọn file. 
                  Định dạng hỗ trợ: {acceptedFiles.join(', ')}. 
                  Kích thước tối đa: {maxFileSize}MB.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div
                  className={`border-2 border-dashed rounded-xl p-6 text-center transition-colors ${
                    dragOver 
                      ? 'border-border bg-muted/50' 
                      : 'border-border hover:border-border'
                  }`}
                  onDragOver={handleDragOver}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop}
                >
                  {uploading ? (
                    <div className="space-y-4">
                      <Loader2 className="h-12 w-12 animate-spin mx-auto text-muted-foreground" />
                      <div>
                        <p className="typography-title-large">Đang xử lý file...</p>
                        <Progress value={progress} className="w-full mt-2" />
                        <p className="typography-body-medium text-muted-foreground mt-1">{Math.round(progress)}% hoàn thành</p>
                      </div>
                    </div>
                  ) : (
                    <div className="space-y-4">
                      <Upload className="h-12 w-12 mx-auto text-gray-400" />
                      <div>
                        <p className="typography-title-large">Kéo thả file vào đây</p>
                        <p className="text-muted-foreground">hoặc</p>
                        <Button 
                          variant="outline" 
                          onClick={() => fileInputRef.current?.click()}
                          className="mt-2"
                        >
                          Chọn file từ máy tính
                        </Button>
                      </div>
                    </div>
                  )}
                </div>

                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  accept={acceptedFiles.join(',')}
                  onChange={(e) => handleFileSelect(e.target.files)}
                  className="hidden"
                />
              </CardContent>
            </Card>
          )}

          {/* Export Section */}
          {(type === 'export' || type === 'payroll-export') && (
            <Card>
              <CardHeader>
                <CardTitle>Chọn định dạng xuất</CardTitle>
                <CardDescription>
                  Chọn loại báo cáo và định dạng file muốn xuất
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {type === 'payroll-export' && (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {payrollExportFormats.map((format) => {
                      const IconComponent = format.icon;
                      return (
                        <Card 
                          key={format.id} 
                          className={`cursor-pointer transition-colors ${
                            selectedExportFormat.id === format.id 
                              ? 'ring-2 ring-slate-500 bg-muted/50' 
                              : 'hover:bg-muted/50'
                          }`}
                          onClick={() => setSelectedExportFormat(format)}
                        >
                          <CardContent className="p-4">
                            <div className="flex items-start gap-3">
                              <IconComponent className="h-8 w-8 text-muted-foreground mt-1" />
                              <div className="flex-1">
                                <h3 className="font-medium">{format.name}</h3>
                                <p className="typography-body-medium text-muted-foreground mb-2">{format.description}</p>
                                <div className="flex flex-wrap gap-1">
                                  {format.fields.slice(0, 3).map((field, index) => (
                                    <Badge key={index} variant="secondary" className="typography-body-small">
                                      {field}
                                    </Badge>
                                  ))}
                                  {format.fields.length > 3 && (
                                    <Badge variant="outline" className="typography-body-small">
                                      +{format.fields.length - 3}
                                    </Badge>
                                  )}
                                </div>
                              </div>
                            </div>
                          </CardContent>
                        </Card>
                      );
                    })}
                  </div>
                )}

                <Separator />

                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="font-medium">File sẽ được tạo:</h4>
                    <p className="typography-body-medium text-muted-foreground">
                      {selectedExportFormat.name}.{selectedExportFormat.format === 'excel' ? 'xlsx' : selectedExportFormat.format}
                    </p>
                  </div>
                  <Button 
                    onClick={handleExport} 
                    disabled={exporting}
                    className="min-w-[120px]"
                  >
                    {exporting ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Đang xuất...
                      </>
                    ) : (
                      <>
                        <Download className="h-4 w-4 mr-2" />
                        Xuất file
                      </>
                    )}
                  </Button>
                </div>

                {exporting && (
                  <div className="space-y-2">
                    <Progress value={progress} className="w-full" />
                    <p className="typography-body-medium text-muted-foreground text-center">
                      Đang tạo file... {Math.round(progress)}%
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Results Section */}
          {results.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  Kết quả xử lý
                  <Badge variant="secondary">{results.length} file</Badge>
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {results.map((result, index) => {
                    const IconComponent = result.success ? CheckCircle : XCircle;
                    return (
                      <div key={index} className="flex items-start gap-3 p-3 border rounded-xl">
                        <IconComponent className={`h-5 w-5 mt-0.5 ${result.success ? 'text-green-500' : 'text-red-500'}`} />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <p className="font-medium truncate">{result.fileName}</p>
                            <Badge variant={result.success ? 'default' : 'destructive'}>
                              {result.success ? 'Thành công' : 'Thất bại'}
                            </Badge>
                          </div>
                          
                          {result.success && result.recordCount && (
                            <p className="typography-body-medium text-green-600 mt-1">
                              Đã xử lý {result.recordCount} bản ghi
                            </p>
                          )}

                          {result.warnings && result.warnings.length > 0 && (
                            <div className="mt-2">
                              {result.warnings.map((warning, wIndex) => (
                                <div key={wIndex} className="flex items-center gap-2 typography-body-medium text-yellow-600">
                                  <AlertTriangle className="h-4 w-4" />
                                  {warning}
                                </div>
                              ))}
                            </div>
                          )}

                          {result.errors && result.errors.length > 0 && (
                            <div className="mt-2">
                              {result.errors.map((error, eIndex) => (
                                <p key={eIndex} className="typography-body-medium text-red-600">
                                  • {error}
                                </p>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Special Payroll Export Info */}
          {type === 'payroll-export' && selectedExportFormat.id === 'bank-transfer' && (
            <Card className="border-border bg-muted/50">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-foreground">
                  <FileSpreadsheet className="h-5 w-5" />
                  Hướng dẫn chuyển khoản
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-foreground">
                <div className="flex items-start gap-2">
                  <div className="w-6 h-6 rounded-full bg-slate-200 flex items-center justify-center typography-body-medium">1</div>
                  <p>Tải xuống file Excel chứa thông tin chuyển khoản</p>
                </div>
                <div className="flex items-start gap-2">
                  <div className="w-6 h-6 rounded-full bg-slate-200 flex items-center justify-center typography-body-medium">2</div>
                  <p>Kiểm tra thông tin tài khoản và số tiền của từng nhân viên</p>
                </div>
                <div className="flex items-start gap-2">
                  <div className="w-6 h-6 rounded-full bg-slate-200 flex items-center justify-center typography-body-medium">3</div>
                  <p>Import file vào hệ thống Internet Banking của ngân hàng</p>
                </div>
                <div className="flex items-start gap-2">
                  <div className="w-6 h-6 rounded-full bg-slate-200 flex items-center justify-center typography-body-medium">4</div>
                  <p>Thực hiện chuyển khoản hàng loạt theo file đã tạo</p>
                </div>
                
                <div className="mt-4 p-3 bg-card rounded-xl border border-border">
                  <p className="typography-body-medium">Lưu ý quan trọng:</p>
                  <ul className="typography-body-medium mt-1 space-y-1">
                    <li>• Kiểm tra kỹ thông tin tài khoản trước khi chuyển khoản</li>
                    <li>• Đảm bảo số dư tài khoản đủ để thực hiện giao dịch</li>
                    <li>• Lưu lại biên lai chuyển khoản để đối chiếu</li>
                  </ul>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Action Buttons */}
          <div className="flex justify-end gap-2">
            <Button variant="outline" onClick={onClose}>
              <X className="h-4 w-4 mr-2" />
              Đóng
            </Button>
            {results.length > 0 && (
              <Button variant="outline" onClick={() => setResults([])}>
                Làm mới
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}