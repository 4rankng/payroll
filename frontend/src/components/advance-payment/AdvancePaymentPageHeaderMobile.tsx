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
    <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-primary/10 bg-primary/[0.07] text-primary/75 shadow-sm">
        <FileSpreadsheet className="h-[18px] w-[18px]" strokeWidth={2} />
      </div>

      <div className="min-w-0">
        <h1 className="font-display text-[22px] font-extrabold leading-none tracking-normal text-slate-950">
          Ứng lương
        </h1>
        <p className="mt-1 text-[13px] font-medium leading-snug text-slate-500">
          Quản lý yêu cầu ứng lương
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-1.5">
        <Button
          onClick={onImportPayroll}
          size="sm"
          className="h-9 min-h-9 rounded-xl px-3 text-[13px] font-semibold shadow-sm touch-manipulation"
        >
          <FileSpreadsheet className="mr-1.5 h-3.5 w-3.5" />
          Nhập
        </Button>

        <Sheet open={open} onOpenChange={setOpen}>
          <SheetTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className="h-9 min-h-9 w-9 min-w-9 rounded-xl border-slate-300 bg-white shadow-sm touch-manipulation"
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
              <Button
                variant="ghost"
                className="h-auto min-h-11 w-full justify-start gap-3 rounded-xl px-2 py-3 text-sm font-medium"
                onClick={() => {
                  onViewEmployees();
                  close();
                }}
              >
                <Users className="h-5 w-5 shrink-0 text-muted-foreground" />
                Danh sách nhân viên
              </Button>
              {renderOverflowContent ? renderOverflowContent(close) : overflowContent}
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  );
}
