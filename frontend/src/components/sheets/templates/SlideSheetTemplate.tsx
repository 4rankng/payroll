import React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetClose } from "@/components/ui/sheet";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useIsMobile } from '@/hooks/useBreakpoint';
import type { SlideSheetTemplateProps } from "./types";

export function SlideSheetTemplate({
  isOpen,
  onClose,
  title,
  description,
  avatar,
  children,
  footer,
  headerActions,
  size = 'default',
  className,
  compact = false
}: SlideSheetTemplateProps) {
  const isMobile = useIsMobile();

  const getSizeClasses = () => {
    if (isMobile) return "w-full";
    switch (size) {
      case 'large':
        return "w-full sm:w-[600px] md:w-[700px] lg:w-[800px] xl:w-[900px]";
      case 'full':
        return "w-full";
      default:
        return "w-full sm:w-[500px] md:w-[600px] lg:w-[700px]";
    }
  };

  const getInitials = (text: string) => {
    return text
      .split(' ')
      .slice(0, 2)
      .map(n => n[0])
      .join('')
      .toUpperCase();
  };

  return (
    <Sheet open={isOpen} onOpenChange={(open) => {
      if (!open) {
        onClose();
      }
    }}>
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        className={cn(
          "p-0 flex flex-col h-full",
          isMobile && "rounded-t-2xl max-h-[94dvh] shadow-[0_-4px_24px_rgba(0,0,0,0.08)]",
          className || getSizeClasses()
        )}
      >
        {/* Mobile drag handle */}
        {isMobile && (
          <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
        )}

        {/* Header Section */}
        <SheetHeader
          className={cn(
            'space-y-0 flex-shrink-0 border-b',
            compact ? 'px-4 py-2.5' : 'px-4 sm:px-5 py-3'
          )}
        >
          {avatar?.custom ? (
            <div className="flex items-start gap-2">
              <SheetTitle className="sr-only">Chi tiết</SheetTitle>
              <div className="flex-1 min-w-0">{avatar.custom}</div>
              {headerActions && (
                <div className="flex items-center gap-2 flex-shrink-0">
                  {headerActions}
                </div>
              )}
              <SheetClose asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-11 w-11 rounded-full flex-shrink-0 self-start text-muted-foreground hover:text-foreground hover:bg-muted touch-manipulation"
                  aria-label="Đóng"
                >
                  <X className="h-4 w-4" />
                </Button>
              </SheetClose>
            </div>
          ) : (
            <div className="flex items-start justify-between gap-2">
              <div className="flex items-start gap-3 flex-1 min-w-0">
                {avatar && (
                  <Avatar className="h-10 w-10 flex-shrink-0">
                    <AvatarFallback className="bg-primary/10 text-primary">
                      {avatar.fallback ||
                        (avatar.icon ? <avatar.icon className="h-5 w-5" /> : null) ||
                        (avatar.text ? getInitials(avatar.text) : null)}
                    </AvatarFallback>
                  </Avatar>
                )}
                <div className="flex-1 min-w-0">
                  {title && (
                    <SheetTitle className="typography-headline-small break-words">
                      {title}
                    </SheetTitle>
                  )}
                  {description && (
                    <SheetDescription className="typography-body-small text-muted-foreground mt-0.5 break-words">
                      {description}
                    </SheetDescription>
                  )}
                </div>
              </div>
              {headerActions && (
                <div className="flex items-center gap-2 flex-shrink-0">
                  {headerActions}
                </div>
              )}
              <SheetClose asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-11 w-11 rounded-full flex-shrink-0 text-muted-foreground hover:text-foreground hover:bg-muted touch-manipulation"
                  aria-label="Đóng"
                >
                  <X className="h-4 w-4" />
                </Button>
              </SheetClose>
            </div>
          )}
        </SheetHeader>

        {/* Content Section */}
        <div className="flex-1 overflow-hidden">
          <div className={cn(
            'h-full overflow-y-auto overscroll-contain',
            isMobile && 'bg-muted/30',
            compact ? 'px-4 py-3' : 'px-4 sm:px-6 py-4 sm:py-6'
          )}>
            {children}
          </div>
        </div>

        {/* Footer Section */}
        {footer && (
          <div
            className={cn(
              'flex-shrink-0 px-4 sm:px-6 py-2.5 border-t bg-card',
              isMobile && 'pb-[max(10px,calc(10px+env(safe-area-inset-bottom)))]'
            )}
          >
            {footer}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
