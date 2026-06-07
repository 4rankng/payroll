import { useCallback, useRef } from "react";
import { UploadCloud, FileSpreadsheet, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface FileDropZoneProps {
  file: File | null;
  onFileChange: (file: File | null) => void;
  accept?: string;
  inputId?: string;
}

export const FileDropZone = ({
  file,
  onFileChange,
  accept = ".xlsx,.xls",
  inputId = "file-drop-input",
}: FileDropZoneProps) => {
  const inputRef = useRef<HTMLInputElement>(null);

  const handleClick = useCallback(() => {
    inputRef.current?.click();
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.currentTarget.setAttribute("data-drag", "true");
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.currentTarget.removeAttribute("data-drag");
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.currentTarget.removeAttribute("data-drag");
      const droppedFile = e.dataTransfer.files?.[0];
      if (droppedFile) onFileChange(droppedFile);
    },
    [onFileChange],
  );

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onFileChange(e.target.files?.[0] || null);
    },
    [onFileChange],
  );

  const acceptLabel = accept
    .split(",")
    .map((s) => s.trim().toUpperCase())
    .join(", ");

  if (file) {
    return (
      <div className="flex items-center gap-3 px-3 py-2.5 rounded-xl border bg-muted/40">
        <div className="w-8 h-8 rounded-xl bg-primary/10 flex items-center justify-center flex-shrink-0">
          <FileSpreadsheet className="w-4 h-4 text-primary" />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{file.name}</p>
          <p className="text-xs text-muted-foreground">
            {(file.size / 1024).toFixed(1)} KB
          </p>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-7 w-7 flex-shrink-0 text-muted-foreground hover:text-destructive"
          onClick={() => onFileChange(null)}
          aria-label="Xóa file"
        >
          <X className="w-3.5 h-3.5" />
        </Button>
        <input ref={inputRef} id={inputId} type="file" accept={accept} onChange={handleInputChange} className="hidden" />
      </div>
    );
  }

  return (
    <div
      className={cn(
        "relative border border-dashed rounded-xl px-4 py-6 text-center cursor-pointer transition-colors",
        "border-border hover:border-primary/50 hover:bg-muted/30",
        "data-[drag]:border-primary data-[drag]:bg-primary/5"
      )}
      onClick={handleClick}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <input ref={inputRef} id={inputId} type="file" accept={accept} onChange={handleInputChange} className="hidden" />
      <div className="flex flex-col items-center gap-2">
        <div className="w-9 h-9 rounded-xl bg-muted flex items-center justify-center">
          <UploadCloud className="w-[18px] h-[18px] text-muted-foreground" />
        </div>
        <div className="space-y-0.5">
          <p className="text-sm font-medium">
            Kéo thả file vào đây{" "}
            <span className="text-primary">hoặc chọn file</span>
          </p>
          <p className="text-xs text-muted-foreground">{acceptLabel}</p>
        </div>
      </div>
    </div>
  );
};
