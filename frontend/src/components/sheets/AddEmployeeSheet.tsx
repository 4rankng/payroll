import { useState, useEffect, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { BankSelector } from '@/components/ui/bank-selector';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { X, User, CreditCard, UserPlus, Plus, Loader2, AlertTriangle, Search, HandHeart } from 'lucide-react';
import { toast } from '@/components/ui/sonner';
import { CreateEmployeeData, DuplicateCheckMatch } from '@/types/api/employee.types';
import { useCreateEmployee, useDuplicateCheck, useRequestEmployeeAccess } from '@/hooks/api/useEmployees';
import { useDebounce } from '@/hooks/useDebounce';
import { SlideSheetTemplate } from './templates/SlideSheetTemplate';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { authManager } from '@/lib/auth';
import { validateFullname, formatVietnameseName, validateMobileNotCCCD } from '@/lib/validation';
import { cn } from '@/lib/utils';
import type { Bank } from '@/types/api/bank.types';
import type { ModalConfig } from '@/types/modal-config.types';

/** Match card: masked existing employee + claim action for partners. */
function DuplicateMatchCard({
  match,
  onClaim,
  isClaiming,
  showClaim,
}: {
  match: DuplicateCheckMatch;
  onClaim: (id: number) => void;
  isClaiming: boolean;
  showClaim: boolean;
}) {
  return (
    <div className="rounded-xl border border-border bg-muted/30 p-3 space-y-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-0.5">
          <p className="typography-body-medium font-semibold text-foreground truncate">
            {match.fullname}
          </p>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-muted-foreground">
            {match.cccd_masked && (
              <span className="typography-label-medium font-mono">{match.cccd_masked}</span>
            )}
            {match.mobile_masked && (
              <span className="typography-label-medium font-mono">{match.mobile_masked}</span>
            )}
            {match.email_masked && (
              <span className="typography-label-medium">{match.email_masked}</span>
            )}
          </div>
          <p className="typography-label-medium text-muted-foreground">
            {[
              match.created_by_name && `Quản lý: ${match.created_by_name}`,
              match.current_project_names?.length
                ? `Dự án: ${match.current_project_names.join(', ')}`
                : null,
              match.created_at
                ? `Tạo: ${new Date(match.created_at).toLocaleDateString('vi-VN')}`
                : null,
            ]
              .filter(Boolean)
              .join(' · ')}
          </p>
        </div>
        {showClaim && (
          <Button
            size="sm"
            variant="default"
            onClick={() => onClaim(match.id)}
            disabled={isClaiming}
            className="shrink-0 gap-1.5"
          >
            {isClaiming ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <HandHeart className="h-3.5 w-3.5" />
            )}
            Yêu cầu quản lý
          </Button>
        )}
      </div>
    </div>
  );
}

function FieldGroup({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-xl border bg-card p-4 space-y-3">
      {children}
    </div>
  );
}

function SectionTitle({ icon: Icon, label }: { icon: React.ElementType; label: string }) {
  return (
    <div className="flex items-center gap-2 mb-1">
      <div className="h-6 w-6 rounded-xl bg-primary/10 flex items-center justify-center">
        <Icon className="h-3.5 w-3.5 text-primary" />
      </div>
      <span className="typography-headline-small text-foreground">{label}</span>
    </div>
  );
}

function Field({ id, label, required, children }: { id?: string; label: string; required?: boolean; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <Label htmlFor={id} className="typography-label-medium text-muted-foreground">
        {label}{required && <span className="text-red-600 ml-0.5">*</span>}
      </Label>
      {children}
    </div>
  );
}

export const modalConfig: ModalConfig = {
  id: 'add-employee-sheet',
  name: 'Add Employee',
  description: 'Create a new employee',
  category: 'employee',
  permissions: {
    action: 'create',
    subject: 'Employee',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: [],
    example: '?modal=add-employee-sheet',
  },
  requiresAuth: true,
  encryptData: false,
};

interface AddEmployeeSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

interface AddEmployeeSheetComponentProps extends AddEmployeeSheetProps {}

