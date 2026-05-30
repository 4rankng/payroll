import React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription, SheetClose } from "@/components/ui/sheet";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
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
  const getSizeClasses = () => {
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
        side="right"
        className={cn(
          "p-0 flex flex-col h-full",
          className || getSizeClasses()
        )}
      >
        {/* Header Section */}
        <SheetHeader
          className={cn(
            'space-y-0 flex-shrink-0 border-b',
            compact ? 'px-4 py-2.5' : 'px-4 sm:px-5 py-3'
          )}
          style={{ paddingTop: `max(${compact ? '10px' : '12px'}, calc(${compact ? '10px' : '12px'} + env(safe-area-inset-top)))` }}
        >
          {avatar?.custom ? (
            <div className="flex items-center gap-2">
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
                  className="h-8 w-8 rounded-full flex-shrink-0 self-start text-muted-foreground hover:text-foreground hover:bg-muted"
                  aria-label="Đóng"
                >
                  <X className="h-4 w-4" />
                </Button>
              </SheetClose>
            </div>
          ) : (
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-3 flex-1 min-w-0">
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
                    <SheetTitle className="typography-headline-small truncate">
                      {title}
                    </SheetTitle>
                  )}
                  {description && (
                    <SheetDescription className="typography-body-small text-muted-foreground mt-0.5">
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
                  className="h-8 w-8 rounded-full flex-shrink-0 text-muted-foreground hover:text-foreground hover:bg-muted"
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
            'h-full overflow-y-auto',
            compact ? 'px-4 py-3' : 'px-4 sm:px-6 py-4 sm:py-6'
          )}>
            {children}
          </div>
        </div>

        {/* Footer Section */}
        {footer && (
          <div
            className="flex-shrink-0 px-4 sm:px-6 py-2 border-t bg-background"
            style={{ paddingBottom: `max(8px, calc(8px + env(safe-area-inset-bottom)))` }}
          >
            {footer}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
