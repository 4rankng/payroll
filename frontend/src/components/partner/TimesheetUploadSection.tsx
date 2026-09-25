import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FileSpreadsheet, Upload } from "lucide-react";
import { formatFileSize } from "@/utils/partnerProjectHelpers";

interface TimesheetUploadSectionProps {
  selectedFile: File | null;
  isUploading: boolean;
  onFileSelect: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onUpload: () => void;
}

export const TimesheetUploadSection = ({
  selectedFile,
  isUploading,
  onFileSelect,
  onUpload
}: TimesheetUploadSectionProps) => {
  return (
    <Card className="bg-gradient-to-br from-slate-50 to-gray-100 border-0 shadow-sm">
      <CardHeader>
        <CardTitle className="flex items-center">
          <FileSpreadsheet className="w-5 h-5 mr-2" />
          Tải lên bảng công
        </CardTitle>
        <CardDescription>Tải lên file Excel chứa thông tin chấm công</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <Label htmlFor="timesheet-file">Chọn file Excel</Label>
          <Input
            id="timesheet-file"
            type="file"
            accept=".xls,.xlsx"
            onChange={onFileSelect}
            className="mt-2"
          />
          <p className="typography-body-small text-muted-foreground mt-1">
            Chỉ hỗ trợ file Excel (.xls, .xlsx)
          </p>
        </div>
        
        {selectedFile && (
          <div className="flex items-center space-x-2 p-3 bg-card/50 rounded-xl border">
            <div className="p-1 bg-green-100 rounded">
              <FileSpreadsheet className="w-4 h-4 text-green-700" />
            </div>
            <div className="flex-1">
              <span className="typography-body-medium">{selectedFile.name}</span>
              <p className="typography-body-small text-muted-foreground">
                {formatFileSize(selectedFile.size)}
              </p>
            </div>
          </div>
        )}
        
        <Button 
          onClick={onUpload}
          disabled={!selectedFile || isUploading}
          className="w-full md:w-auto"
        >
          {isUploading ? (
            <>
              <div className="w-4 h-4 mr-2 border-2 border-white border-t-transparent rounded-full animate-spin" />
              Đang tải lên...
            </>
          ) : (
            <>
              <Upload className="w-4 h-4 mr-2" />
              Tải lên bảng công
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  );
};