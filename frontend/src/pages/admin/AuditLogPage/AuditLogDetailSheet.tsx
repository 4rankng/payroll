import React from 'react';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/use-mobile';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
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
import { Monitor, Globe, User, Clock, Hash, FileText, MapPin, LogIn } from 'lucide-react';
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
    <div className="flex items-start gap-3 py-2.5 border-b border-border last:border-0">
      <div className="flex items-center gap-2 w-32 shrink-0 pt-0.5">
        <Icon className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
        <span className="text-xs text-muted-foreground font-medium">{label}</span>
      </div>
      <div className="text-xs text-foreground break-all flex-1">{value ?? '—'}</div>
    </div>
  );
}

export function AuditLogDetailSheet({ logId, onClose }: AuditLogDetailSheetProps) {
  const { data: log, isLoading } = useAuditLogDetail(logId);
  const isMobile = useIsMobile();
  const variant = log ? getActionVariant(log.action) : 'gray';

  const location = log ? extractLocation(log.metadata) : null;
  const loginIdentifier = (log?.action === 'LOGIN') ? extractLoginIdentifier(log.metadata) : null;

  return (
    <Sheet open={logId !== null} onOpenChange={(open) => { if (!open) onClose(); }}>
      <SheetContent side={isMobile ? "bottom" : "right"} className="w-full sm:max-w-lg overflow-y-auto">
        <SheetHeader className="pb-4 border-b border-border">
          <SheetTitle className="text-base font-semibold">
            Chi tiết Audit Log
          </SheetTitle>
          {log && (
            <div className="flex items-center gap-2 mt-1">
              <Badge
                variant="outline"
                className={cn('text-xs font-semibold border px-2 py-0.5', VARIANT_CLASSES[variant])}
              >
                {getActionLabel(log.action)}
              </Badge>
              <span className="text-xs text-muted-foreground">
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

        {log && (
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
                    <span className="font-medium text-amber-600">{loginIdentifier}</span>
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
            <MetadataRenderer metadata={log.metadata} />
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
