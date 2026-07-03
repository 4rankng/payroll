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
      className="bg-white px-0 pb-0"
      actions={
        <div className="flex items-center gap-1.5 shrink-0">
          <Button
            onClick={onImportPayroll}
            size="sm"
            className="min-h-11 rounded-xl px-3 text-xs font-semibold shadow-sm touch-manipulation"
          >
            <FileSpreadsheet className="h-3.5 w-3.5 mr-1" />
            Nhập
          </Button>

          <Button
            variant="outline"
            size="icon"
            className="h-11 w-11 rounded-xl border-slate-300 bg-white shadow-sm touch-manipulation"
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
                className="h-11 w-11 rounded-xl border-slate-300 bg-white shadow-sm touch-manipulation"
                aria-label="Tùy chọn"
              >
                <MoreHorizontal className="h-3.5 w-3.5" />
              </Button>
            </SheetTrigger>
            <SheetContent
              side="bottom"
              className="h-auto rounded-t-3xl border-slate-200 bg-white px-4 pb-[max(1rem,calc(1rem+env(safe-area-inset-bottom)))] pt-3 shadow-[0_-20px_48px_rgba(15,23,42,0.16)]"
            >
              <div className="mx-auto mb-3 h-1 w-11 rounded-full bg-slate-300" />
              <SheetHeader className="mb-3 text-left">
                <SheetTitle>Tùy chọn</SheetTitle>
              </SheetHeader>

              <div className="space-y-1">
                {renderOverflowContent ? renderOverflowContent(close) : overflowContent}
              </div>
            </SheetContent>
          </Sheet>
        </div>
      }
    />
  );
}
