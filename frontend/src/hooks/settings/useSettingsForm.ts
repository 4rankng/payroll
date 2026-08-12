import { useState, useEffect } from 'react';
import { useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';

const SETTINGS_KEYS = {
  WEEKLY_PAYMENT_PERCENTAGE: 'bulk_transfer_payment_percentage',
  MONTHLY_PAYMENT_PERCENTAGE: 'monthly_payment_percentage',
  PARTNER_COMPANY: 'partner_company',
  BULK_TRANSFER_WORKBOOK_LIMIT_VND: 'bulk_transfer_workbook_limit_vnd',
  SELF_CHECK_IN_ADVANCE_PERCENTAGE: 'self_check_in_advance_percentage',
  SELF_CHECK_IN_ADVANCE_HOLD_HOURS: 'self_check_in_advance_hold_hours',
} as const;

export interface SettingsFormState {
  weeklyPaymentPercentage: string;
  originalWeeklyPayment: string;
  monthlyPaymentPercentage: string;
  originalMonthlyPayment: string;
  partnerCompany: string;
  originalPartnerCompany: string;
  bulkTransferWorkbookLimitVnd: string;
  originalBulkTransferWorkbookLimitVnd: string;
  bulkTransferWorkbookLimitSaveError: string | null;
  bulkTransferWorkbookLimitUnavailableMessage: string | null;
  selfCheckInAdvancePercentage: string;
  originalSelfCheckInAdvancePercentage: string;
  selfCheckInAdvanceHoldHours: string;
  originalSelfCheckInAdvanceHoldHours: string;
  loadError: string | null;
  isSaving: boolean;
  isLoading: boolean;
  setWeeklyPaymentPercentage: (v: string) => void;
  setMonthlyPaymentPercentage: (v: string) => void;
  setPartnerCompany: (v: string) => void;
  setBulkTransferWorkbookLimitVnd: (v: string) => void;
  setSelfCheckInAdvancePercentage: (v: string) => void;
  setSelfCheckInAdvanceHoldHours: (v: string) => void;
  handleSaveWeeklyPayment: () => void;
  handleSaveMonthlyPayment: () => void;
  handleSavePartnerCompany: () => void;
  handleSaveBulkTransferWorkbookLimitVnd: () => Promise<void>;
  handleSaveSelfCheckInAdvancePercentage: () => void;
  handleSaveSelfCheckInAdvanceHoldHours: () => void;
  retryLoading: () => void;
}

export function useSettingsForm(): SettingsFormState {
  const {
    data: settings,
    isError: isSettingsError,
    isLoading,
    refetch,
  } = useMultipleSettings([
    SETTINGS_KEYS.WEEKLY_PAYMENT_PERCENTAGE,
    SETTINGS_KEYS.MONTHLY_PAYMENT_PERCENTAGE,
    SETTINGS_KEYS.PARTNER_COMPANY,
    SETTINGS_KEYS.BULK_TRANSFER_WORKBOOK_LIMIT_VND,
    SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_PERCENTAGE,
    SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_HOLD_HOURS,
  ]);

  const updateMutation = useUpdateSetting();

  const [weeklyPaymentPercentage, setWeeklyPaymentPercentage] = useState('');
  const [originalWeeklyPayment, setOriginalWeeklyPayment] = useState('');
  const [monthlyPaymentPercentage, setMonthlyPaymentPercentage] = useState('');
  const [originalMonthlyPayment, setOriginalMonthlyPayment] = useState('');
  const [partnerCompany, setPartnerCompany] = useState('');
  const [originalPartnerCompany, setOriginalPartnerCompany] = useState('');
  const [bulkTransferWorkbookLimitVnd, setBulkTransferWorkbookLimitVndState] = useState('');
  const [originalBulkTransferWorkbookLimitVnd, setOriginalBulkTransferWorkbookLimitVnd] = useState('');
  const [selfCheckInAdvancePercentage, setSelfCheckInAdvancePercentage] = useState('');
  const [originalSelfCheckInAdvancePercentage, setOriginalSelfCheckInAdvancePercentage] = useState('');
  const [selfCheckInAdvanceHoldHours, setSelfCheckInAdvanceHoldHours] = useState('');
  const [originalSelfCheckInAdvanceHoldHours, setOriginalSelfCheckInAdvanceHoldHours] = useState('');
  const [bulkTransferWorkbookLimitSaveError, setBulkTransferWorkbookLimitSaveError] =
    useState<string | null>(null);

  useEffect(() => {
    if (settings && Array.isArray(settings)) {
      const weeklySetting = settings.find((s) => s?.key === SETTINGS_KEYS.WEEKLY_PAYMENT_PERCENTAGE);
      const monthlySetting = settings.find((s) => s?.key === SETTINGS_KEYS.MONTHLY_PAYMENT_PERCENTAGE);
      const partnerCompanySetting = settings.find((s) => s?.key === SETTINGS_KEYS.PARTNER_COMPANY);
      const bulkTransferWorkbookLimitSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.BULK_TRANSFER_WORKBOOK_LIMIT_VND,
      );
      const selfCheckInAdvancePercentageSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_PERCENTAGE,
      );
      const selfCheckInAdvanceHoldHoursSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_HOLD_HOURS,
      );

      if (weeklySetting?.value) {
        const value = (parseFloat(weeklySetting.value) * 100).toString();
        setWeeklyPaymentPercentage(value);
        setOriginalWeeklyPayment(value);
      }
      if (monthlySetting?.value) {
        const value = (parseFloat(monthlySetting.value) * 100).toString();
        setMonthlyPaymentPercentage(value);
        setOriginalMonthlyPayment(value);
      }
      if (partnerCompanySetting?.value) {
        setPartnerCompany(partnerCompanySetting.value);
        setOriginalPartnerCompany(partnerCompanySetting.value);
      }
      if (bulkTransferWorkbookLimitSetting?.value != null) {
        setBulkTransferWorkbookLimitVndState(bulkTransferWorkbookLimitSetting.value);
        setOriginalBulkTransferWorkbookLimitVnd(bulkTransferWorkbookLimitSetting.value);
      }
      if (selfCheckInAdvancePercentageSetting?.value != null) {
        setSelfCheckInAdvancePercentage(selfCheckInAdvancePercentageSetting.value);
        setOriginalSelfCheckInAdvancePercentage(selfCheckInAdvancePercentageSetting.value);
      }
      if (selfCheckInAdvanceHoldHoursSetting?.value != null) {
        setSelfCheckInAdvanceHoldHours(selfCheckInAdvanceHoldHoursSetting.value);
        setOriginalSelfCheckInAdvanceHoldHours(selfCheckInAdvanceHoldHoursSetting.value);
      }
    }
  }, [settings]);

  const bulkTransferWorkbookLimitSetting = settings?.find(
    (setting) => setting?.key === SETTINGS_KEYS.BULK_TRANSFER_WORKBOOK_LIMIT_VND,
  );
  const loadError = isSettingsError
    ? 'Không thể tải cài đặt. Vui lòng thử lại.'
    : null;
  const bulkTransferWorkbookLimitUnavailableMessage =
    loadError ??
    (!isLoading && !bulkTransferWorkbookLimitSetting
      ? 'Không tìm thấy cài đặt giới hạn file Chuyển lô. Vui lòng thử tải lại.'
      : null);

  const setBulkTransferWorkbookLimitVnd = (value: string) => {
    setBulkTransferWorkbookLimitSaveError(null);
    setBulkTransferWorkbookLimitVndState(value);
  };

  const handleSaveWeeklyPayment = () => {
    if (!settings || !Array.isArray(settings)) return;
    const setting = settings.find((s) => s?.key === SETTINGS_KEYS.WEEKLY_PAYMENT_PERCENTAGE);
    if (!setting) return;
    updateMutation.mutate(
      { id: setting.id, data: { value: (parseFloat(weeklyPaymentPercentage) / 100).toString() } },
      { onSuccess: () => setOriginalWeeklyPayment(weeklyPaymentPercentage) },
    );
  };

  const handleSaveMonthlyPayment = () => {
    if (!settings || !Array.isArray(settings)) return;
    const setting = settings.find((s) => s?.key === SETTINGS_KEYS.MONTHLY_PAYMENT_PERCENTAGE);
    if (!setting) return;
    updateMutation.mutate(
      { id: setting.id, data: { value: (parseFloat(monthlyPaymentPercentage) / 100).toString() } },
      { onSuccess: () => setOriginalMonthlyPayment(monthlyPaymentPercentage) },
    );
  };

  const handleSavePartnerCompany = () => {
    if (!settings || !Array.isArray(settings)) return;
    const setting = settings.find((s) => s?.key === SETTINGS_KEYS.PARTNER_COMPANY);
    if (!setting) return;
    updateMutation.mutate(
      { id: setting.id, data: { value: partnerCompany } },
      { onSuccess: () => setOriginalPartnerCompany(partnerCompany) },
    );
  };

  const handleSaveSelfCheckInAdvancePercentage = () => {
    if (!settings || !Array.isArray(settings)) return;
    const setting = settings.find(
      (item) => item?.key === SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_PERCENTAGE,
    );
    if (!setting) return;
    updateMutation.mutate(
      { id: setting.id, data: { value: selfCheckInAdvancePercentage } },
      { onSuccess: () => setOriginalSelfCheckInAdvancePercentage(selfCheckInAdvancePercentage) },
    );
  };

  const handleSaveSelfCheckInAdvanceHoldHours = () => {
    if (!settings || !Array.isArray(settings)) return;
    const setting = settings.find(
      (item) => item?.key === SETTINGS_KEYS.SELF_CHECK_IN_ADVANCE_HOLD_HOURS,
    );
    if (!setting) return;
    updateMutation.mutate(
      { id: setting.id, data: { value: selfCheckInAdvanceHoldHours } },
      { onSuccess: () => setOriginalSelfCheckInAdvanceHoldHours(selfCheckInAdvanceHoldHours) },
    );
  };

  const handleSaveBulkTransferWorkbookLimitVnd = async () => {
    if (!bulkTransferWorkbookLimitSetting) return;

    setBulkTransferWorkbookLimitSaveError(null);
    try {
      await updateMutation.mutateAsync({
        id: bulkTransferWorkbookLimitSetting.id,
        data: { value: bulkTransferWorkbookLimitVnd },
      });
      setOriginalBulkTransferWorkbookLimitVnd(bulkTransferWorkbookLimitVnd);
    } catch {
      setBulkTransferWorkbookLimitSaveError(
        'Không thể lưu giới hạn file Chuyển lô. Kiểm tra giá trị và thử lại.',
      );
    }
  };

  const retryLoading = () => {
    void refetch();
  };

  return {
    weeklyPaymentPercentage,
    originalWeeklyPayment,
    monthlyPaymentPercentage,
    originalMonthlyPayment,
    partnerCompany,
    originalPartnerCompany,
    bulkTransferWorkbookLimitVnd,
    originalBulkTransferWorkbookLimitVnd,
    bulkTransferWorkbookLimitSaveError,
    bulkTransferWorkbookLimitUnavailableMessage,
    selfCheckInAdvancePercentage,
    originalSelfCheckInAdvancePercentage,
    selfCheckInAdvanceHoldHours,
    originalSelfCheckInAdvanceHoldHours,
    loadError,
    isSaving: updateMutation.isPending,
    isLoading,
    setWeeklyPaymentPercentage,
    setMonthlyPaymentPercentage,
    setPartnerCompany,
    setBulkTransferWorkbookLimitVnd,
    setSelfCheckInAdvancePercentage,
    setSelfCheckInAdvanceHoldHours,
    handleSaveWeeklyPayment,
    handleSaveMonthlyPayment,
    handleSavePartnerCompany,
    handleSaveBulkTransferWorkbookLimitVnd,
    handleSaveSelfCheckInAdvancePercentage,
    handleSaveSelfCheckInAdvanceHoldHours,
    retryLoading,
  };
}
