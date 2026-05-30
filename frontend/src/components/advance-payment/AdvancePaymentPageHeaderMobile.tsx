import { useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { MoreHorizontal, FileSpreadsheet, Users } from "lucide-react";

interface AdvancePaymentPageHeaderMobileProps {
  onImportPayroll: () => void;
  onViewEmployees: () => void;
  overflowContent?: ReactNode;
  /** Render-prop variant — receives close() so overflow actions can dismiss the sheet. */
  renderOverflowContent?: (close: () => void) => ReactNode;
}

export function AdvancePaymentPageHeaderMobile({
  onImportPayroll,
  onViewEmployees,
  overflowContent,
  renderOverflowContent,
}: AdvancePaymentPageHeaderMobileProps) {
  const [open, setOpen] = useState(false);
  const close = () => setOpen(false);

  return (
    <div className="flex items-center justify-between gap-2">
      <div className="min-w-0">
        <h1 className="text-base font-semibold text-foreground tracking-tight leading-snug">Ứng lương</h1>
        <p className="text-xs text-muted-foreground mt-0.5">
          Quản lý yêu cầu ứng lương
        </p>
      </div>

      <div className="flex items-center gap-2 shrink-0">
        <Button
          onClick={onImportPayroll}
          size="sm"
          className="h-9 px-3 touch-manipulation"
        >
          <FileSpreadsheet className="h-4 w-4 mr-1" />
          Nhập
        </Button>

        <Button
          variant="outline"
          size="sm"
          className="h-9 px-3 touch-manipulation"
          onClick={onViewEmployees}
          aria-label="Danh sách nhân viên"
        >
          <Users className="h-4 w-4" />
        </Button>

        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="h-9 w-9 touch-manipulation"
              aria-label="Tùy chọn"
            >
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </SheetTrigger>
          <SheetContent side="top" className="h-auto px-4 pb-3" style={{ paddingTop: "max(16px, calc(16px + env(safe-area-inset-top)))" }}>
            <SheetHeader className="mb-2">
              <SheetTitle>Tùy chọn</SheetTitle>
            </SheetHeader>

            <div className="space-y-0.5">
              {renderOverflowContent ? renderOverflowContent(close) : overflowContent}
            </div>

            <div className="flex justify-center pt-3 pb-1">
              <div className="w-10 h-1 rounded-full bg-muted-foreground/25" />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  );
}
