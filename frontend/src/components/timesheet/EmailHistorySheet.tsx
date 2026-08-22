import { memo, useMemo, useCallback, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Mail, Users, Loader2, AlertCircle, X, Clock, User, Send } from 'lucide-react';
import { Sheet, SheetClose, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { useEmailHistory } from '@/hooks/api/useEmails';
import { useSenderProfiles } from '@/hooks/useSenderProfiles';
import { EmptyState } from '@/components/shared/EmptyState';
import type { EmailHistoryRecord } from '@/types/api/email.types';

interface EmailHistorySheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const EMAIL_TYPE_LABELS: Record<string, string> = {
  payroll_report: 'Báo cáo sao kê',
  generic: 'Email',
};

const EMAIL_CHANNEL_LABELS: Record<string, string> = {
  email: 'Email',
  sms: 'SMS',
  push: 'Push',
  webhook: 'Webhook',
};

const EMAIL_BODY_PREVIEW_LIMIT = 240;

export const EmailHistorySheet = memo(function EmailHistorySheet({
  open,
  onOpenChange,
}: EmailHistorySheetProps) {
  const isMobile = useIsMobile();
  const [expandedEmailIds, setExpandedEmailIds] = useState<Set<EmailHistoryRecord['id']>>(
    () => new Set()
  );

  const {
    data,
    isLoading,
    isError,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useEmailHistory({
    enabled: open,
    pageSize: 20,
  });

  const historyRecords = useMemo<EmailHistoryRecord[]>(() => {
    if (!data?.pages) {
      return [];
    }
    return data.pages.flatMap((page) => page.data ?? []);
  }, [data]);

  const isEmpty = useMemo(() => !isLoading && historyRecords.length === 0, [historyRecords.length, isLoading]);

  const senderIdsNeedingLookup = useMemo(() => {
    const ids = new Set<number>();
    historyRecords.forEach((record) => {
      if (typeof record.senderId !== 'number' || record.senderId <= 0) {
        return;
      }
      const hasInlineName =
        (typeof record.senderName === 'string' && record.senderName.trim().length > 0) ||
        (typeof record.senderUsername === 'string' && record.senderUsername.trim().length > 0);
      if (!hasInlineName) {
        ids.add(record.senderId);
      }
    });
    return Array.from(ids);
  }, [historyRecords]);

  const {
    data: senderLookup,
    isLoading: isSenderLoading,
    isFetching: isSenderFetching,
  } = useSenderProfiles(senderIdsNeedingLookup, {
    enabled: open && senderIdsNeedingLookup.length > 0,
  });

  const isSenderDataLoading = useMemo(
    () => (isSenderLoading || isSenderFetching) && senderIdsNeedingLookup.length > 0,
    [isSenderLoading, isSenderFetching, senderIdsNeedingLookup.length]
  );

  const senderLookupTargets = useMemo(() => new Set(senderIdsNeedingLookup), [senderIdsNeedingLookup]);

  const formatSentAt = useCallback((value: string) => {
    try {
      return format(new Date(value), 'dd/MM/yyyy HH:mm', { locale: vi });
    } catch {
      return value;
    }
  }, []);

  const getTypeLabel = useCallback((type: string) => {
    return EMAIL_TYPE_LABELS[type] ?? type;
  }, []);

  const getChannelLabel = useCallback((channel: string) => {
    return EMAIL_CHANNEL_LABELS[channel] ?? channel;
  }, []);

  const getSenderLabel = useCallback((record: EmailHistoryRecord) => {
    const inlineName = typeof record.senderName === 'string' ? record.senderName.trim() : '';
    const inlineUsername = typeof record.senderUsername === 'string' ? record.senderUsername.trim() : '';

    if (inlineName && inlineUsername) {
      return `${inlineName} (${inlineUsername})`;
    }

    if (inlineName) {
      return inlineName;
    }

    if (inlineUsername) {
      return inlineUsername;
    }

    if (!record.senderId) {
      return 'Không rõ người gửi';
    }

    const sender = senderLookup?.get(record.senderId);
    if (!sender) {
      return null;
    }

    const lookupName =
      typeof sender.fullname === 'string' && sender.fullname.trim().length > 0
        ? sender.fullname.trim()
        : '';
    const lookupUsername =
      typeof sender.username === 'string' && sender.username.trim().length > 0
        ? sender.username.trim()
        : '';
    const lookupEmail =
      typeof sender.email === 'string' && sender.email.trim().length > 0
        ? sender.email.trim()
        : '';

    if (lookupName && lookupUsername) {
      return `${lookupName} (${lookupUsername})`;
    }

    if (lookupName) {
      return lookupName;
    }

    if (lookupUsername) {
      return lookupUsername;
    }

    return lookupEmail || null;
  }, [senderLookup]);

  const handleLoadMore = useCallback(() => {
    if (hasNextPage) {
      void fetchNextPage();
    }
  }, [fetchNextPage, hasNextPage]);

  const handleRetry = useCallback(() => {
    void refetch();
  }, [refetch]);

  const toggleEmailBody = useCallback((id: EmailHistoryRecord['id']) => {
    setExpandedEmailIds((previous) => {
      const next = new Set(previous);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const renderHistoryItem = useCallback((record: EmailHistoryRecord) => {
    const sentAtLabel = formatSentAt(record.sentAt);
    const typeLabel = getTypeLabel(record.type);
    const channelLabel = getChannelLabel(record.channel);
    const senderLabel = getSenderLabel(record);
    const needsLookup = record.senderId ? senderLookupTargets.has(record.senderId) : false;
    const showSenderSkeleton = needsLookup && senderLabel === null && isSenderDataLoading;
    const senderValue = senderLabel ?? 'Không rõ người gửi';
    const normalizedBody = (record.body ?? '').trim();
    const isExpanded = expandedEmailIds.has(record.id);
    const shouldTruncate = !isExpanded && normalizedBody.length > EMAIL_BODY_PREVIEW_LIMIT;
    const displayBody = shouldTruncate
      ? `${normalizedBody.slice(0, EMAIL_BODY_PREVIEW_LIMIT)}…`
      : normalizedBody;
    const bodySectionId = `email-body-${record.id}`;
    const handleCardToggle = () => {
      toggleEmailBody(record.id);
    };
    const handleCardKeyDown = (event: KeyboardEvent<HTMLElement>) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        toggleEmailBody(record.id);
      }
    };

    return (
      <article
        key={record.id}
        className="mt-5 cursor-pointer rounded-2xl border border-border bg-card p-5 shadow-sm transition-all duration-200 first:mt-0 hover:-translate-y-[1px] focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
        tabIndex={0}
        aria-label={`Email ${record.subject}`}
        role="button"
        aria-expanded={isExpanded}
        aria-controls={bodySectionId}
        onClick={handleCardToggle}
        onKeyDown={handleCardKeyDown}
      >
        <header className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex min-w-0 flex-1 items-start gap-3">
            <span className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary/10">
              <Mail className="h-5 w-5 text-primary" aria-hidden="true" />
            </span>
            <div className="min-w-0 flex-1">
              <h3 className="truncate typography-body-large font-semibold text-foreground" title={record.subject}>
                {record.subject}
              </h3>
              <div className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground" aria-label={`Gửi lúc ${sentAtLabel}`}>
                <Clock className="h-3.5 w-3.5" aria-hidden="true" />
                <span>{sentAtLabel}</span>
              </div>
            </div>
          </div>
          <div className="flex flex-shrink-0 flex-wrap items-center gap-2 text-sm">
            <Badge variant="secondary" className="typography-label-small">
              {typeLabel}
            </Badge>
            <Badge variant="outline" className="typography-label-small">
              {channelLabel}
            </Badge>
          </div>
        </header>

        <div className="mt-4 grid grid-cols-1 gap-y-3 pt-2 sm:grid-cols-2 sm:gap-x-6">
          <div className="flex items-center gap-3 text-sm" aria-label={`Người gửi: ${senderValue}`}>
            <User className="h-4 w-4 flex-shrink-0 text-primary" aria-hidden="true" />
            {showSenderSkeleton ? (
              <Skeleton className="h-4 w-32" aria-label="Đang tải thông tin người gửi" />
            ) : (
              <span className="font-medium text-foreground">{senderValue}</span>
            )}
          </div>

          <div className="flex items-start gap-3 text-sm" aria-label="Người nhận">
            <Send className="h-4 w-4 flex-shrink-0 text-primary mt-0.5" aria-hidden="true" />
            <div className="flex flex-wrap items-center gap-1.5">
              {!record.recipients || record.recipients.length === 0 ? (
                <span className="text-foreground">Không có người nhận</span>
              ) : (
                record.recipients.map((recipient, index) => (
                  <Badge key={index} variant="outline" className="font-normal text-muted-foreground">
                    {recipient.name && recipient.name.trim().length > 0
                      ? `${recipient.name} <${recipient.address}>`
                      : recipient.address}
                  </Badge>
                ))
              )}
            </div>
          </div>
        </div>

        <section className="mt-5 space-y-2">
          <div
            id={bodySectionId}
            className="relative overflow-hidden"
          >
            <p className="whitespace-pre-line text-sm leading-relaxed text-foreground">
              {displayBody || 'Không có nội dung email'}
            </p>
            {shouldTruncate && (
              <div
                aria-hidden="true"
                className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-white via-white/90 to-transparent"
              />
            )}
          </div>
        </section>
      </article>
    );
  }, [
    expandedEmailIds,
    formatSentAt,
    getChannelLabel,
    getTypeLabel,
    getSenderLabel,
    isSenderDataLoading,
    senderLookupTargets,
    toggleEmailBody,
  ]);

  const historyItems = useMemo(() => historyRecords.map(renderHistoryItem), [historyRecords, renderHistoryItem]);

  const skeletonItems = useMemo(() => (
    <div className="space-y-4" aria-label="Đang tải lịch sử email">
      {[0, 1, 2, 3].map((item) => (
        <Skeleton key={item} className="h-28 w-full rounded-xl" />
      ))}
    </div>
  ), []);

  const errorMessage = useMemo(() => {
    if (!error) {
      return 'Đã xảy ra lỗi khi tải lịch sử email. Vui lòng thử lại.';
    }
    if (error instanceof Error) {
      return error.message || 'Đã xảy ra lỗi khi tải lịch sử email.';
    }
    if (typeof error === 'object' && error !== null && 'message' in error) {
      const message = (error as { message?: unknown }).message;
      if (typeof message === 'string' && message.trim().length > 0) {
        return message;
      }
    }
    if (typeof error === 'string') {
      return error;
    }
    return 'Đã xảy ra lỗi khi tải lịch sử email. Vui lòng thử lại.';
  }, [error]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        className={cn("flex h-full w-full flex-col bg-muted/50 p-0 sm:max-w-[760px]", isMobile && "rounded-t-2xl max-h-[94dvh] shadow-[0_-4px_24px_rgba(0,0,0,0.08)]")}
        aria-label="Lịch sử email đã gửi"
      >
        {/* Mobile drag handle */}
        {isMobile && (
          <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
            <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
          </div>
        )}
        <div className="flex items-center justify-between border-b bg-card px-4 py-4 sm:px-6">
          <SheetHeader className="text-left">
            <SheetTitle className="typography-headline-small">
              Lịch sử email
            </SheetTitle>
            <SheetDescription>
              Theo dõi chi tiết các email bảng lương đã gửi đi gần đây.
            </SheetDescription>
          </SheetHeader>
          <SheetClose asChild>
            <Button
              variant="ghost"
              size="icon"
              className="min-h-[44px] min-w-[44px]"
              aria-label="Đóng lịch sử email"
            >
              <X className="h-5 w-5" />
            </Button>
          </SheetClose>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-6 sm:px-6">
          {isLoading && skeletonItems}

          {!isLoading && isError && (
            <Alert variant="destructive" className="mb-4">
              <AlertCircle className="h-5 w-5" />
              <div>
                <AlertTitle>Tải dữ liệu thất bại</AlertTitle>
                <AlertDescription>
                  {errorMessage}
                </AlertDescription>
              </div>
              <Button
                type="button"
                onClick={handleRetry}
                variant="outline"
                className="mt-4 min-h-[44px]"
              >
                Thử lại
              </Button>
            </Alert>
          )}

          {!isLoading && !isError && isEmpty && (
            <div className="rounded-xl border border-dashed border-border bg-muted/50/80 px-6">
              <EmptyState
                title="Chưa có email nào"
                description="Hệ thống chưa ghi nhận email thanh toán nào được gửi đi."
                size="sm"
              />
            </div>
          )}

          {!isLoading && !isError && (
            <div className="space-y-5">{historyItems}</div>
          )}
        </div>

        <div className="border-t bg-card px-4 py-4 sm:px-6" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
          {hasNextPage ? (
            <Button
              type="button"
              onClick={handleLoadMore}
              disabled={isFetchingNextPage}
              className="w-full min-h-[44px]"
            >
              {isFetchingNextPage ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Đang tải thêm...
                </>
              ) : (
                'Tải thêm'
              )}
            </Button>
          ) : (
            <p className="text-center text-sm text-muted-foreground">
              Đã hiển thị toàn bộ lịch sử email.
            </p>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
});
