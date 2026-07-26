import { useState, useEffect } from 'react';
import { useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';

const SETTINGS_KEYS = {
  WEEKLY_PAYMENT_PERCENTAGE: 'bulk_transfer_payment_percentage',
  MONTHLY_PAYMENT_PERCENTAGE: 'monthly_payment_percentage',
  PARTNER_COMPANY: 'partner_company',
  BULK_TRANSFER_WORKBOOK_LIMIT_VND: 'bulk_transfer_workbook_limit_vnd',
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
  loadError: string | null;
  isSaving: boolean;
  isLoading: boolean;
  setWeeklyPaymentPercentage: (v: string) => void;
  setMonthlyPaymentPercentage: (v: string) => void;
  setPartnerCompany: (v: string) => void;
  setBulkTransferWorkbookLimitVnd: (v: string) => void;
  handleSaveWeeklyPayment: () => void;
  handleSaveMonthlyPayment: () => void;
  handleSavePartnerCompany: () => void;
  handleSaveBulkTransferWorkbookLimitVnd: () => Promise<void>;
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
    loadError,
    isSaving: updateMutation.isPending,
    isLoading,
    setWeeklyPaymentPercentage,
    setMonthlyPaymentPercentage,
    setPartnerCompany,
    setBulkTransferWorkbookLimitVnd,
    handleSaveWeeklyPayment,
    handleSaveMonthlyPayment,
    handleSavePartnerCompany,
    handleSaveBulkTransferWorkbookLimitVnd,
    retryLoading,
  };
}
