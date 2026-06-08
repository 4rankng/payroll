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

const TAB_GENERAL = 'general';
const TAB_FEE_CONFIG = 'fee-config';
const VALID_TABS = new Set([TAB_GENERAL, TAB_FEE_CONFIG]);

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
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-9 w-64" />
        <div className="grid grid-cols-1 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="pb-20 max-w-full overflow-hidden">
      <MobilePageHeader
        title="Cài đặt"
        icon={Settings}
        subtitle="Quản lý cấu hình hệ thống"
      />

      <div className="p-4 space-y-3">
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-3">
          <TabsList className="grid w-full grid-cols-2 h-auto p-1">
            <TabsTrigger value={TAB_GENERAL} className="px-3 py-2 typography-body-medium">
              Trả lương
            </TabsTrigger>
            <TabsTrigger value={TAB_FEE_CONFIG} className="px-3 py-2 typography-body-medium">
              Tạm ứng
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
        </Tabs>
      </div>
    </div>
  );
};

export default SettingsPageMobile;
