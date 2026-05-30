import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { FileDropZone } from "./FileDropZone";
import { useUploadAndSettleReconciliation } from "@/hooks/api/useAdvancePaymentReconciliation";

interface UploadAndSettleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  forMonth?: string;
}

export function UploadAndSettleDialog({
  open,
  onOpenChange,
  forMonth,
}: UploadAndSettleDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const uploadAndSettleMutation = useUploadAndSettleReconciliation();

  const handleSubmit = useCallback(() => {
    if (!selectedFile) return;
    const formData = new FormData();
    formData.append("file", selectedFile);
    formData.append(
      "forMonth",
      forMonth || new Date().toISOString().slice(0, 7),
    );
    uploadAndSettleMutation.mutate(formData, {
      onSettled: () => {
        onOpenChange(false);
        setSelectedFile(null);
      },
    });
  }, [selectedFile, forMonth, uploadAndSettleMutation, onOpenChange]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
    setSelectedFile(null);
  }, [onOpenChange]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Tải lên & Đối soát</DialogTitle>
          <DialogDescription>
            Tải lên file điều chỉnh đối soát thanh toán ứng lương
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <Label className="mb-2 block">File Excel kết quả điều chỉnh</Label>
            <FileDropZone
              file={selectedFile}
              onFileChange={setSelectedFile}
              inputId="settle-file-input"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={uploadAndSettleMutation.isPending}
          >
            Hủy
          </Button>
          <Button
            variant="info"
            onClick={handleSubmit}
            disabled={!selectedFile || uploadAndSettleMutation.isPending}
          >
            {uploadAndSettleMutation.isPending
              ? "Đang xử lý..."
              : "Tải lên & Đối soát"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
