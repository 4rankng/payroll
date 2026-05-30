import { useEffect, useRef, useState } from 'react';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

type LongTextProps = {
  children: React.ReactNode;
  className?: string;
  contentClassName?: string;
};

function checkOverflow(element: HTMLElement | null): boolean {
  if (!element) return false;
  return element.scrollHeight > element.offsetHeight || element.scrollWidth > element.offsetWidth;
}

export function LongText({
  children,
  className = '',
  contentClassName = '',
}: LongTextProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [isOverflown, setIsOverflown] = useState(false);

  useEffect(() => {
    if (checkOverflow(ref.current)) {
      setIsOverflown(true);
      return;
    }
    setIsOverflown(false);
  }, [children]);

  if (!isOverflown) {
    return (
      <div ref={ref} className={className}>
        {children}
      </div>
    );
  }

  return (
    <>
      {/* Desktop: Tooltip */}
      <div className="hidden sm:block">
        <TooltipProvider>
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <div ref={ref} className={className}>
                {children}
              </div>
            </TooltipTrigger>
            <TooltipContent className={contentClassName} side="top" align="start">
              <div className="max-w-xs whitespace-normal break-words">
                {children}
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      {/* Mobile: Popover */}
      <div className="block sm:hidden">
        <Popover>
          <PopoverTrigger asChild>
            <div ref={ref} className={`cursor-pointer ${className}`}>
              {children}
            </div>
          </PopoverTrigger>
          <PopoverContent className={contentClassName} side="top" align="start">
            <div className="whitespace-normal break-words">
              {children}
            </div>
          </PopoverContent>
        </Popover>
      </div>
    </>
  );
}