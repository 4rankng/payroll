import { useState, useEffect } from 'react';
import { useCreateSetting, useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';

const SETTINGS_KEYS = {
  WEEKLY_PAYMENT_PERCENTAGE: 'bulk_transfer_payment_percentage',
  MONTHLY_PAYMENT_PERCENTAGE: 'monthly_payment_percentage',
  PARTNER_COMPANY: 'partner_company',
  BULK_TRANSFER_WORKBOOK_LIMIT_VND: 'bulk_transfer_workbook_limit_vnd',
  SELF_CHECK_IN_ADVANCE_PERCENTAGE: 'self_check_in_advance_percentage',
  SELF_CHECK_IN_ADVANCE_HOLD_HOURS: 'self_check_in_advance_hold_hours',
  TRANSFER_BANK_HOLDER: 'transfer_bank_account_holder',
  TRANSFER_BANK_NUMBER: 'transfer_bank_account_number',
  TRANSFER_BANK_NAME: 'transfer_bank_name',
  TRANSFER_BANK_VISIBLE: 'transfer_bank_visible',
  FLEXPAY_TRANSFER_BANK_HOLDER: 'flexpay_transfer_bank_account_holder',
  FLEXPAY_TRANSFER_BANK_NUMBER: 'flexpay_transfer_bank_account_number',
  FLEXPAY_TRANSFER_BANK_NAME: 'flexpay_transfer_bank_name',
  FLEXPAY_TRANSFER_BANK_VISIBLE: 'flexpay_transfer_bank_visible',
} as const;

// Defaults shown until the admin saves a value (must mirror backend defaults).
const TRANSFER_BANK_DEFAULTS = {
  HOLDER: 'CONG TY TNHH MTV GPPM TING TING',
  NUMBER: '271866699',
  NAME: 'Ngân hàng Quân đội (MB)',
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
  transferBankHolder: string;
  originalTransferBankHolder: string;
  transferBankNumber: string;
  originalTransferBankNumber: string;
  transferBankName: string;
  originalTransferBankName: string;
  transferBankVisible: boolean;
  originalTransferBankVisible: boolean;
  flexPayTransferBankHolder: string;
  originalFlexPayTransferBankHolder: string;
  flexPayTransferBankNumber: string;
  originalFlexPayTransferBankNumber: string;
  flexPayTransferBankName: string;
  originalFlexPayTransferBankName: string;
  flexPayTransferBankVisible: boolean;
  originalFlexPayTransferBankVisible: boolean;
  loadError: string | null;
  isSaving: boolean;
  isLoading: boolean;
  setWeeklyPaymentPercentage: (v: string) => void;
  setMonthlyPaymentPercentage: (v: string) => void;
  setPartnerCompany: (v: string) => void;
  setBulkTransferWorkbookLimitVnd: (v: string) => void;
  setSelfCheckInAdvancePercentage: (v: string) => void;
  setSelfCheckInAdvanceHoldHours: (v: string) => void;
  setTransferBankHolder: (v: string) => void;
  setTransferBankNumber: (v: string) => void;
  setTransferBankName: (v: string) => void;
  setTransferBankVisible: (v: boolean) => void;
  setFlexPayTransferBankHolder: (v: string) => void;
  setFlexPayTransferBankNumber: (v: string) => void;
  setFlexPayTransferBankName: (v: string) => void;
  setFlexPayTransferBankVisible: (v: boolean) => void;
  handleSaveWeeklyPayment: () => void;
  handleSaveMonthlyPayment: () => void;
  handleSavePartnerCompany: () => void;
  handleSaveBulkTransferWorkbookLimitVnd: () => Promise<void>;
  handleSaveSelfCheckInAdvancePercentage: () => void;
  handleSaveSelfCheckInAdvanceHoldHours: () => void;
  handleSaveTransferBank: () => void;
  handleSaveTransferBankVisible: () => void;
  handleSaveFlexPayTransferBank: () => void;
  handleSaveFlexPayTransferBankVisible: () => void;
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
    SETTINGS_KEYS.TRANSFER_BANK_HOLDER,
    SETTINGS_KEYS.TRANSFER_BANK_NUMBER,
    SETTINGS_KEYS.TRANSFER_BANK_NAME,
    SETTINGS_KEYS.TRANSFER_BANK_VISIBLE,
    SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_HOLDER,
    SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NUMBER,
    SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NAME,
    SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_VISIBLE,
  ]);

  const updateMutation = useUpdateSetting();
  const createMutation = useCreateSetting();

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
  const [transferBankHolder, setTransferBankHolder] = useState('');
  const [originalTransferBankHolder, setOriginalTransferBankHolder] = useState('');
  const [transferBankNumber, setTransferBankNumber] = useState('');
  const [originalTransferBankNumber, setOriginalTransferBankNumber] = useState('');
  const [transferBankName, setTransferBankName] = useState('');
  const [originalTransferBankName, setOriginalTransferBankName] = useState('');
  const [transferBankVisible, setTransferBankVisibleState] = useState(true);
  const [originalTransferBankVisible, setOriginalTransferBankVisible] = useState(true);
  const [flexPayTransferBankHolder, setFlexPayTransferBankHolder] = useState('');
  const [originalFlexPayTransferBankHolder, setOriginalFlexPayTransferBankHolder] = useState('');
  const [flexPayTransferBankNumber, setFlexPayTransferBankNumber] = useState('');
  const [originalFlexPayTransferBankNumber, setOriginalFlexPayTransferBankNumber] = useState('');
  const [flexPayTransferBankName, setFlexPayTransferBankName] = useState('');
  const [originalFlexPayTransferBankName, setOriginalFlexPayTransferBankName] = useState('');
  const [flexPayTransferBankVisible, setFlexPayTransferBankVisibleState] = useState(true);
  const [originalFlexPayTransferBankVisible, setOriginalFlexPayTransferBankVisible] = useState(true);
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
      const transferBankHolderSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.TRANSFER_BANK_HOLDER,
      );
      const transferBankNumberSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.TRANSFER_BANK_NUMBER,
      );
      const transferBankNameSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.TRANSFER_BANK_NAME,
      );
      const transferBankVisibleSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.TRANSFER_BANK_VISIBLE,
      );
      const flexPayTransferBankHolderSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_HOLDER,
      );
      const flexPayTransferBankNumberSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NUMBER,
      );
      const flexPayTransferBankNameSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NAME,
      );
      const flexPayTransferBankVisibleSetting = settings.find(
        (s) => s?.key === SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_VISIBLE,
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
      // Missing rows fall back to the shared defaults so admins see what is
      // currently printed on statements before saving an override.
      setTransferBankHolder(transferBankHolderSetting?.value ?? TRANSFER_BANK_DEFAULTS.HOLDER);
      setOriginalTransferBankHolder(transferBankHolderSetting?.value ?? TRANSFER_BANK_DEFAULTS.HOLDER);
      setTransferBankNumber(transferBankNumberSetting?.value ?? TRANSFER_BANK_DEFAULTS.NUMBER);
      setOriginalTransferBankNumber(transferBankNumberSetting?.value ?? TRANSFER_BANK_DEFAULTS.NUMBER);
      setTransferBankName(transferBankNameSetting?.value ?? TRANSFER_BANK_DEFAULTS.NAME);
      setOriginalTransferBankName(transferBankNameSetting?.value ?? TRANSFER_BANK_DEFAULTS.NAME);
      // Mirrors the backend parse: missing/empty row => default visible;
      // otherwise only an explicit "true" keeps the block visible.
      const rawBankVisible = transferBankVisibleSetting?.value?.trim().toLowerCase();
      const bankVisible = rawBankVisible === undefined || rawBankVisible === '' || rawBankVisible === 'true';
      setTransferBankVisibleState(bankVisible);
      setOriginalTransferBankVisible(bankVisible);

      // FlexPay rows mirror the backend fallback: an unset (or empty) FlexPay
      // field shows the weekly payroll account so admins see what is currently
      // printed before saving a FlexPay-only override.
      const flexPayHolder =
        flexPayTransferBankHolderSetting?.value ||
        transferBankHolderSetting?.value ||
        TRANSFER_BANK_DEFAULTS.HOLDER;
      const flexPayNumber =
        flexPayTransferBankNumberSetting?.value ||
        transferBankNumberSetting?.value ||
        TRANSFER_BANK_DEFAULTS.NUMBER;
      const flexPayName =
        flexPayTransferBankNameSetting?.value ||
        transferBankNameSetting?.value ||
        TRANSFER_BANK_DEFAULTS.NAME;
      setFlexPayTransferBankHolder(flexPayHolder);
      setOriginalFlexPayTransferBankHolder(flexPayHolder);
      setFlexPayTransferBankNumber(flexPayNumber);
      setOriginalFlexPayTransferBankNumber(flexPayNumber);
      setFlexPayTransferBankName(flexPayName);
      setOriginalFlexPayTransferBankName(flexPayName);
      const rawFlexPayVisible = flexPayTransferBankVisibleSetting?.value?.trim().toLowerCase();
      const flexPayVisible =
        rawFlexPayVisible === undefined || rawFlexPayVisible === ''
          ? bankVisible
          : rawFlexPayVisible === 'true';
      setFlexPayTransferBankVisibleState(flexPayVisible);
      setOriginalFlexPayTransferBankVisible(flexPayVisible);
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

  // Upserts one string setting. The row may not exist yet on an environment
  // where the key was never seeded, so a miss creates it.
  const upsertStringSetting = (key: string, value: string, onSaved: () => void) => {
    const setting = settings?.find((s) => s?.key === key);
    if (setting?.id) {
      updateMutation.mutate({ id: setting.id, data: { value } }, { onSuccess: onSaved });
    } else {
      createMutation.mutate({ key, value, value_type: 'string' }, { onSuccess: onSaved });
    }
  };

  const upsertVisibleSetting = (key: string, visible: boolean, onSaved: () => void) => {
    upsertStringSetting(key, visible ? 'true' : 'false', onSaved);
  };

  const handleSaveTransferBank = () => {
    upsertStringSetting(SETTINGS_KEYS.TRANSFER_BANK_HOLDER, transferBankHolder, () =>
      setOriginalTransferBankHolder(transferBankHolder),
    );
    upsertStringSetting(SETTINGS_KEYS.TRANSFER_BANK_NUMBER, transferBankNumber, () =>
      setOriginalTransferBankNumber(transferBankNumber),
    );
    upsertStringSetting(SETTINGS_KEYS.TRANSFER_BANK_NAME, transferBankName, () =>
      setOriginalTransferBankName(transferBankName),
    );
  };

  const handleSaveTransferBankVisible = () => {
    upsertVisibleSetting(SETTINGS_KEYS.TRANSFER_BANK_VISIBLE, transferBankVisible, () =>
      setOriginalTransferBankVisible(transferBankVisible),
    );
  };

  const handleSaveFlexPayTransferBank = () => {
    upsertStringSetting(SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_HOLDER, flexPayTransferBankHolder, () =>
      setOriginalFlexPayTransferBankHolder(flexPayTransferBankHolder),
    );
    upsertStringSetting(SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NUMBER, flexPayTransferBankNumber, () =>
      setOriginalFlexPayTransferBankNumber(flexPayTransferBankNumber),
    );
    upsertStringSetting(SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_NAME, flexPayTransferBankName, () =>
      setOriginalFlexPayTransferBankName(flexPayTransferBankName),
    );
  };

  const handleSaveFlexPayTransferBankVisible = () => {
    upsertVisibleSetting(
      SETTINGS_KEYS.FLEXPAY_TRANSFER_BANK_VISIBLE,
      flexPayTransferBankVisible,
      () => setOriginalFlexPayTransferBankVisible(flexPayTransferBankVisible),
    );
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
    transferBankHolder,
    originalTransferBankHolder,
    transferBankNumber,
    originalTransferBankNumber,
    transferBankName,
    originalTransferBankName,
    transferBankVisible,
    originalTransferBankVisible,
    flexPayTransferBankHolder,
    originalFlexPayTransferBankHolder,
    flexPayTransferBankNumber,
    originalFlexPayTransferBankNumber,
    flexPayTransferBankName,
    originalFlexPayTransferBankName,
    flexPayTransferBankVisible,
    originalFlexPayTransferBankVisible,
    loadError,
    // Covers both save paths: a missing settings row (e.g. the not-yet-seeded
    // transfer_bank_visible toggle) is created, an existing one updated.
    isSaving: updateMutation.isPending || createMutation.isPending,
    isLoading,
    setWeeklyPaymentPercentage,
    setMonthlyPaymentPercentage,
    setPartnerCompany,
    setBulkTransferWorkbookLimitVnd,
    setSelfCheckInAdvancePercentage,
    setSelfCheckInAdvanceHoldHours,
    setTransferBankHolder,
    setTransferBankNumber,
    setTransferBankName,
    setTransferBankVisible: setTransferBankVisibleState,
    setFlexPayTransferBankHolder,
    setFlexPayTransferBankNumber,
    setFlexPayTransferBankName,
    setFlexPayTransferBankVisible: setFlexPayTransferBankVisibleState,
    handleSaveWeeklyPayment,
    handleSaveMonthlyPayment,
    handleSavePartnerCompany,
    handleSaveBulkTransferWorkbookLimitVnd,
    handleSaveSelfCheckInAdvancePercentage,
    handleSaveSelfCheckInAdvanceHoldHours,
    handleSaveTransferBank,
    handleSaveTransferBankVisible,
    handleSaveFlexPayTransferBank,
    handleSaveFlexPayTransferBankVisible,
    retryLoading,
  };
}
