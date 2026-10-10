import { useEffect, useMemo, useRef, useState } from "react";
import { BannerDualActionBrandFullWidth } from "@/components/marketing/banners/banner-dual-action-brand-full-width";
import { useAssets } from "@/hooks/api/useAssets";
import { useMarkAsReadSilent, useUnreadNotifications } from "@/hooks/api/useNotifications";
import type { Notification } from "@/types/api/notification.types";
import { downloadAssetFile } from "@/utils/file-download";

/**
 * Day-9 check-in roster reminder on Tổng quan (desktop + mobile dashboards).
 *
 * The backend job `prepare_check_in_roster` stores the combined roster workbook
 * as a `checkin_roster` asset, then pushes a `checkin_roster` notification to
 * every admin. Unread notification = pending work:
 *
 * - "Tải file" downloads the newest prepared workbook.
 * - "Đã xong" (and the X) marks EVERY checkin_roster notice read — a stale
 *   notice from a previous month must not resurrect the banner.
 *
 * Renders nothing when there is no unread roster notice.
 */
export function CheckInRosterReminderBanner() {
  const { data: unreadData } = useUnreadNotifications();

  const rosterNotices = useMemo(() => {
    const notices = (unreadData?.notifications ?? []).filter(
      (n) => n.type === "checkin_roster"
    );
    // Show the newest month's copy; dismissed older ones are cleared together.
    return notices.sort((a, b) => b.created_at.localeCompare(a.created_at));
  }, [unreadData?.notifications]);

  if (rosterNotices.length === 0) return null;
  // Child component so the assets query only mounts while a notice exists
  // (useAssets has no `enabled` flag).
  return <RosterReminderActions notices={rosterNotices} />;
}

function RosterReminderActions({ notices }: { notices: Notification[] }) {
  const latest = notices[0];
  // Backend defaults sort to created_at DESC, so limit 1 = newest workbook.
  const { data: assetsData, isLoading: assetsLoading, refetch } = useAssets({
    upload_type: "checkin_roster",
    limit: 1,
  });
  const asset = assetsData?.data?.[0];
  const markAsRead = useMarkAsReadSilent();
  const [downloading, setDownloading] = useState(false);

  // Bounded auto-recovery: the workbook is written ~200ms before the
  // notification, but a refetch racing that window (or a persisted empty
  // result) can leave the primary action dead-ended on a missing asset while
  // the copy promises the file exists. Retry a few times before giving up.
  const retryCountRef = useRef(0);
  useEffect(() => {
    if (asset || assetsLoading || retryCountRef.current >= 3) return;
    const timer = setTimeout(() => {
      retryCountRef.current += 1;
      void refetch();
    }, 1500);
    return () => clearTimeout(timer);
  }, [asset, assetsLoading, refetch]);

  const handleDownload = async () => {
    if (!asset) return;
    setDownloading(true);
    try {
      await downloadAssetFile(asset.id, asset.filename);
    } finally {
      setDownloading(false);
    }
  };

  const handleDone = () => {
    for (const notice of notices) markAsRead.mutate(notice.id);
  };

  return (
    <BannerDualActionBrandFullWidth
      title={latest.title}
      description={latest.message}
      primaryAction={{
        label: asset ? "Tải file" : "Đang tải file…",
        onClick: handleDownload,
        loading: downloading,
        disabled: !asset || assetsLoading,
      }}
      secondaryAction={{
        label: "Đã xong",
        onClick: handleDone,
        loading: markAsRead.isPending,
      }}
      onDismiss={handleDone}
      dismissLabel="Đánh dấu đã xong"
    />
  );
}
