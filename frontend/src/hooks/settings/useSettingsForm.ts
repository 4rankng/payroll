import { useState, useEffect } from 'react';
import { useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';

const SETTINGS_KEYS = {
  WEEKLY_PAYMENT_PERCENTAGE: 'bulk_transfer_payment_percentage',
  MONTHLY_PAYMENT_PERCENTAGE: 'monthly_payment_percentage',
  PARTNER_COMPANY: 'partner_company',
} as const;

export interface SettingsFormState {
  weeklyPaymentPercentage: string;
  originalWeeklyPayment: string;
  monthlyPaymentPercentage: string;
  originalMonthlyPayment: string;
  partnerCompany: string;
  originalPartnerCompany: string;
  isSaving: boolean;
  isLoading: boolean;
  setWeeklyPaymentPercentage: (v: string) => void;
  setMonthlyPaymentPercentage: (v: string) => void;
  setPartnerCompany: (v: string) => void;
  handleSaveWeeklyPayment: () => void;
  handleSaveMonthlyPayment: () => void;
  handleSavePartnerCompany: () => void;
}

export function useSettingsForm(): SettingsFormState {
  const { data: settings, isLoading } = useMultipleSettings([
    SETTINGS_KEYS.WEEKLY_PAYMENT_PERCENTAGE,
    SETTINGS_KEYS.MONTHLY_PAYMENT_PERCENTAGE,
    SETTINGS_KEYS.PARTNER_COMPANY,
  ]);

  const updateMutation = useUpdateSetting();

  const [weeklyPaymentPercentage, setWeeklyPaymentPercentage] = useState('');
  const [originalWeeklyPayment, setOriginalWeeklyPayment] = useState('');
  const [monthlyPaymentPercentage, setMonthlyPaymentPercentage] = useState('');
  const [originalMonthlyPayment, setOriginalMonthlyPayment] = useState('');
  const [partnerCompany, setPartnerCompany] = useState('');
  const [originalPartnerCompany, setOriginalPartnerCompany] = useState('');

  useEffect(() => {
    if (settings && Array.isArray(settings)) {
      const weeklySetting = settings.find((s) => s?.key === SETTINGS_KEYS.WEEKLY_PAYMENT_PERCENTAGE);
      const monthlySetting = settings.find((s) => s?.key === SETTINGS_KEYS.MONTHLY_PAYMENT_PERCENTAGE);
      const partnerCompanySetting = settings.find((s) => s?.key === SETTINGS_KEYS.PARTNER_COMPANY);

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
    }
  }, [settings]);

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

  return {
    weeklyPaymentPercentage,
    originalWeeklyPayment,
    monthlyPaymentPercentage,
    originalMonthlyPayment,
    partnerCompany,
    originalPartnerCompany,
    isSaving: updateMutation.isPending,
    isLoading,
    setWeeklyPaymentPercentage,
    setMonthlyPaymentPercentage,
    setPartnerCompany,
    handleSaveWeeklyPayment,
    handleSaveMonthlyPayment,
    handleSavePartnerCompany,
  };
}