function AddEmployeeSheetComponent({
  isOpen,
  onClose
}: AddEmployeeSheetComponentProps) {
  const createEmployee = useCreateEmployee();
  const queryClient = useQueryClient();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';
  const requestAccess = useRequestEmployeeAccess();

  const [formData, setFormData] = useState<CreateEmployeeData>({
    fullname: '',
    email: '',
    cccd: '',
    address: '',
    mobile: '',
    bank_id: undefined,
    bank_account_number: '',
    bank_account_name: '',
    date_of_birth: ''
  });

  const [selectedBank, setSelectedBank] = useState<Bank | null>(null);

  const [errors, setErrors] = useState<Record<string, string>>({});

  // --- Duplicate detection (global, pre-create) ---
  // Debounced check on the three stable identifiers; CCCD match is a hard
  // block (backend 409s anyway), mobile/email matches are soft warnings.
  const debouncedCCCD = useDebounce(formData.cccd?.trim() || '', 500);
  const debouncedMobile = useDebounce(formData.mobile?.replace(/\s/g, '') || '', 500);
  const debouncedEmail = useDebounce(formData.email?.trim() || '', 500);

  const checkParams = useMemo(() => {
    const params: { cccd?: string; mobile?: string; email?: string } = {};
    // Only check once identifiers are plausibly complete to avoid noisy
    // partial-input matches.
    if (debouncedCCCD.length >= 8) params.cccd = debouncedCCCD;
    if (debouncedMobile.length >= 9) params.mobile = debouncedMobile;
    if (debouncedEmail.includes('@') && debouncedEmail.includes('.')) params.email = debouncedEmail;
    return params;
  }, [debouncedCCCD, debouncedMobile, debouncedEmail]);

  const duplicateCheck = useDuplicateCheck(checkParams);
  const duplicates = useMemo(
    () => (duplicateCheck.data?.has_duplicates ? duplicateCheck.data.data : []),
    [duplicateCheck.data],
  );
  const cccdBlocked = useMemo(
    () => duplicates.some((m) => m.matched_on.includes('cccd')),
    [duplicates],
  );
  const softWarnings = useMemo(
    () => duplicates.filter((m) => !m.matched_on.includes('cccd')),
    [duplicates],
  );

  const handleClaim = async (employeeId: number) => {
    try {
      await requestAccess.mutateAsync(employeeId);
      toast({ title: 'Đã thêm nhân viên vào danh sách quản lý của bạn' });
      handleClose();
    } catch {
      // Error notification handled globally
    }
  };

  useEffect(() => {
    if (isOpen) {
      // Reset form when sheet opens
      setFormData({
        fullname: '',
        email: '',
        cccd: '',
        address: '',
        mobile: '',
        bank_id: undefined,
        bank_account_number: '',
        bank_account_name: '',
        date_of_birth: ''
      });
      setSelectedBank(null);
      setErrors({});
    }
  }, [isOpen]);

  const handleSave = async () => {
    const newErrors: Record<string, string> = {};

    // Required fields according to API documentation
    const fullnameValidation = validateFullname(formData.fullname || '');
    if (!fullnameValidation.valid) {
      newErrors.fullname = fullnameValidation.error || 'Họ tên không hợp lệ';
    }

    if (!formData.cccd?.trim()) {
      newErrors.cccd = 'CCCD là bắt buộc';
    } else if (!/^[a-zA-Z0-9]+$/.test(formData.cccd.replace(/\s/g, ''))) {
      newErrors.cccd = 'CCCD chỉ được chứa chữ cái và số';
    }

    // Optional fields validation
    if (formData.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Email không hợp lệ';
    }

    if (formData.mobile && !/^\d+$/.test(formData.mobile.replace(/\s/g, ''))) {
      newErrors.mobile = 'Số điện thoại chỉ được chứa số';
    } else if (formData.mobile) {
      const cccdGuard = validateMobileNotCCCD(formData.mobile);
      if (!cccdGuard.valid) {
        newErrors.mobile = cccdGuard.error!;
      }
    }

    if (formData.date_of_birth && !/^\d{4}-\d{2}-\d{2}$/.test(formData.date_of_birth)) {
      newErrors.date_of_birth = 'Ngày sinh phải có định dạng YYYY-MM-DD';
    }

    // Bank validation - if any bank field is provided, all are required
    const hasBankInfo = selectedBank || formData.bank_account_number?.trim() || formData.bank_account_name?.trim();
    if (hasBankInfo) {
      if (!selectedBank) {
        newErrors.bank_id = 'Vui lòng chọn ngân hàng';
      }
      if (!formData.bank_account_number?.trim()) {
        newErrors.bank_account_number = 'Số tài khoản là bắt buộc khi có thông tin ngân hàng';
      }
      if (!formData.bank_account_name?.trim()) {
        newErrors.bank_account_name = 'Tên chủ tài khoản là bắt buộc khi có thông tin ngân hàng';
      }
    }

    setErrors(newErrors);

    if (Object.keys(newErrors).length > 0) {
      return;
    }

    // Only include fields that have values, excluding empty optional fields
    const requestData: CreateEmployeeData = {
      fullname: formData.fullname,
      cccd: formData.cccd
    };

    // Add optional fields only if they have values
    if (formData.email?.trim()) {
      requestData.email = formData.email.trim();
    }
    if (formData.address?.trim()) {
      requestData.address = formData.address.trim();
    }
    if (formData.mobile?.trim()) {
      requestData.mobile = formData.mobile.trim();
    }
    if (selectedBank) {
      requestData.bank_id = selectedBank.id;
    }
    if (formData.bank_account_number?.trim()) {
      requestData.bank_account_number = formData.bank_account_number.trim();
    }
    if (formData.bank_account_name?.trim()) {
      requestData.bank_account_name = formData.bank_account_name.trim();
    }
    if (formData.date_of_birth?.trim()) {
      requestData.date_of_birth = formData.date_of_birth.trim();
    }

    try {
      await createEmployee.mutateAsync(requestData);
      handleClose();
    } catch (error) {
      // Error handling is done in the mutation
    }
  };

  const handleInputChange = (field: keyof CreateEmployeeData, value: string | number) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  // Wrapped close handler
  const handleClose = () => {
    onClose();
  };


  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={handleClose}
      avatar={{ icon: UserPlus }}
      title="Thêm nhân viên mới"
      description="Điền thông tin để tạo hồ sơ nhân viên"
      footer={
        <div className="grid w-full grid-cols-1 gap-2 min-[380px]:grid-cols-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClose}
            className="min-h-11 px-3 text-muted-foreground hover:text-foreground gap-1.5"
          >
            <X className="h-3.5 w-3.5" />
            Đóng
          </Button>
          <Button
            size="sm"
            onClick={handleSave}
            disabled={createEmployee.isPending || cccdBlocked}
            className="min-h-11 px-5 gap-1.5"
          >
            {createEmployee.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Plus className="h-3.5 w-3.5" />
            )}
            {createEmployee.isPending ? 'Đang thêm...' : 'Thêm nhân viên'}
          </Button>
        </div>
      }
    >
      <div className="space-y-3">
        {/* Personal Info Section */}
        <FieldGroup>
          <SectionTitle icon={User} label="Thông tin cá nhân" />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field id="fullname" label="Họ và tên" required>
              <Input
                id="fullname"
                placeholder="Nguyễn Văn A"
                value={formData.fullname || ''}
                onChange={(e) => handleInputChange("fullname", e.target.value)}
                onBlur={(e) => {
                  const formatted = formatVietnameseName(e.target.value);
                  if (formatted !== e.target.value) {
                    handleInputChange("fullname", formatted);
                  }
                }}
                className={cn("h-11 typography-body-medium", errors.fullname && "border-red-500")}
              />
              {errors.fullname && <p role="alert" className="typography-body-small text-financial-negative">{errors.fullname}</p>}
            </Field>

            <Field id="date_of_birth" label="Ngày sinh">
              <Input
                id="date_of_birth"
                type="date"
                value={formData.date_of_birth || ''}
                onChange={(e) => handleInputChange("date_of_birth", e.target.value)}
                className={cn("h-11 typography-body-medium", errors.date_of_birth && "border-red-500")}
              />
              {errors.date_of_birth && <p role="alert" className="typography-body-small text-financial-negative">{errors.date_of_birth}</p>}
            </Field>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field id="cccd" label="Số CCCD" required>
              <Input
                id="cccd"
                placeholder="001234567890"
                value={formData.cccd || ''}
                onChange={(e) => handleInputChange("cccd", e.target.value)}
                className={cn("h-11 typography-body-medium font-mono tracking-wide", errors.cccd && "border-red-500")}
              />
              {errors.cccd && <p role="alert" className="typography-body-small text-financial-negative">{errors.cccd}</p>}
            </Field>

            <Field id="mobile" label="Số điện thoại">
              <Input
                id="mobile"
                placeholder="0901234567"
                value={formData.mobile || ''}
                onChange={(e) => handleInputChange("mobile", e.target.value)}
                className={cn("h-11 typography-body-medium font-mono", errors.mobile && "border-red-500")}
              />
              {errors.mobile && <p role="alert" className="typography-body-small text-financial-negative">{errors.mobile}</p>}
            </Field>
          </div>

          <Field id="address" label="Địa chỉ">
            <Input
              id="address"
              placeholder="Số nhà, đường, quận/huyện, tỉnh/thành phố"
              value={formData.address || ''}
              onChange={(e) => handleInputChange("address", e.target.value)}
              className="h-11 typography-body-medium"
            />
          </Field>
        </FieldGroup>

        {/* Duplicate detection results */}
        {cccdBlocked && duplicates.length > 0 && (
          <Alert variant="destructive" className="space-y-3">
            <div className="flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
              <div className="space-y-1">
                <p className="typography-body-medium font-semibold">
                  Nhân viên với CCCD này đã tồn tại trong hệ thống
                </p>
                <AlertDescription>
                  Không thể tạo mới trùng CCCD. Nếu đây là người bạn cần quản lý, hãy yêu cầu quản lý nhân viên hiện có.
                </AlertDescription>
              </div>
            </div>
            <div className="space-y-2 pl-6">
              {duplicates.map((match) => (
                <DuplicateMatchCard
                  key={match.id}
                  match={match}
                  onClaim={handleClaim}
                  isClaiming={requestAccess.isPending}
                  showClaim={!isAdmin}
                />
              ))}
            </div>
          </Alert>
        )}

        {!cccdBlocked && softWarnings.length > 0 && (
          <Alert className="space-y-3 border-warning/40 bg-warning/5 text-foreground">
            <div className="flex items-start gap-2">
              <Search className="h-4 w-4 shrink-0 mt-0.5 text-warning" />
              <div className="space-y-1">
                <p className="typography-body-medium font-semibold">
                  Thông tin liên hệ này đã được sử dụng
                </p>
                <AlertDescription>
                  Có thể nhân viên này đã tồn tại. Bạn vẫn có thể tạo mới nếu chắc chắn đây là người khác.
                </AlertDescription>
              </div>
            </div>
            <div className="space-y-2 pl-6">
              {softWarnings.map((match) => (
                <DuplicateMatchCard
                  key={match.id}
                  match={match}
                  onClaim={handleClaim}
                  isClaiming={requestAccess.isPending}
                  showClaim={!isAdmin}
                />
              ))}
            </div>
          </Alert>
        )}

        {/* Bank Info Section */}
        <FieldGroup>
          <SectionTitle icon={CreditCard} label="Thông tin ngân hàng" />

          <Field label="Ngân hàng">
            <BankSelector
              value={selectedBank}
              onSelect={(bank) => {
                setSelectedBank(bank);
                handleInputChange("bank_id", bank?.id);
              }}
              placeholder="Chọn ngân hàng..."
              className={errors.bank_id ? 'border-red-500' : ''}
              canCreateBank={isAdmin}
            />
            {errors.bank_id && <p role="alert" className="typography-body-small text-financial-negative">{errors.bank_id}</p>}
          </Field>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field id="bank_account_number" label="Số tài khoản">
              <Input
                id="bank_account_number"
                placeholder="190123456789"
                value={formData.bank_account_number || ''}
                onChange={(e) => handleInputChange("bank_account_number", e.target.value)}
                className={cn("h-11 typography-body-medium font-mono", errors.bank_account_number && "border-red-500")}
              />
              {errors.bank_account_number && <p role="alert" className="typography-body-small text-financial-negative">{errors.bank_account_number}</p>}
            </Field>

            <Field id="bank_account_name" label="Tên chủ tài khoản">
              <Input
                id="bank_account_name"
                placeholder="NGUYỄN VĂN A"
                value={formData.bank_account_name || ''}
                onChange={(e) => handleInputChange("bank_account_name", e.target.value)}
                onBlur={(e) => {
                  const formatted = formatVietnameseName(e.target.value);
                  if (formatted !== e.target.value) {
                    handleInputChange("bank_account_name", formatted);
                  }
                }}
                className={cn("h-11 typography-body-medium", errors.bank_account_name && "border-red-500")}
              />
              {errors.bank_account_name && <p role="alert" className="typography-body-small text-financial-negative">{errors.bank_account_name}</p>}
            </Field>
          </div>
        </FieldGroup>
      </div>
    </SlideSheetTemplate>
  );
}

/**
 * Container component that provides URL synchronization for AddEmployeeSheet
 * Updated to work with modal navigation system
 */
export function AddEmployeeSheet({
  isOpen: propIsOpen,
  onClose: propOnClose
}: AddEmployeeSheetProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const queryClient = useQueryClient();

  // Check if modal is open via URL
  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'add_employee';

  // Use URL state if available, otherwise use props
  const isOpen = isOpenViaUrl || propIsOpen;

  // Close modal using centralized navigation or provided onClose
  const handleClose = () => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  };

  return (
    <AddEmployeeSheetComponent
      isOpen={isOpen}
      onClose={handleClose}
    />
  );
}

export default AddEmployeeSheet;
