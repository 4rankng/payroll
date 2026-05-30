import React, { useState, useCallback, useMemo } from "react";
import { ShieldAlert, Clock, AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useFailedLogins } from "@/hooks/api/useSystemHealth";
import { SectionLabel } from "./SectionLabel";
import { FailedLoginDetailDialog } from "./FailedLoginDetailDialog";
import type { LoginIdentifierSummary } from "@/types/api/system-health.types";

function getBadgeVariant(count: number): "destructive" | "warning" | "secondary" {
  if (count > 10) return "destructive";
  if (count >= 3) return "warning";
  return "secondary";
}

function formatLastSeen(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 60) return `${diffMin} phút trước`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour} giờ trước`;
  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} ngày trước`;
}

function BadgeItem({
  summary,
  days,
  onClick,
}: {
  summary: LoginIdentifierSummary;
  days: number;
  onClick: (identifier: string) => void;
}) {
  const handleClick = useCallback(() => {
    onClick(summary.identifier);
  }, [onClick, summary.identifier]);

  const variant = useMemo(() => getBadgeVariant(summary.count), [summary.count]);

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "group relative inline-flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm",
        "transition-all hover:shadow-sm hover:scale-[1.02] active:scale-[0.98]",
        "cursor-pointer text-left",
        variant === "destructive" && "border-red-200 bg-red-50 hover:bg-red-100",
        variant === "warning" && "border-amber-200 bg-amber-50 hover:bg-amber-100",
        variant === "secondary" && "border-slate-200 bg-slate-50 hover:bg-slate-100",
      )}
    >
      <span className="font-medium text-foreground truncate max-w-[200px]">
        {summary.identifier}
      </span>
      <Badge variant={variant} className="text-[10px] tabular-nums ml-1 shrink-0">
        {summary.count}×
      </Badge>
      <span className="text-[10px] text-muted-foreground ml-1 shrink-0">
        {formatLastSeen(summary.last_seen)}
      </span>
    </button>
  );
}

interface FailedLoginBadgesProps {
  days: number;
}

export function FailedLoginBadges({ days }: FailedLoginBadgesProps) {
  const { data, isLoading } = useFailedLogins(days);
  const [selectedIdentifier, setSelectedIdentifier] = useState<string | null>(null);

  const handleBadgeClick = useCallback((identifier: string) => {
    setSelectedIdentifier(identifier);
  }, []);

  const handleDialogClose = useCallback(() => {
    setSelectedIdentifier(null);
  }, []);

  const summaryStats = useMemo(() => {
    if (!data?.summary) return null;
    const totalAttempts = data.summary.reduce((s, item) => s + item.count, 0);
    const uniqueIdentifiers = data.summary.length;
    return { totalAttempts, uniqueIdentifiers };
  }, [data?.summary]);

  if (isLoading) {
    return (
      <section>
        <SectionLabel icon={ShieldAlert}>Đăng Nhập Thất Bại</SectionLabel>
        <div className="animate-pulse flex gap-2 flex-wrap">
          {[1, 2, 3].map((i) => (
            <div key={i} className="h-9 w-40 rounded-lg bg-muted" />
          ))}
        </div>
      </section>
    );
  }

  if (!data?.summary?.length) {
    return (
      <section>
        <SectionLabel icon={ShieldAlert}>Đăng Nhập Thất Bại</SectionLabel>
        <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
          <Clock className="h-4 w-4" />
          <span>Không có lần đăng nhập thất bại nào trong {days} ngày qua</span>
        </div>
      </section>
    );
  }

  return (
    <section>
      <SectionLabel icon={ShieldAlert}>Đăng Nhập Thất Bại</SectionLabel>

      {summaryStats && (
        <div className="flex items-center gap-4 mb-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <AlertTriangle className="h-3 w-3" />
            {summaryStats.totalAttempts} lần thất bại
          </span>
          <span>{summaryStats.uniqueIdentifiers} tài khoản</span>
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {data.summary.map((item) => (
          <BadgeItem
            key={item.identifier}
            summary={item}
            days={days}
            onClick={handleBadgeClick}
          />
        ))}
      </div>

      <FailedLoginDetailDialog
        identifier={selectedIdentifier}
        days={days}
        onClose={handleDialogClose}
      />
    </section>
  );
}
