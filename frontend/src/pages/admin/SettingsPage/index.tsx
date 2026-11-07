import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, RefreshCw05 } from '@untitledui/icons';
import { PageHeader } from '@/components/shared/PageHeader';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { SettingsGeneralPanel } from '@/components/settings/SettingsGeneralPanel';
import { SettingsEmployeePanel } from '@/components/settings/SettingsEmployeePanel';
import {
  SETTINGS_TABS,
  SettingsTabList,
  VALID_SETTINGS_TABS,
} from '@/components/settings/SettingsTabList';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { WeeklyPaymentFeeScheduleSection } from '@/components/admin/WeeklyPaymentFeeSchedule/WeeklyPaymentFeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { ZaloConnectionSection } from '@/components/settings/ZaloConnectionSection';
import { AdBannerSection } from '@/components/settings/AdBannerSection';
import { ApiKeysSection } from '@/components/settings/ApiKeysSection';
import { McpApiKeysSection } from '@/components/settings/McpApiKeysSection';
import { OpsApiKeysSection } from '@/components/settings/OpsApiKeysSection';

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

  if (form.isLoading) {
    return (
      <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
        <PageHeader kicker="Hệ thống" title="Cài đặt" />
        <Skeleton className="h-9 w-80 max-w-full" />
        <div className="console-panel space-y-3 p-3">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-56 w-full" />
        </div>
      </div>
    );
  }

  if (form.loadError) {
    return (
      <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
        <PageHeader kicker="Hệ thống" title="Cài đặt" />
        <div
          role="alert"
          className="flex flex-col gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3 sm:flex-row sm:items-center sm:justify-between"
        >
          <div className="flex min-w-0 items-start gap-2.5">
            <AlertCircle aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
            <p className="text-[13px] leading-5 text-destructive">{form.loadError}</p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={form.retryLoading}
            className="h-9 shrink-0 gap-2"
          >
            <RefreshCw05 aria-hidden="true" className="h-3.5 w-3.5" />
            Thử lại
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-[1280px] space-y-3 p-3 sm:p-4">
      <PageHeader kicker="Hệ thống" title="Cài đặt" />

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
  );
};

export default SettingsPage;
