import React from 'react';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetClose,
} from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { ErrorState } from '@/components/ui/error-state';
import { MetadataRenderer } from './MetadataRenderer';
import {
  getActionLabel,
  getActionVariant,
  getEntityLabel,
  formatDateTime,
  extractLocation,
  extractLoginIdentifier,
  VARIANT_CLASSES,
} from './utils';
import { useAuditLogDetail } from '@/hooks/api/useAuditLogs';
import { Monitor, Globe, User, Clock, Hash, FileText, MapPin, LogIn, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface AuditLogDetailSheetProps {
  logId: number | null;
  onClose: () => void;
}

function DetailRow({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ElementType;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5 border-b border-border py-2.5 last:border-0 min-[380px]:flex-row min-[380px]:items-start min-[380px]:gap-3">
      <div className="flex shrink-0 items-center gap-2 pt-0.5 min-[380px]:w-32">
        <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground">{label}</span>
      </div>
      <div className="min-w-0 flex-1 break-all text-xs text-foreground">{value ?? '—'}</div>
    </div>
  );
}

export function AuditLogDetailSheet({ logId, onClose }: AuditLogDetailSheetProps) {
  const { data: log, isLoading, isError, refetch } = useAuditLogDetail(logId);
  const isMobile = useIsMobile();
  const variant = log ? getActionVariant(log.action) : 'gray';

  const location = log ? extractLocation(log.metadata) : null;
  const loginIdentifier = (log?.action === 'LOGIN') ? extractLoginIdentifier(log.metadata) : null;

  return (
    <Sheet open={logId !== null} onOpenChange={(open) => { if (!open) onClose(); }}>
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        className={cn(
          "w-full overflow-y-auto p-4 sm:max-w-lg",
          isMobile && "max-h-[92dvh] rounded-t-2xl pb-[calc(env(safe-area-inset-bottom)+1rem)]",
        )}
      >
        <SheetHeader className="pb-4 border-b border-border">
          <div className="flex items-center justify-between gap-3">
            <SheetTitle className="text-left text-base font-semibold">
              Chi tiết nhật ký
            </SheetTitle>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-11 w-11 shrink-0" aria-label="Đóng chi tiết nhật ký">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
          {log && (
            <div className="mt-1 flex min-w-0 flex-wrap items-center gap-2">
              <Badge
                variant="outline"
                className={cn('border px-2 py-0.5 text-xs font-semibold', VARIANT_CLASSES[variant])}
              >
                {getActionLabel(log.action)}
              </Badge>
              <span className="min-w-0 break-words text-xs text-muted-foreground">
                {getEntityLabel(log.entity_type)}
                {log.entity_id ? ` #${log.entity_id}` : ''}
              </span>
            </div>
          )}
        </SheetHeader>

        {isLoading && (
          <div className="space-y-3 pt-4">
            {Array.from({ length: 7 }).map((_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        )}

        {isError && <div role="alert"><ErrorState onRetry={() => void refetch()} /></div>}

        {!isError && log && (
          <div className="pt-4 space-y-5">
            {/* Core fields */}
            <div className="rounded-lg border border-border bg-muted/20 px-3">
              <DetailRow icon={Hash} label="ID" value={`#${log.id}`} />

              {/* User: fullname + username */}
              <DetailRow
                icon={User}
                label="Người dùng"
                value={
                  <div className="flex flex-col gap-0.5">
                    {log.user_fullname && (
                      <span className="font-medium">{log.user_fullname}</span>
                    )}
                    {log.user_username && (
                      <span className="text-muted-foreground">@{log.user_username}</span>
                    )}
                    {!log.user_fullname && !log.user_username && (
                      <span className="text-muted-foreground">#{log.user_id}</span>
                    )}
                  </div>
                }
              />

              <DetailRow icon={Clock} label="Thời gian" value={formatDateTime(log.created_at)} />
              <DetailRow icon={FileText} label="Nội dung" value={log.message} />

              {/* Login identifier — what the user actually typed */}
              {loginIdentifier && loginIdentifier !== log.user_username && (
                <DetailRow
                  icon={LogIn}
                  label="Đăng nhập bằng"
                  value={
                    <span className="font-medium text-amber-700">{loginIdentifier}</span>
                  }
                />
              )}

              {log.ip_address && (
                <DetailRow icon={Globe} label="IP" value={log.ip_address} />
              )}

              {/* Geo location */}
              {location && (
                <DetailRow
                  icon={MapPin}
                  label="Vị trí"
                  value={[location.city, location.region, location.country].filter(Boolean).join(', ')}
                />
              )}

              {(log.browser || log.platform) && (
                <DetailRow
                  icon={Monitor}
                  label="Thiết bị"
                  value={[log.browser, log.platform].filter(Boolean).join(' · ')}
                />
              )}
            </div>

            {/* Metadata */}
            <MetadataRenderer
              metadata={log.metadata}
              action={log.action}
              entityType={log.entity_type}
            />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
