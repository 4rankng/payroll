import { X } from 'lucide-react';
import { useId } from 'react';

import { SettingsSection } from '@/components/settings/SettingsSection';

import { SettingCard } from '@/components/settings/SettingCard';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { cn } from '@/lib/utils';
import type { SettingsFormState } from '@/hooks/settings/useSettingsForm';

interface SettingToggleCardProps {
  title: string;
  description: string;
  checked: boolean;
  originalChecked: boolean;
  onCheckedChange: (checked: boolean) => void;
  onSave: () => void;
  onReset: () => void;
  isSaving: boolean;
}

const SettingToggleCard = ({
  title,
  description,
  checked,
  originalChecked,
  onCheckedChange,
  onSave,
  onReset,
  isSaving,
}: SettingToggleCardProps) => {
  const generatedId = useId().replace(/:/g, '');
  const switchId = `setting-toggle-${generatedId}`;
  const descriptionId = `${switchId}-description`;
  const isDirty = checked !== originalChecked;

  return (
    <div
      data-slot="setting-row"
      className={cn(
        'group min-w-0 px-3 py-2 transition-colors',
        isDirty && 'bg-warning/5',
      )}
    >
      <div className="grid min-w-0 items-center gap-x-4 gap-y-2 md:grid-cols-[minmax(0,1fr)_auto]">
        <div className="min-w-0">
          <Label
            htmlFor={switchId}
            className="block text-[13px] font-medium leading-5 text-foreground"
          >
            {title}
          </Label>
          <p
            id={descriptionId}
            className="text-xs leading-4 text-muted-foreground"
          >
            {description}
          </p>
        </div>
        <div className="flex min-w-0 items-center gap-1.5 md:justify-end">
          <Switch
            id={switchId}
            checked={checked}
            onCheckedChange={onCheckedChange}
            aria-describedby={descriptionId}
            disabled={isSaving}
          />
          {isDirty && (
            <>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                aria-label="Hoàn tác"
                title="Hoàn tác"
                onClick={onReset}
                disabled={isSaving}
                className="text-muted-foreground hover:text-foreground"
              >
                <X aria-hidden="true" className="h-4 w-4" />
              </Button>
              <Button type="button" size="sm" onClick={onSave} disabled={isSaving}>
                {isSaving ? 'Đang lưu…' : 'Lưu'}
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

interface SettingsGeneralPanelProps {
  form: SettingsFormState;
}

export const SettingsGeneralPanel = ({ form }: SettingsGeneralPanelProps) => (
  <div className="columns-1 gap-3 xl:columns-2 [&>*]:mb-3 [&>*]:break-inside-avoid">
    <SettingsSection kicker="Chu kỳ tuần" title="Lương tuần">
      <SettingCard
        title="Tỷ lệ trả lương tuần"
        description="Phần trăm ngân sách được phép trả theo chu kỳ tuần."
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
    </SettingsSection>

    <SettingsSection kicker="Chu kỳ ngày" title="Lương ngày">
      <SettingCard
        title="Tỷ lệ ứng lương tự chấm công"
        description="Phần trăm tiền công đã ghi nhận được cộng vào hạn mức ứng lương."
        value={form.selfCheckInAdvancePercentage}
        originalValue={form.originalSelfCheckInAdvancePercentage}
        onChange={form.setSelfCheckInAdvancePercentage}
        onSave={form.handleSaveSelfCheckInAdvancePercentage}
        onReset={() =>
          form.setSelfCheckInAdvancePercentage(form.originalSelfCheckInAdvancePercentage)
        }
        isDirty={
          form.selfCheckInAdvancePercentage !== form.originalSelfCheckInAdvancePercentage
        }
        isSaving={form.isSaving}
        type="number"
        min={1}
        max={100}
        step={1}
        suffix="%"
      />
      <SettingCard
        title="Thời gian chờ ứng lương tự chấm công sau khi tan ca"
        description="Số giờ chờ trước khi tiền công của ca hoàn tất được cộng vào hạn mức ứng lương."
        value={form.selfCheckInAdvanceHoldHours}
        originalValue={form.originalSelfCheckInAdvanceHoldHours}
        onChange={form.setSelfCheckInAdvanceHoldHours}
        onSave={form.handleSaveSelfCheckInAdvanceHoldHours}
        onReset={() =>
          form.setSelfCheckInAdvanceHoldHours(form.originalSelfCheckInAdvanceHoldHours)
        }
        isDirty={
          form.selfCheckInAdvanceHoldHours !== form.originalSelfCheckInAdvanceHoldHours
        }
        isSaving={form.isSaving}
        type="number"
        min={0}
        max={720}
        step={1}
        wholeNumber
        suffix="giờ"
      />
      <SettingCard
        title="Ngày mở kỳ ứng lương tự chấm công"
        description="Ngày trong tháng bắt đầu cho phép nhân viên tự chấm công xin ứng lương cho kỳ hiện tại, áp dụng đến cuối tháng."
        value={form.selfCheckInAdvanceWindowOpenDay}
        originalValue={form.originalSelfCheckInAdvanceWindowOpenDay}
        onChange={form.setSelfCheckInAdvanceWindowOpenDay}
        onSave={form.handleSaveSelfCheckInAdvanceWindowOpenDay}
        onReset={() =>
          form.setSelfCheckInAdvanceWindowOpenDay(form.originalSelfCheckInAdvanceWindowOpenDay)
        }
        isDirty={
          form.selfCheckInAdvanceWindowOpenDay !== form.originalSelfCheckInAdvanceWindowOpenDay
        }
        isSaving={form.isSaving}
        type="number"
        min={1}
        max={28}
        step={1}
        wholeNumber
        suffix="ngày"
      />
      <SettingCard
        title="Ngày khóa yêu cầu ứng lương"
        description="Từ ngày này trong tháng, nhân viên không thể tạo yêu cầu ứng lương mới. Mặc định 9, tối đa 15."
        value={form.advanceRequestStopDay}
        originalValue={form.originalAdvanceRequestStopDay}
        onChange={form.setAdvanceRequestStopDay}
        onSave={form.handleSaveAdvanceRequestStopDay}
        onReset={() => form.setAdvanceRequestStopDay(form.originalAdvanceRequestStopDay)}
        isDirty={form.advanceRequestStopDay !== form.originalAdvanceRequestStopDay}
        isSaving={form.isSaving}
        type="number"
        min={2}
        max={15}
        step={1}
        wholeNumber
        suffix="ngày"
      />
    </SettingsSection>

    <SettingsSection kicker="Chu kỳ tháng" title="Lương tháng">
      <SettingCard
        title="Tỷ lệ trả lương tháng"
        description="Phần trăm ngân sách được phép trả theo chu kỳ tháng."
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
        title="Giới hạn tổng tiền mỗi file Chuyển lô"
        description="Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này."
        value={form.bulkTransferWorkbookLimitVnd}
        originalValue={form.originalBulkTransferWorkbookLimitVnd}
        onChange={form.setBulkTransferWorkbookLimitVnd}
        onSave={form.handleSaveBulkTransferWorkbookLimitVnd}
        onReset={() =>
          form.setBulkTransferWorkbookLimitVnd(form.originalBulkTransferWorkbookLimitVnd)
        }
        isDirty={
          form.bulkTransferWorkbookLimitVnd !== form.originalBulkTransferWorkbookLimitVnd
        }
        isSaving={form.isSaving}
        displayMode="currency-slider"
        min="100000000"
        max="500000000"
        sliderPresets={[
          '100000000',
          '200000000',
          '300000000',
          '400000000',
          '500000000',
        ]}
        errorMessage={form.bulkTransferWorkbookLimitSaveError}
        unavailableMessage={form.bulkTransferWorkbookLimitUnavailableMessage}
        onRetry={form.retryLoading}
      />
    </SettingsSection>

    <SettingsSection kicker="Cảnh báo" title="Số dư ví tiền">
      <SettingCard
        title="Ngưỡng cảnh báo số dư ví tiền"
        description="Khi số dư ví tiền xuống dưới ngưỡng này, hệ thống gửi thông báo đẩy đến toàn bộ quản trị viên. Mỗi lần giảm chạm ngưỡng chỉ nhận một thông báo."
        value={form.walletBalanceAlertThresholdVnd}
        originalValue={form.originalWalletBalanceAlertThresholdVnd}
        onChange={form.setWalletBalanceAlertThresholdVnd}
        onSave={form.handleSaveWalletBalanceAlertThresholdVnd}
        onReset={() =>
          form.setWalletBalanceAlertThresholdVnd(form.originalWalletBalanceAlertThresholdVnd)
        }
        isDirty={form.walletBalanceAlertThresholdVnd !== form.originalWalletBalanceAlertThresholdVnd}
        isSaving={form.isSaving}
        displayMode="currency-slider"
        min="1000000"
        max="1000000000"
        sliderPresets={[
          '10000000',
          '50000000',
          '100000000',
          '200000000',
          '500000000',
        ]}
        errorMessage={form.walletBalanceAlertThresholdSaveError}
        unavailableMessage={form.walletBalanceAlertThresholdUnavailableMessage}
        onRetry={form.retryLoading}
      />
    </SettingsSection>

    <SettingsSection kicker="Đối tác" title="Thông tin đối tác">
      <SettingCard
        title="Tên khách hàng"
        description="Tên công ty đối tác sử dụng dịch vụ tạm ứng."
        value={form.partnerCompany}
        originalValue={form.originalPartnerCompany}
        onChange={form.setPartnerCompany}
        onSave={form.handleSavePartnerCompany}
        onReset={() => form.setPartnerCompany(form.originalPartnerCompany)}
        isDirty={form.partnerCompany !== form.originalPartnerCompany}
        isSaving={form.isSaving}
      />
    </SettingsSection>

    <SettingsSection kicker="Ngân hàng" title="Tài khoản nhận lương tuần">
      <SettingToggleCard
        title="Hiện tài khoản nhận chuyển khoản lương tuần"
        description="Khi tắt, sao kê lương tuần (email và file Excel) chỉ là thông báo đối chiếu, không kèm thông tin chuyển khoản."
        checked={form.transferBankVisible}
        originalChecked={form.originalTransferBankVisible}
        onCheckedChange={form.setTransferBankVisible}
        onSave={form.handleSaveTransferBankVisible}
        onReset={() => form.setTransferBankVisible(form.originalTransferBankVisible)}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Chủ tài khoản lương tuần"
        description="Tên chủ tài khoản thụ hưởng in trên sao kê lương tuần."
        value={form.transferBankHolder}
        originalValue={form.originalTransferBankHolder}
        onChange={form.setTransferBankHolder}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankHolder(form.originalTransferBankHolder)}
        isDirty={form.transferBankHolder !== form.originalTransferBankHolder}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Số tài khoản lương tuần"
        description="Số tài khoản nhận chuyển khoản in trên sao kê lương tuần."
        value={form.transferBankNumber}
        originalValue={form.originalTransferBankNumber}
        onChange={form.setTransferBankNumber}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankNumber(form.originalTransferBankNumber)}
        isDirty={form.transferBankNumber !== form.originalTransferBankNumber}
        isSaving={form.isSaving}
        displayMode="account-number"
      />
      <SettingCard
        title="Ngân hàng lương tuần"
        description="Tên ngân hàng nhận chuyển khoản in trên sao kê lương tuần."
        value={form.transferBankName}
        originalValue={form.originalTransferBankName}
        onChange={form.setTransferBankName}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankName(form.originalTransferBankName)}
        isDirty={form.transferBankName !== form.originalTransferBankName}
        isSaving={form.isSaving}
      />
    </SettingsSection>

    <SettingsSection kicker="Ngân hàng" title="Tài khoản FlexPay (OnePay)">
      <SettingToggleCard
        title="Hiện tài khoản nhận chuyển khoản FlexPay"
        description="Khi tắt, sao kê FlexPay (email và file Excel) chỉ là thông báo đối chiếu, không kèm thông tin chuyển khoản."
        checked={form.flexPayTransferBankVisible}
        originalChecked={form.originalFlexPayTransferBankVisible}
        onCheckedChange={form.setFlexPayTransferBankVisible}
        onSave={form.handleSaveFlexPayTransferBankVisible}
        onReset={() => form.setFlexPayTransferBankVisible(form.originalFlexPayTransferBankVisible)}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Chủ tài khoản FlexPay"
        description="Tên chủ tài khoản thụ hưởng in trên sao kê FlexPay."
        value={form.flexPayTransferBankHolder}
        originalValue={form.originalFlexPayTransferBankHolder}
        onChange={form.setFlexPayTransferBankHolder}
        onSave={form.handleSaveFlexPayTransferBank}
        onReset={() =>
          form.setFlexPayTransferBankHolder(form.originalFlexPayTransferBankHolder)
        }
        isDirty={form.flexPayTransferBankHolder !== form.originalFlexPayTransferBankHolder}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Số tài khoản FlexPay"
        description="Số tài khoản nhận chuyển khoản in trên sao kê FlexPay."
        value={form.flexPayTransferBankNumber}
        originalValue={form.originalFlexPayTransferBankNumber}
        onChange={form.setFlexPayTransferBankNumber}
        onSave={form.handleSaveFlexPayTransferBank}
        onReset={() =>
          form.setFlexPayTransferBankNumber(form.originalFlexPayTransferBankNumber)
        }
        isDirty={form.flexPayTransferBankNumber !== form.originalFlexPayTransferBankNumber}
        isSaving={form.isSaving}
        displayMode="account-number"
      />
      <SettingCard
        title="Ngân hàng FlexPay"
        description="Tên ngân hàng nhận chuyển khoản in trên sao kê FlexPay."
        value={form.flexPayTransferBankName}
        originalValue={form.originalFlexPayTransferBankName}
        onChange={form.setFlexPayTransferBankName}
        onSave={form.handleSaveFlexPayTransferBank}
        onReset={() => form.setFlexPayTransferBankName(form.originalFlexPayTransferBankName)}
        isDirty={form.flexPayTransferBankName !== form.originalFlexPayTransferBankName}
        isSaving={form.isSaving}
      />
    </SettingsSection>
  </div>
);
