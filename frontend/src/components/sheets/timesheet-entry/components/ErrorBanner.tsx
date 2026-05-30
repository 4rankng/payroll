import { memo } from "react";
import { AlertTriangle } from "lucide-react";

interface ErrorBannerProps {
  errors: string[];
  className?: string;
}

export const ErrorBanner = memo(({ errors, className }: ErrorBannerProps) => {
  if (!errors.length) return null;
  return (
    <div
      className={`flex items-start gap-2 px-3 py-2 bg-red-50 rounded-xl border border-red-200 ${className ?? ""}`}
    >
      <AlertTriangle className="h-3.5 w-3.5 text-red-500 shrink-0 mt-0.5" />
      <div className="space-y-0.5">
        {errors.map((err, i) => (
          <p key={i} className="text-xs font-medium text-red-600">
            {err}
          </p>
        ))}
      </div>
    </div>
  );
});

ErrorBanner.displayName = "ErrorBanner";
