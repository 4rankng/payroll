import { useState, useEffect, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { formatVietnameseName } from '@/lib/validation';
import { Textarea } from '@/components/ui/textarea';
import { BankSelector } from '@/components/ui/bank-selector';
import { X, Save, Trash2 } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { useUpdateLender, useLender, useDeleteLender } from '@/hooks/api/useLoans';
import { SlideSheetTemplate } from './templates/SlideSheetTemplate';
import { authManager } from '@/lib/auth';
import type { UpdateLenderRequest } from '@/types/api/loan.types';
import type { Bank } from '@/types/api/bank.types';
import { ConfirmationModal } from '@/components/modals/ConfirmationModal';

interface EditLenderSheetProps {
  isOpen: boolean;
  onClose: () => void;
  lenderId: number;
}

export function EditLenderSheet({ isOpen, onClose, lenderId }: EditLenderSheetProps) {
  const updateLender = useUpdateLender();
  const deleteLender = useDeleteLender();
  const { data: lenderData, isLoading } = useLender(lenderId, isOpen);
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';

  const [formData, setFormData] = useState<UpdateLenderRequest>({
    name: '',
    cccd: '',
    email: '',
    mobile: '',
    notes: '',
    bank_id: undefined,
    bank_account_number: '',
    bank_account_name: '',
  });

  const [selectedBank, setSelectedBank] = useState<Bank | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Populate form when lender data is loaded
  useEffect(() => {
    if (isOpen && lenderData) {
      setFormData({
        name: lenderData.name || '',
        cccd: lenderData.cccd || '',
        email: lenderData.email || '',
        mobile: lenderData.mobile || '',
        notes: lenderData.notes || '',
        bank_id: lenderData.bank_id || undefined,
        bank_account_number: lenderData.bank_account_number || '',
        bank_account_name: lenderData.bank_account_name || '',
      });
      setSelectedBank(lenderData.bank || null);
      setErrors({});
    }
  }, [isOpen, lenderData]);

  const handleSave = async () => {
    const newErrors: Record<string, string> = {};

    // Required field
    if (!formData.name?.trim()) {
      newErrors.name = 'Tên chủ nợ là bắt buộc';
    }

    // Optional fields validation
    if (formData.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Email không hợp lệ';
    }

    if (formData.mobile && !/^\d+$/.test(formData.mobile.replace(/\s/g, ''))) {
      newErrors.mobile = 'Số điện thoại chỉ được chứa số';
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

    // Only include fields that have values
    const requestData: UpdateLenderRequest = {};

    if (formData.name?.trim()) {
      requestData.name = formData.name.trim();
    }
    if (formData.cccd?.trim()) {
      requestData.cccd = formData.cccd.trim();
    }
    if (formData.email?.trim()) {
      requestData.email = formData.email.trim();
    }
    if (formData.mobile?.trim()) {
      requestData.mobile = formData.mobile.trim();
    }
    if (formData.notes?.trim()) {
      requestData.notes = formData.notes.trim();
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

    try {
      await updateLender.mutateAsync({ id: lenderId, data: requestData });
      onClose();
    } catch (error) {
      // Error handling is done in the mutation
    }
  };

  const handleInputChange = (field: keyof UpdateLenderRequest, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);

  const handleDeleteOpen = useCallback(() => {
    setIsDeleteModalOpen(true);
  }, []);

  const handleDeleteModalClose = useCallback(() => {
    setIsDeleteModalOpen(false);
  }, []);

  const handleDeleteConfirm = useCallback(async () => {
    if (!lenderId) return;
    try {
      await deleteLender.mutateAsync(lenderId);
      setIsDeleteModalOpen(false);
      onClose();
    } catch (error) {
      // Error toast is handled inside the hook
    }
  }, [deleteLender, lenderId, onClose]);

  const hasActiveLoans = (lenderData?.summary?.active_loans_count ?? 0) > 0;
  const deleteDisabled = deleteLender.isPending || isLoading || hasActiveLoans;

  return (
    <>
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      title="Chỉnh sửa chủ nợ"
      description={lenderData?.name || 'Đang tải...'}
      compact
      footer={
        <div className="grid grid-cols-3 gap-3 w-full px-4">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="destructive"
                  onClick={handleDeleteOpen}
                  disabled={deleteDisabled}
                  className="w-full"
                  aria-label="Xóa người cho vay"
                >
                  <Trash2 className="w-4 h-4 mr-2" />
                  Xóa
                </Button>
              </TooltipTrigger>
              <TooltipContent className="max-w-72" side="top">
                <div className="typography-body-medium">
                  {hasActiveLoans
                    ? `Không thể xóa vì còn khoản vay hiện tại (${lenderData?.summary?.active_loans_count}). Hãy đóng tất cả khoản vay trước.`
                    : 'Xóa người cho vay khỏi danh sách.'}
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <Button variant="outline" onClick={onClose} className="w-full">
            <X className="w-4 h-4 mr-2" />
            Hủy
          </Button>
          <Button
            variant="default"
            onClick={handleSave}
            disabled={updateLender.isPending || isLoading}
            className="w-full"
          >
            <Save className="w-4 h-4 mr-2" />
            {updateLender.isPending ? 'Đang lưu...' : 'Lưu'}
          </Button>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="typography-body-medium text-muted-foreground">Đang tải thông tin...</div>
        </div>
      ) : (
        <div className="space-y-5 mt-2">
          <div className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="name" className="typography-label-medium">Tên chủ nợ *</Label>
              <Input
                id="name"
                value={formData.name || ''}
                onChange={(e) => handleInputChange('name', e.target.value)}
                onBlur={(e) => {
                  const formatted = formatVietnameseName(e.target.value);
                  if (formatted !== e.target.value) {
                    handleInputChange('name', formatted);
                  }
                }}
                className={`h-9 ${errors.name ? 'border-red-500' : ''}`}
                placeholder="Nhập tên chủ nợ"
              />
              {errors.name && <p className="typography-body-small text-financial-negative mt-1">{errors.name}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="cccd" className="typography-label-medium">Số CCCD</Label>
              <Input
                id="cccd"
                value={formData.cccd || ''}
                onChange={(e) => handleInputChange('cccd', e.target.value)}
                className="h-9"
                placeholder="Nhập số CCCD"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="email" className="typography-label-medium">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={formData.email || ''}
                  onChange={(e) => handleInputChange('email', e.target.value)}
                  className={`h-9 ${errors.email ? 'border-red-500' : ''}`}
                  placeholder="email@example.com"
                />
                {errors.email && <p className="typography-body-small text-financial-negative mt-1">{errors.email}</p>}
              </div>

              <div className="space-y-2">
                <Label htmlFor="mobile" className="typography-label-medium">Số điện thoại</Label>
                <Input
                  id="mobile"
                  value={formData.mobile || ''}
                  onChange={(e) => handleInputChange('mobile', e.target.value)}
                  className={`h-9 ${errors.mobile ? 'border-red-500' : ''}`}
                  placeholder="0912345678"
                />
                {errors.mobile && <p className="typography-body-small text-financial-negative mt-1">{errors.mobile}</p>}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="notes" className="typography-label-medium">Ghi chú</Label>
              <Textarea
                id="notes"
                value={formData.notes || ''}
                onChange={(e) => handleInputChange('notes', e.target.value)}
                className="min-h-[80px]"
                placeholder="Nhập ghi chú (nếu có)"
              />
            </div>
          </div>

          {/* Bank Details Section */}
          <div className="space-y-3 pt-2 border-t">
            <h3 className="typography-label-large">Thông tin ngân hàng</h3>
            <div className="space-y-2">
              <Label className="typography-label-medium">Ngân hàng</Label>
              <BankSelector
                value={selectedBank}
                onSelect={(bank) => {
                  setSelectedBank(bank);
                  handleInputChange('bank_id', bank?.id != null ? String(bank.id) : '');
                }}
                className={errors.bank_id ? 'border-red-500' : ''}
                canCreateBank={isAdmin}
              />
              {errors.bank_id && <p className="typography-body-small text-financial-negative mt-1">{errors.bank_id}</p>}
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="bank_account_number" className="typography-label-medium">Số tài khoản</Label>
                <Input
                  id="bank_account_number"
                  value={formData.bank_account_number || ''}
                  onChange={(e) => handleInputChange('bank_account_number', e.target.value)}
                  className={`h-9 ${errors.bank_account_number ? 'border-red-500' : ''}`}
                  placeholder="1234567890"
                />
                {errors.bank_account_number && <p className="typography-body-small text-financial-negative mt-1">{errors.bank_account_number}</p>}
              </div>

              <div className="space-y-2">
                <Label htmlFor="bank_account_name" className="typography-label-medium">Tên chủ tài khoản</Label>
                <Input
                  id="bank_account_name"
                  value={formData.bank_account_name || ''}
                  onChange={(e) => handleInputChange('bank_account_name', e.target.value)}
                  onBlur={(e) => {
                    const formatted = formatVietnameseName(e.target.value);
                    if (formatted !== e.target.value) {
                      handleInputChange('bank_account_name', formatted);
                    }
                  }}
                  className={`h-9 ${errors.bank_account_name ? 'border-red-500' : ''}`}
                  placeholder="Nguyễn Văn A"
                />
                {errors.bank_account_name && <p className="typography-body-small text-financial-negative mt-1">{errors.bank_account_name}</p>}
              </div>
            </div>
          </div>

          <div className="space-y-3">
            {/* Display financial summary if available */}
            {lenderData?.summary && (
              <div className="pt-4 border-t space-y-3">
                <h3 className="typography-label-large">Thông tin tài chính</h3>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <p className="typography-body-small text-muted-foreground">Tổng vay</p>
                    <p className="typography-body-medium font-semibold">
                      {new Intl.NumberFormat('vi-VN', { minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(lenderData.summary.total_principal_borrowed)}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="typography-body-small text-muted-foreground">Dư nợ</p>
                    <p className="typography-body-medium font-semibold text-orange-600">
                      {new Intl.NumberFormat('vi-VN', { minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(lenderData.summary.outstanding_principal)}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="typography-body-small text-muted-foreground">Đã trả</p>
                    <p className="typography-body-medium font-semibold text-green-600">
                      {new Intl.NumberFormat('vi-VN', { minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(lenderData.summary.total_principal_repaid)}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="typography-body-small text-muted-foreground">Khoản vay hiện tại</p>
                    <p className="typography-body-medium font-semibold">
                      {lenderData.summary.active_loans_count}
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </SlideSheetTemplate>

      {/* Xác nhận xóa người cho vay */}
      <ConfirmationModal
        isOpen={isDeleteModalOpen}
        onClose={handleDeleteModalClose}
        onConfirm={handleDeleteConfirm}
        title="Xóa người cho vay"
        description={
          <>
            Bạn có chắc chắn muốn xóa người cho vay <strong>{lenderData?.name}</strong>?
            <br />
            <br />
            Hành động này không thể hoàn thành.
          </>
        }
        confirmText="Xóa"
        cancelText="Hủy bỏ"
        variant="destructive"
        isLoading={deleteLender.isPending}
      />
    </>
  );
}

export default EditLenderSheet;
