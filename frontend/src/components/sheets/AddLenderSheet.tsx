import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { formatVietnameseName } from '@/lib/validation';
import { Textarea } from '@/components/ui/textarea';
import { BankSelector } from '@/components/ui/bank-selector';
import { X, UserPlus } from 'lucide-react';
import { useCreateLender } from '@/hooks/api/useLoans';
import { SlideSheetTemplate } from './templates/SlideSheetTemplate';
import { authManager } from '@/lib/auth';
import type { CreateLenderRequest } from '@/types/api/loan.types';
import type { Bank } from '@/types/api/bank.types';

interface AddLenderSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AddLenderSheet({ isOpen, onClose }: AddLenderSheetProps) {
  const createLender = useCreateLender();
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';

  const [formData, setFormData] = useState<CreateLenderRequest>({
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

  useEffect(() => {
    if (isOpen) {
      // Reset form when sheet opens
      setFormData({
        name: '',
        cccd: '',
        email: '',
        mobile: '',
        notes: '',
        bank_id: undefined,
        bank_account_number: '',
        bank_account_name: '',
      });
      setSelectedBank(null);
      setErrors({});
    }
  }, [isOpen]);

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
    const requestData: CreateLenderRequest = {
      name: formData.name.trim(),
    };

    // Add optional fields only if they have values
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
      await createLender.mutateAsync(requestData);
      onClose();
    } catch (error) {
      // Error handling is done in the mutation
    }
  };

  const handleInputChange = (field: keyof CreateLenderRequest, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: '' }));
    }
  };

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      title="Thêm chủ nợ"
      description="Tạo mới thông tin chủ nợ"
      compact
      footer={
        <div className="grid grid-cols-2 gap-3 w-full px-4">
          <Button variant="outline" onClick={onClose} className="w-full">
            <X className="w-4 h-4 mr-2" />
            Hủy
          </Button>
          <Button variant="default" onClick={handleSave} disabled={createLender.isPending} className="w-full">
            {createLender.isPending ? 'Đang thêm...' : 'Thêm'}
          </Button>
        </div>
      }
    >
      <div className="space-y-5 mt-2">
        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="name" className="typography-label-medium">Tên chủ nợ *</Label>
            <Input
              id="name"
              value={formData.name}
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
      </div>
    </SlideSheetTemplate>
  );
}

export default AddLenderSheet;
