import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, RefreshCw05 } from '@untitledui/icons';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { SettingsGeneralPanel } from '@/components/settings/SettingsGeneralPanel';
import { SettingsEmployeePanel } from '@/components/settings/SettingsEmployeePanel';
import {
  SETTINGS_TABS,
  SettingsTabList,
  VALID_SETTINGS_TABS,
} from '@/components/settings/SettingsTabList';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { WeeklyPaymentFeeScheduleSection } from '@/components/admin/WeeklyPaymentFeeSchedule/WeeklyPaymentFeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { ZaloConnectionSection } from '@/components/settings/ZaloConnectionSection';
import { AdBannerSection } from '@/components/settings/AdBannerSection';
import { ApiKeysSection } from '@/components/settings/ApiKeysSection';
import { McpApiKeysSection } from '@/components/settings/McpApiKeysSection';
import { OpsApiKeysSection } from '@/components/settings/OpsApiKeysSection';

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
      <div className="min-h-[100dvh] bg-background">
        <MobilePageHeader kicker="Hệ thống" title="Cài đặt" className="px-3" />
        <div className="space-y-3 px-3 pt-3">
          <Skeleton className="h-11 w-full" />
          <div className="console-panel space-y-3 p-3">
            <Skeleton className="h-40 w-full" />
            <Skeleton className="h-40 w-full" />
          </div>
        </div>
      </div>
    );
  }

  if (form.loadError) {
    return (
      <div className="min-h-[100dvh] bg-background">
        <MobilePageHeader kicker="Hệ thống" title="Cài đặt" className="px-3" />
        <div className="px-3 pt-3">
          <div
            role="alert"
            className="space-y-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3"
          >
            <div className="flex min-w-0 items-start gap-2.5">
              <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
              <p className="min-w-0 break-words text-[13px] leading-5 text-destructive">
                {form.loadError}
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={form.retryLoading}
              className="h-11 w-full gap-2"
            >
              <RefreshCw05 aria-hidden="true" className="h-4 w-4" />
              Thử lại
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[100dvh] max-w-full overflow-x-clip bg-background">
      <MobilePageHeader kicker="Hệ thống" title="Cài đặt" className="px-3" />

      <div className="px-3 pb-[calc(5.75rem+env(safe-area-inset-bottom))] pt-3">
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-3">
          <SettingsTabList />

          <TabsContent value={SETTINGS_TABS.general} className="mt-0">
            <SettingsGeneralPanel form={form} />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.employees} className="mt-0">
            <SettingsEmployeePanel />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.feeConfig} className="mt-0 space-y-3">
            <WeeklyPaymentFeeScheduleSection />
            <FeeScheduleSection />
            <DisbursementFeeScheduleSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.zalo} className="mt-0">
            <ZaloConnectionSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.ads} className="mt-0">
            <AdBannerSection />
          </TabsContent>

          <TabsContent value={SETTINGS_TABS.apiKeys} className="mt-0 space-y-3">
            <ApiKeysSection />
            <McpApiKeysSection />
            <OpsApiKeysSection />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default SettingsPageMobile;
