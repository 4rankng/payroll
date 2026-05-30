import { useState } from "react";
import { TimesheetUpload } from "@/types/partnerProject";
import { validateExcelFile } from "@/utils/partnerProjectHelpers";

export const useTimesheetUpload = () => {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploads] = useState<TimesheetUpload[]>([]);

  const handleFileSelect = (file: File | null): { success: boolean; error?: string } => {
    if (!file) {
      setSelectedFile(null);
      return { success: true };
    }

    const validation = validateExcelFile(file);
    if (!validation.isValid) {
      return { success: false, error: validation.error };
    }

    setSelectedFile(file);
    return { success: true };
  };

  const handleUpload = async (): Promise<{ success: boolean; error?: string }> => {
    if (!selectedFile) {
      return { success: false, error: "Chưa chọn file. Vui lòng chọn file Excel để tải lên" };
    }

    setIsUploading(true);
    
    try {
      // Simulate upload process with delay
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // In a real app, this would be an actual API call
      // const uploadResult = await uploadTimesheetFile(selectedFile);
      
      setSelectedFile(null);
      return { success: true };
    } catch (error) {
      return { success: false, error: "Lỗi khi tải lên file. Vui lòng thử lại" };
    } finally {
      setIsUploading(false);
    }
  };

  const clearSelectedFile = () => {
    setSelectedFile(null);
  };

  return {
    selectedFile,
    isUploading,
    uploads,
    handleFileSelect,
    handleUpload,
    clearSelectedFile,
  };
};