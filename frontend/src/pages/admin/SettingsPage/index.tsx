import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Settings, Percent, Building2, Receipt, Mail, Bell } from 'lucide-react';
import { PageHeader } from '@/components/shared/PageHeader';
import { Skeleton } from '@/components/ui/skeleton';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SettingCard } from '@/components/settings/SettingCard';
import { useSettingsForm } from '@/hooks/settings/useSettingsForm';
import { FeeScheduleSection } from '@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection';
import { DisbursementFeeScheduleSection } from '@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection';
import { AdminEmailComposer } from '@/components/email/AdminEmailComposer';
import { SendNotificationDialog } from '@/components/settings/SendNotificationDialog';
import { Button } from '@/components/ui/button';

const TAB_GENERAL = 'general';
const TAB_FEE_CONFIG = 'fee-config';
const TAB_EMAIL = 'email';
const TAB_NOTIFICATIONS = 'notifications';
const VALID_TABS = new Set([TAB_GENERAL, TAB_FEE_CONFIG, TAB_EMAIL, TAB_NOTIFICATIONS]);

const SettingsPage = () => {
  const form = useSettingsForm();
  const [searchParams, setSearchParams] = useSearchParams();
  const [isNotificationDialogOpen, setIsNotificationDialogOpen] = useState(false);
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
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-6">
        {headerSkeleton}
        <Skeleton className="h-9 w-72" />
        <div className="space-y-3">
          <Skeleton className="h-4 w-32" />
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Skeleton className="h-28" />
            <Skeleton className="h-28" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
      <PageHeader
        icon={Settings}
        title="Cài đặt hệ thống"
        description="Quản lý các cài đặt nghiệp vụ của hệ thống"
      />

      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-5">
        <TabsList>
          <TabsTrigger value={TAB_GENERAL} className="gap-1.5">
            <Settings className="h-3.5 w-3.5" />
            Trả lương
          </TabsTrigger>
          <TabsTrigger value={TAB_FEE_CONFIG} className="gap-1.5">
            <Receipt className="h-3.5 w-3.5" />
            Tạm ứng
          </TabsTrigger>
          <TabsTrigger value={TAB_EMAIL} className="gap-1.5">
            <Mail className="h-3.5 w-3.5" />
            Email
          </TabsTrigger>
          <TabsTrigger value={TAB_NOTIFICATIONS} className="gap-1.5">
            <Bell className="h-3.5 w-3.5" />
            Thông báo
          </TabsTrigger>
        </TabsList>

        <TabsContent value={TAB_GENERAL} className="space-y-8 mt-0">
          <section className="space-y-4">
            <div className="flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-blue-500/10">
                <Percent className="h-3.5 w-3.5 text-blue-600" />
              </div>
              <div>
                <h2 className="text-sm font-semibold">Giới hạn thanh toán</h2>
                <p className="text-xs text-muted-foreground">
                  Tỷ lệ phần trăm ngân sách được phép giải ngân theo từng chu kỳ
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <SettingCard
                title="Trả lương tuần"
                description="Giới hạn % ngân sách theo chu kỳ tuần"
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
                title="Trả lương tháng"
                description="Giới hạn % ngân sách theo chu kỳ tháng"
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
            </div>
          </section>

          <Separator />

          <section className="space-y-4">
            <div className="flex items-center gap-2">
              <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-violet-500/10">
                <Building2 className="h-3.5 w-3.5 text-violet-600" />
              </div>
              <div>
                <h2 className="text-sm font-semibold">Thông tin đối tác</h2>
                <p className="text-xs text-muted-foreground">
                  Cấu hình thông tin khách hàng sử dụng hệ thống
                </p>
              </div>
            </div>
            <div className="max-w-md">
              <SettingCard
                title="Tên khách hàng"
                description="Tên công ty đối tác sử dụng dịch vụ tạm ứng"
                value={form.partnerCompany}
                originalValue={form.originalPartnerCompany}
                onChange={form.setPartnerCompany}
                onSave={form.handleSavePartnerCompany}
                onReset={() => form.setPartnerCompany(form.originalPartnerCompany)}
                isDirty={form.partnerCompany !== form.originalPartnerCompany}
                isSaving={form.isSaving}
              />
            </div>
          </section>
        </TabsContent>

        <TabsContent value={TAB_FEE_CONFIG} className="mt-0 space-y-8">
          <FeeScheduleSection />
          <Separator />
          <DisbursementFeeScheduleSection />
        </TabsContent>

        <TabsContent value={TAB_EMAIL} className="mt-0">
          <AdminEmailComposer embedded />
        </TabsContent>

        <TabsContent value={TAB_NOTIFICATIONS} className="mt-0">
          <section className="max-w-2xl rounded-xl border bg-card p-5 sm:p-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/10">
                    <Bell className="h-4 w-4 text-primary" />
                  </div>
                  <h2 className="text-sm font-semibold">Gửi Thông Báo</h2>
                </div>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  Soạn và gửi thông báo đến người dùng trong hệ thống.
                </p>
              </div>
              <Button className="min-h-11 shrink-0" onClick={() => setIsNotificationDialogOpen(true)}>
                <Bell className="mr-2 h-4 w-4" />
                Gửi thông báo
              </Button>
            </div>
          </section>
        </TabsContent>
      </Tabs>

      <SendNotificationDialog
        open={isNotificationDialogOpen}
        onOpenChange={setIsNotificationDialogOpen}
      />
    </div>
  );
};

export default SettingsPage;
