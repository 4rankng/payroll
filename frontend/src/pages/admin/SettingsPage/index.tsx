import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, RefreshCw, Settings } from 'lucide-react';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
} from '@/components/shared/AdminPageFrame';
import { PageHeader } from '@/components/shared/PageHeader';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { SettingsGeneralPanel } from '@/components/settings/SettingsGeneralPanel';
import {
  SETTINGS_TABS,
  SettingsTabList,
  VALID_SETTINGS_TABS,
} from '@/components/settings/SettingsTabList';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { WeeklyPaymentFeeScheduleSection } from '@/components/admin/WeeklyPaymentFeeSchedule/WeeklyPaymentFeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { AdminEmailComposer } from '@/components/email/AdminEmailComposer';
import { SendNotificationComposer } from '@/components/settings/SendNotificationComposer';
import { ZaloConnectionSection } from '@/components/settings/ZaloConnectionSection';
import { AdBannerSection } from '@/components/settings/AdBannerSection';
import { ApiKeysSection } from '@/components/settings/ApiKeysSection';

const SettingsPage = () => {
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

  const headerSkeleton = useMemo(
    () => (
      <div className="space-y-2">
        <Skeleton className="h-7 w-52" />
        <Skeleton className="h-4 w-80" />
      </div>
    ),
    [],
  );

  if (form.isLoading) {
    return (
      <AdminPageCanvas contentClassName="max-w-[1180px] space-y-6">
        <AdminPageHeaderCard>{headerSkeleton}</AdminPageHeaderCard>
        <Skeleton className="h-9 w-72" />
        <div className="space-y-3">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-72 w-full" />
        </div>
      </AdminPageCanvas>
    );
  }

  if (form.loadError) {
    return (
      <AdminPageCanvas contentClassName="max-w-[1180px] space-y-6">
        <AdminPageHeaderCard>
          <PageHeader
            icon={Settings}
            title="Cài đặt hệ thống"
            description="Quản lý các cài đặt nghiệp vụ của hệ thống"
          />
        </AdminPageHeaderCard>
        <div
          role="alert"
          className="flex flex-col gap-4 rounded-xl border border-destructive/30 bg-destructive/5 p-4 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="flex min-w-0 items-start gap-3">
            <AlertCircle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-destructive" />
            <p className="text-sm text-destructive">{form.loadError}</p>
          </div>
          <Button type="button" variant="outline" onClick={form.retryLoading} className="h-11 gap-2">
            <RefreshCw aria-hidden="true" className="h-4 w-4" />
            Thử lại
          </Button>
        </div>
      </AdminPageCanvas>
    );
  }

  return (
    <AdminPageCanvas contentClassName="max-w-[1180px] space-y-5">
      <AdminPageHeaderCard>
        <PageHeader
          icon={Settings}
          title="Cài đặt hệ thống"
          description="Quản lý các cài đặt nghiệp vụ của hệ thống"
        />
      </AdminPageHeaderCard>

      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-5">
        <SettingsTabList />

        <TabsContent value={SETTINGS_TABS.general} className="mt-0">
          <SettingsGeneralPanel form={form} />
        </TabsContent>

        <TabsContent value={SETTINGS_TABS.feeConfig} className="mt-0 space-y-8">
          <FeeScheduleSection />
          <Separator />
          <WeeklyPaymentFeeScheduleSection />
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
    </AdminPageCanvas>
  );
};

export default SettingsPage;
