import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, MessageCircle, RefreshCw, Settings } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { SettingCard } from '@/components/settings/SettingCard';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { AdminEmailComposer } from '@/components/email/AdminEmailComposer';
import { SendNotificationComposer } from '@/components/settings/SendNotificationComposer';
import { ZaloConnectionSection } from '@/components/settings/ZaloConnectionSection';

const TAB_GENERAL = 'general';
const TAB_FEE_CONFIG = 'fee-config';
const TAB_EMAIL = 'email';
const TAB_NOTIFICATIONS = 'notifications';
const TAB_ZALO = 'zalo';
const VALID_TABS = new Set([TAB_GENERAL, TAB_FEE_CONFIG, TAB_EMAIL, TAB_NOTIFICATIONS, TAB_ZALO]);

const SettingsPageMobile = () => {
  const form = useSettingsForm();
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedTab = searchParams.get('tab');
  const activeTab = requestedTab && VALID_TABS.has(requestedTab) ? requestedTab : TAB_GENERAL;

  const handleTabChange = useCallback(
    (value: string) => {
      const next = new URLSearchParams(searchParams);
      if (value === TAB_GENERAL) {
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
      <div className="p-4 space-y-4">
        <Skeleton className="h-11 w-48" />
        <Skeleton className="h-11 w-64" />
        <div className="grid grid-cols-1 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      </div>
    );
  }

  if (form.loadError) {
    return (
      <div className="max-w-full overflow-hidden pb-[calc(5rem+env(safe-area-inset-bottom))]">
        <MobilePageHeader
          title="Cài đặt"
          icon={Settings}
          subtitle="Quản lý cấu hình hệ thống"
        />
        <div className="p-4">
          <div
            role="alert"
            className="space-y-4 rounded-xl border border-destructive/30 bg-destructive/5 p-4"
          >
            <div className="flex min-w-0 items-start gap-3">
              <AlertCircle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-destructive" />
              <p className="min-w-0 break-words text-sm text-destructive">{form.loadError}</p>
            </div>
            <Button
              type="button"
              variant="outline"
              onClick={form.retryLoading}
              className="h-11 w-full gap-2"
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
    <div className="max-w-full overflow-hidden pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        title="Cài đặt"
        icon={Settings}
        subtitle="Quản lý cấu hình hệ thống"
      />

      <div className="p-4 space-y-3">
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-3">
          <TabsList className="grid h-auto w-full grid-cols-6 p-1">
            <TabsTrigger value={TAB_GENERAL} className="col-span-2 min-h-11 px-2 py-2 typography-body-medium">
              Trả lương
            </TabsTrigger>
            <TabsTrigger value={TAB_FEE_CONFIG} className="col-span-2 min-h-11 px-2 py-2 typography-body-medium">
              Tạm ứng
            </TabsTrigger>
            <TabsTrigger value={TAB_EMAIL} className="col-span-2 min-h-11 px-2 py-2 typography-body-medium">
              Email
            </TabsTrigger>
            <TabsTrigger value={TAB_NOTIFICATIONS} className="col-span-3 min-h-11 px-2 py-2 typography-body-medium">
              Thông báo
            </TabsTrigger>
            <TabsTrigger value={TAB_ZALO} className="col-span-3 min-h-11 gap-1.5 px-2 py-2 typography-body-medium">
              <MessageCircle aria-hidden="true" className="size-3.5" />
              Zalo ZNS
            </TabsTrigger>
          </TabsList>

          <TabsContent value={TAB_GENERAL} className="mt-0 space-y-3">
            <div className="grid grid-cols-1 gap-3">
              <SettingCard
                title="Giới hạn tổng tiền mỗi file Chuyển lô"
                description="Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này."
                value={form.bulkTransferWorkbookLimitVnd}
                originalValue={form.originalBulkTransferWorkbookLimitVnd}
                onChange={form.setBulkTransferWorkbookLimitVnd}
                onSave={form.handleSaveBulkTransferWorkbookLimitVnd}
                onReset={() =>
                  form.setBulkTransferWorkbookLimitVnd(
                    form.originalBulkTransferWorkbookLimitVnd,
                  )
                }
                isDirty={
                  form.bulkTransferWorkbookLimitVnd !==
                  form.originalBulkTransferWorkbookLimitVnd
                }
                isSaving={form.isSaving}
                displayMode="currency-vnd"
                min="2"
                max="9223372036854775807"
                errorMessage={form.bulkTransferWorkbookLimitSaveError}
                unavailableMessage={form.bulkTransferWorkbookLimitUnavailableMessage}
                onRetry={form.retryLoading}
              />
              <SettingCard
                title="Tỷ lệ trả lương tuần"
                description="Giới hạn phần trăm ngân sách được phép trả theo chu kỳ tuần"
                value={form.weeklyPaymentPercentage}
                originalValue={form.originalWeeklyPayment}
                onChange={form.setWeeklyPaymentPercentage}
                onSave={form.handleSaveWeeklyPayment}
                onReset={() => form.setWeeklyPaymentPercentage(form.originalWeeklyPayment)}
                isDirty={form.weeklyPaymentPercentage !== form.originalWeeklyPayment}
                isSaving={form.isSaving}
                type="number"
                suffix="%"
              />
              <SettingCard
                title="Tỷ lệ trả lương tháng"
                description="Giới hạn phần trăm ngân sách được phép trả theo chu kỳ tháng"
                value={form.monthlyPaymentPercentage}
                originalValue={form.originalMonthlyPayment}
                onChange={form.setMonthlyPaymentPercentage}
                onSave={form.handleSaveMonthlyPayment}
                onReset={() => form.setMonthlyPaymentPercentage(form.originalMonthlyPayment)}
                isDirty={form.monthlyPaymentPercentage !== form.originalMonthlyPayment}
                isSaving={form.isSaving}
                type="number"
                suffix="%"
              />
              <SettingCard
                title="Khách hàng"
                description="Tên khách hàng sử dụng dịch vụ tạm ứng"
                value={form.partnerCompany}
                originalValue={form.originalPartnerCompany}
                onChange={form.setPartnerCompany}
                onSave={form.handleSavePartnerCompany}
                onReset={() => form.setPartnerCompany(form.originalPartnerCompany)}
                isDirty={form.partnerCompany !== form.originalPartnerCompany}
                isSaving={form.isSaving}
              />
            </div>
          </TabsContent>

          <TabsContent value={TAB_FEE_CONFIG} className="mt-0 space-y-3">
            <FeeScheduleSection />
            <Separator />
            <DisbursementFeeScheduleSection />
          </TabsContent>

          <TabsContent value={TAB_EMAIL} className="mt-0">
            <AdminEmailComposer embedded />
          </TabsContent>

          <TabsContent value={TAB_NOTIFICATIONS} className="mt-0">
            <SendNotificationComposer />
          </TabsContent>

          <TabsContent value={TAB_ZALO} className="mt-0">
            <ZaloConnectionSection />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
};

export default SettingsPageMobile;
