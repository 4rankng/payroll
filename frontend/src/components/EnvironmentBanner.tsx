import React from "react";
import { cn } from "@/lib/utils";

interface EnvironmentBannerProps {
  headline: string;
  detail?: string;
}

const EnvironmentBannerComponent = ({ headline, detail }: EnvironmentBannerProps) => (
  <div
    className={cn(
      "sticky top-0 z-50 w-full bg-amber-100/95 border-b border-amber-200",
      "shadow-soft min-h-[30px]"
    )}
    style={{ paddingTop: 'env(safe-area-inset-top, 0px)' }}
    role="status"
    aria-live="polite"
    aria-label="Thông báo môi trường phát triển"
  >
    <div className="mx-auto flex max-w-6xl items-center justify-center gap-3 px-4 py-[4px] text-sm font-semibold text-foreground">
      <span className="tracking-wide">{headline}</span>
      {detail ? (
        <span className="text-xs font-normal text-slate-800/90">{detail}</span>
      ) : null}
    </div>
  </div>
);

export const EnvironmentBanner = React.memo(EnvironmentBannerComponent);
EnvironmentBanner.displayName = "EnvironmentBanner";
