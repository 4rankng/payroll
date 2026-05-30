import { useState, useRef, useCallback } from 'react';
import { Upload, X, File, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';
import { assetService } from '@/services/api/asset.service';

interface FileUploadProps {
  onFilesSelected: (files: File[]) => void;
  accept?: string;
  multiple?: boolean;
  maxFiles?: number;
  maxSize?: number;
  className?: string;
  disabled?: boolean;
  children?: React.ReactNode;
  validateFile?: (file: File) => { isValid: boolean; error?: string };
  showSelectedFilesList?: boolean;
}

interface FileWithPreview extends File {
  preview?: string;
  error?: string;
  uploading?: boolean;
  progress?: number;
}

export function FileUpload({
  onFilesSelected,
  accept = "image/*,.pdf,.doc,.docx,.xls,.xlsx,.txt",
  multiple = true,
  maxFiles = 10,
  maxSize,
  className,
  disabled = false,
  children,
  validateFile,
  showSelectedFilesList = true,
}: FileUploadProps) {
  const [selectedFiles, setSelectedFiles] = useState<FileWithPreview[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFiles = useCallback((files: FileList | null) => {
    if (!files) return;

    const fileArray = Array.from(files);
    const newFiles: FileWithPreview[] = [];
    const errors: string[] = [];

    if (multiple) {
      // Check total file count for multi-select
      if (selectedFiles.length + fileArray.length > maxFiles) {
        errors.push(`Chỉ được phép tải lên tối đa ${maxFiles} file`);
        return;
      }
    } else {
      fileArray.splice(1);
    }

    fileArray.forEach((file) => {
      // Use custom validation if provided, otherwise use asset service validation
      const validation = validateFile ? validateFile(file) : assetService.validateFileForEvidence(file);

      if (!validation.isValid) {
        errors.push(`${file.name}: ${validation.error}`);
        return;
      }

      // Create preview for images
      let preview: string | undefined;
      if (file.type.startsWith('image/')) {
        preview = URL.createObjectURL(file);
      }

      const fileWithPreview: FileWithPreview = Object.assign(file, {
        preview,
        error: undefined,
        uploading: false,
        progress: 0,
      });

      newFiles.push(fileWithPreview);
    });

    if (errors.length > 0) {
      // Handle errors - you might want to show a toast here
      console.error('File validation errors:', errors);
      return;
    }

    const updatedFiles = multiple ? [...selectedFiles, ...newFiles] : newFiles;
    setSelectedFiles(updatedFiles);
    onFilesSelected(updatedFiles);
  }, [selectedFiles, maxFiles, validateFile, onFilesSelected, multiple]);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!disabled) {
      setDragActive(true);
    }
  }, [disabled]);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (!disabled) {
      const files = e.dataTransfer.files;
      handleFiles(files);
    }
  }, [disabled, handleFiles]);

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    handleFiles(e.target.files);
    // Reset input
    if (inputRef.current) {
      inputRef.current.value = '';
    }
  }, [handleFiles]);

  const removeFile = useCallback((index: number) => {
    setSelectedFiles((prev) => {
      const file = prev[index];
      if (file.preview) {
        URL.revokeObjectURL(file.preview);
      }
      const updated = prev.filter((_, i) => i !== index);
      onFilesSelected(updated);
      return updated;
    });
  }, [onFilesSelected]);

  const openFileDialog = useCallback(() => {
    if (!disabled) {
      inputRef.current?.click();
    }
  }, [disabled]);

  const getFileIcon = (file: File) => {
    const type = file.type.toLowerCase();
    const name = file.name.toLowerCase();

    if (type.startsWith('image/')) return '🖼️';
    if (type.includes('pdf') || name.endsWith('.pdf')) return '📄';
    if (type.includes('word') || name.endsWith('.doc') || name.endsWith('.docx')) return '📝';
    if (type.includes('excel') || type.includes('spreadsheet') || name.endsWith('.xls') || name.endsWith('.xlsx')) return '📊';
    if (type.includes('text') || name.endsWith('.txt')) return '📋';
    
    return '📎';
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-4 w-full">
      {/* Upload Area */}
      <div
        className={cn(
          "relative border-2 border-dashed rounded-lg p-4 transition-colors w-full",
          dragActive && !disabled && "border-blue-400 bg-blue-50",
          !dragActive && !disabled && "border-gray-300 hover:border-gray-400",
          disabled && "border-gray-200 bg-gray-50 cursor-not-allowed opacity-60",
          className
        )}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <input
          ref={inputRef}
          type="file"
          accept={accept}
          multiple={multiple}
          onChange={handleInputChange}
          disabled={disabled}
          className="sr-only"
        />

        {children ? (
          <div onClick={openFileDialog} className="cursor-pointer">
            {children}
          </div>
        ) : (
          <div
            onClick={openFileDialog}
            className={cn(
              "flex flex-col items-center justify-center text-center space-y-2 py-2",
              !disabled && "cursor-pointer"
            )}
          >
            <div
              className={cn(
                "w-10 h-10 rounded-full flex items-center justify-center",
                dragActive && !disabled && "bg-blue-100",
                !dragActive && !disabled && "bg-gray-100",
                disabled && "bg-gray-200"
              )}
            >
              <Upload
                className={cn(
                  "w-5 h-5",
                  dragActive && !disabled && "text-blue-600",
                  !dragActive && !disabled && "text-gray-500",
                  disabled && "text-gray-400"
                )}
              />
            </div>

            <div className="space-y-1">
              <p className="text-sm font-medium text-gray-900">
                {dragActive ? "Thả file vào đây" : "Tải lên chứng từ"}
              </p>
              <p className="text-sm text-gray-500">
                Kéo thả hoặc{" "}
                <span className="text-blue-600 hover:text-blue-700 font-medium">
                  chọn file
                </span>
              </p>
              <p className="text-xs text-gray-400">
                PNG, JPG, PDF, DOC, XLS (tối đa {formatFileSize(maxSize || 10 * 1024 * 1024)})
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Selected Files */}
      {showSelectedFilesList && selectedFiles.length > 0 && (
        <div className="space-y-2">
          <h4 className="typography-body-medium text-gray-900">
            File đã chọn ({selectedFiles.length})
          </h4>
          <div className="space-y-2">
            {selectedFiles.map((file, index) => (
              <div
                key={`${file.name}-${index}`}
                className="flex items-center gap-3 p-3 border rounded-lg bg-gray-50"
              >
                {/* File Icon/Preview */}
                <div className="flex-shrink-0">
                  {file.preview ? (
                    <img
                      src={file.preview}
                      alt={file.name}
                      className="w-10 h-10 rounded object-cover"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded bg-gray-200 flex items-center justify-center typography-title-large">
                      {getFileIcon(file)}
                    </div>
                  )}
                </div>

                {/* File Info */}
                <div className="flex-1 min-w-0">
                  <p className="typography-body-medium text-gray-900 truncate">
                    {file.name}
                  </p>
                  <p className="typography-body-small text-gray-500">
                    {formatFileSize(file.size)}
                  </p>

                  {/* Upload Progress */}
                  {file.uploading && (
                    <div className="mt-1">
                      <Progress value={file.progress || 0} className="h-1" />
                      <p className="typography-body-small text-gray-400 mt-1">
                        Đang tải lên... {file.progress || 0}%
                      </p>
                    </div>
                  )}

                  {/* Error */}
                  {file.error && (
                    <div className="flex items-center gap-1 mt-1">
                      <AlertCircle className="w-3 h-3 text-red-500" />
                      <p className="typography-body-small text-red-600">{file.error}</p>
                    </div>
                  )}
                </div>

                {/* Remove Button */}
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeFile(index)}
                  disabled={file.uploading}
                  className="flex-shrink-0 h-8 w-8 p-0 hover:bg-red-50 hover:text-red-600"
                >
                  <X className="w-4 h-4" />
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
