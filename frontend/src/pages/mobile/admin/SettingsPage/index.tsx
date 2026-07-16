import { useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Settings } from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { SettingCard } from '@/components/settings/SettingCard';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { AdminEmailComposer } from '@/components/email/AdminEmailComposer';

const TAB_GENERAL = 'general';
const TAB_FEE_CONFIG = 'fee-config';
const TAB_EMAIL = 'email';
const VALID_TABS = new Set([TAB_GENERAL, TAB_FEE_CONFIG, TAB_EMAIL]);

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

  return (
    <div className="max-w-full overflow-hidden pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        title="Cài đặt"
        icon={Settings}
        subtitle="Quản lý cấu hình hệ thống"
      />

      <div className="p-4 space-y-3">
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-3">
          <TabsList className="grid h-auto w-full grid-cols-3 p-1">
            <TabsTrigger value={TAB_GENERAL} className="typography-body-medium min-h-11 px-3 py-2">
              Trả lương
            </TabsTrigger>
            <TabsTrigger value={TAB_FEE_CONFIG} className="typography-body-medium min-h-11 px-3 py-2">
              Tạm ứng
            </TabsTrigger>
            <TabsTrigger value={TAB_EMAIL} className="typography-body-medium min-h-11 px-3 py-2">
              Email
            </TabsTrigger>
          </TabsList>

          <TabsContent value={TAB_GENERAL} className="mt-0 space-y-3">
            <div className="grid grid-cols-1 gap-3">
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
        </Tabs>
      </div>
    </div>
  );
};

export default SettingsPageMobile;
