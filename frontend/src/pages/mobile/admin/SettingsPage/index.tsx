import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, RefreshCw, Settings } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { SettingsGeneralPanel } from '@/components/settings/SettingsGeneralPanel';
import {
  SETTINGS_TABS,
  SettingsTabList,
  VALID_SETTINGS_TABS,
} from '@/components/settings/SettingsTabList';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { AdminEmailComposer } from '@/components/email/AdminEmailComposer';
import { SendNotificationComposer } from '@/components/settings/SendNotificationComposer';
import { ZaloConnectionSection } from '@/components/settings/ZaloConnectionSection';
import { AdBannerSection } from '@/components/settings/AdBannerSection';
import { ApiKeysSection } from '@/components/settings/ApiKeysSection';

const SettingsPageMobile = () => {
  const form = useSettingsForm();
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get('tab');
  const activeTab =
    requestedTab && VALID_SETTINGS_TABS.has(requestedTab)
      ? requestedTab
      : SETTINGS_TABS.general;

  const handleTabChange = useCallback(
    (value: string) => {
      const next = new URLSearchParams(searchParams);
      if (value === SETTINGS_TABS.general) {
        next.delete('tab');
      } else {
        next.set('tab', value);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  if (form.isLoading) {
    return (
      <div className="min-h-[100dvh] bg-[hsl(var(--surface-page))]">
        <MobilePageHeader
          title="Cài đặt"
          icon={Settings}
          subtitle="Quản lý cấu hình hệ thống"
        />
        <div className="px-4 pt-4 space-y-4">
          <Skeleton className="h-11 w-48" />
          <Skeleton className="h-11 w-64" />
          <div className="grid grid-cols-1 gap-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-40" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  if (form.loadError) {
    return (
      <div className="min-h-[100dvh] bg-[hsl(var(--surface-page))]">
        <MobilePageHeader
          title="Cài đặt"
          icon={Settings}
          subtitle="Quản lý cấu hình hệ thống"
        />
        <div className="px-4 pt-4">
          <div
            role="alert"
            className="space-y-4 rounded-2xl border border-destructive/30 bg-destructive/5 p-5"
          >
            <div className="flex min-w-0 items-start gap-3">
              <AlertCircle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-destructive" />
              <p className="min-w-0 break-words text-sm leading-relaxed text-destructive">{form.loadError}</p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={form.retryLoading}
              className="h-12 w-full gap-2"
            >
              <RefreshCw aria-hidden="true" className="h-4 w-4" />
              Thử lại
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[100dvh] max-w-full overflow-x-clip bg-[hsl(var(--surface-page))]">
      <MobilePageHeader
        title="Cài đặt"
        icon={Settings}
        subtitle="Quản lý cấu hình hệ thống"
      />

      {/* Content area with consistent padding, safe-area, and nav bar clearance */}
      <div
        className="px-4 pb-[calc(5.75rem+env(safe-area-inset-bottom))] pt-4"
      >
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-4">
          <SettingsTabList />

          <TabsContent value={SETTINGS_TABS.general} className="mt-0">
            <SettingsGeneralPanel form={form} />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.feeConfig} className="mt-0 space-y-4">
            <FeeScheduleSection />
            <Separator />
            <DisbursementFeeScheduleSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.email} className="mt-0">
            <AdminEmailComposer embedded />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.notifications} className="mt-0">
            <SendNotificationComposer />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.zalo} className="mt-0">
            <ZaloConnectionSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.ads} className="mt-0">
            <AdBannerSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.apiKeys} className="mt-0">
            <ApiKeysSection />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default SettingsPageMobile;
