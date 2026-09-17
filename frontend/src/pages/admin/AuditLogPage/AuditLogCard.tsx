import { format } from "date-fns";
import React from 'react';
import { Badge } from '@/components/ui/badge';
import type { BackendAuditLog } from '@/types/api/audit.types';
import {
  getActionLabel,
  getActionVariant,
  getEntityLabel,
  extractLocation,
  extractLoginIdentifier,
  VARIANT_CLASSES,
} from './utils';
import { Globe, Monitor, MapPin, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

interface AuditLogCardProps {
  log: BackendAuditLog;
  onClick: (id: number) => void;
  compact?: boolean;
}

const ROLE_LABELS: Record<string, string> = {
  admin: 'Quản trị',
  partner: 'Quản lý',
  employee: 'Nhân viên',
};

function getRoleLabel(role: string): string {
  return ROLE_LABELS[role] ?? 'Người dùng';
}

function resolveUser(log: BackendAuditLog): { fullname: string; username: string } {
  const fullname = log.user_fullname?.trim() ?? '';
  const username = log.user_username?.trim() ?? '';
  if (!fullname) {
    const fromMsg = log.message?.split(' đã ')?.[0]?.trim() ?? '';
    return { fullname: fromMsg.length > 0 && fromMsg.length < 60 ? fromMsg : '', username };
  }
  return { fullname, username };
}

export function AuditLogCard({ log, onClick, compact = false }: AuditLogCardProps) {
  const variant = getActionVariant(log.action);
  const location = extractLocation(log.metadata);
  const loginId = log.action === 'LOGIN' ? extractLoginIdentifier(log.metadata) : null;
  const showLoginId = loginId && loginId !== log.user_username?.trim();
  const { fullname, username } = resolveUser(log);

  const date = new Date(log.created_at);
  const dateStr = format(date, 'dd/MM/yyyy');
  const timeStr = date.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  const compactTime = `${format(date, 'dd/MM')} · ${format(date, 'HH:mm')}`;

  // Build meta chips for the footer row
  const metaChips: { icon: React.ReactNode; label: string }[] = [];
  if (log.ip_address) metaChips.push({ icon: <Globe className="w-3 h-3" />, label: log.ip_address });
  if (location) metaChips.push({ icon: <MapPin className="w-3 h-3" />, label: [location.city, location.country].filter(Boolean).join(', ') });
  if (log.browser || log.platform) metaChips.push({ icon: <Monitor className="w-3 h-3" />, label: [log.browser, log.platform].filter(Boolean).join(' · ') });

  return (
    <button
      type="button"
      onClick={() => onClick(log.id)}
      className={cn(
        'group w-full overflow-hidden border text-left transition-all duration-150 hover:border-primary/20 hover:bg-muted/30 active:bg-muted/50',
        compact
          ? 'rounded-2xl border-slate-200/90 bg-white shadow-[0_8px_20px_-18px_rgba(15,23,42,0.34)]'
          : 'rounded-xl border-border bg-card shadow-sm',
      )}
    >
      {/* Top bar: action badge + role | datetime + chevron */}
      <div className={cn(
        'flex flex-col gap-2 border-b border-border/60 px-3 pb-2.5 pt-3 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between',
        compact && 'flex-row items-center justify-between gap-2 border-slate-100 bg-slate-50/80 px-3.5 py-2.5',
      )}>
        <div className="flex min-w-0 items-center gap-2">
          <Badge
            variant="outline"
            className={cn('shrink-0 border px-2 py-0.5 text-xs font-semibold', compact && 'rounded-md text-xs', VARIANT_CLASSES[variant])}
          >
            {getActionLabel(log.action)}
          </Badge>
          <span className={cn('min-w-0 truncate text-xs text-muted-foreground', compact && 'font-medium text-slate-500')}>
            {getRoleLabel(log.user_role)}
          </span>
        </div>
        <div className={cn(
          'flex min-w-0 items-center justify-between gap-1.5 min-[380px]:shrink-0 min-[380px]:justify-end',
          compact && 'shrink-0 justify-end',
        )}>
          <span className={cn('min-w-0 text-xs text-muted-foreground min-[380px]:whitespace-nowrap', compact && 'text-xs font-medium text-slate-600')}>
            {compact ? compactTime : `${dateStr} ${timeStr}`}
          </span>
          <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground/40 transition-colors group-hover:text-muted-foreground" />
        </div>
      </div>

      {/* Body */}
      <div className={cn('space-y-2 px-3 pb-3 pt-2.5', compact && 'space-y-2.5 px-3.5 pb-3.5 pt-3')}>
        {/* User name + username */}
        <div>
          {fullname
            ? <p className={cn('break-words text-sm font-semibold leading-tight text-foreground', compact && 'text-[15px]')}>{fullname}</p>
            : <p className="text-sm text-muted-foreground">#{log.user_id}</p>
          }
          {username && (
            <p className={cn('break-all text-xs leading-tight text-muted-foreground', compact && 'mt-0.5 text-slate-600')}>@{username}</p>
          )}
        </div>

        {/* Message */}
        <p className={cn('break-words text-sm leading-snug text-foreground', compact && 'text-slate-600')}>
          {log.message}
        </p>

        {/* Login identifier */}
        {showLoginId && (
          <p className={cn('break-words text-xs font-medium text-amber-600', compact && 'rounded-lg bg-amber-50 px-2.5 py-2 text-amber-700')}>
            Đăng nhập bằng: {loginId}
          </p>
        )}

        {/* Meta footer: IP · location · device as a single wrapping row of chips */}
        {metaChips.length > 0 && (
          <div className={cn('flex flex-wrap items-center gap-x-3 gap-y-1 pt-0.5', compact && 'border-t border-slate-100 pt-2.5')}>
            {metaChips.map((chip, i) => (
              <span key={i} className={cn('inline-flex min-w-0 items-center gap-1 text-xs text-muted-foreground', compact && 'text-slate-600')}>
                {chip.icon}
                <span className="min-w-0 break-all">{chip.label}</span>
              </span>
            ))}
          </div>
        )}
      </div>
    </button>
  );
}

export function AuditLogCardSkeleton() {
  return (
    <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden animate-pulse">
      <div className="flex items-center justify-between px-4 pt-3.5 pb-2.5 border-b border-border/60">
        <div className="flex items-center gap-2">
          <div className="h-5 w-16 bg-muted rounded-full" />
          <div className="h-4 w-20 bg-muted rounded" />
        </div>
        <div className="h-4 w-28 bg-muted rounded" />
      </div>
      <div className="px-4 py-3 space-y-2.5">
        <div className="flex justify-between">
          <div className="space-y-1.5">
            <div className="h-4 w-36 bg-muted rounded" />
            <div className="h-3 w-24 bg-muted rounded" />
          </div>
          <div className="space-y-1.5 items-end flex flex-col">
            <div className="h-3 w-24 bg-muted rounded" />
            <div className="h-3 w-20 bg-muted rounded" />
          </div>
        </div>
        <div className="h-4 w-full bg-muted rounded" />
        <div className="h-4 w-3/4 bg-muted rounded" />
      </div>
    </div>
  );
}
