import { useState, type ReactNode } from "react";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
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
    <MobilePageHeader
      title="Ứng lương"
      subtitle="Quản lý yêu cầu ứng lương"
      icon={FileSpreadsheet}
      sticky={false}
      bordered={false}
      className="bg-transparent px-0 pb-0 supports-[backdrop-filter]:bg-transparent"
      actions={
        <div className="flex items-center gap-1.5 shrink-0">
          <Button
            onClick={onImportPayroll}
            size="sm"
            className="h-8 px-2.5 text-xs touch-manipulation"
          >
            <FileSpreadsheet className="h-3.5 w-3.5 mr-1" />
            Nhập
          </Button>

          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8 touch-manipulation"
            onClick={onViewEmployees}
            aria-label="Danh sách nhân viên"
          >
            <Users className="h-3.5 w-3.5" />
          </Button>

          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8 touch-manipulation"
                aria-label="Tùy chọn"
              >
                <MoreHorizontal className="h-3.5 w-3.5" />
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
      }
    />
  );
}
