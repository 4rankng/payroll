import { useState, useEffect, useCallback } from 'react';
import { useForm, useFieldArray } from 'react-hook-form';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Trash2, Plus, AlertTriangle } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ledgerService } from '@/services/api/ledger.service';
import { dateToString } from '@/utils/dateHelpers';
import { useLedgerManagement } from '@/hooks/ledger/useLedgerManagement';
import { useLedgerAccountOptions } from '@/hooks/ledger/useLedgerMetadata';
import type { CreateLedgerEntry, CreateTransactionRequest } from '@/types/api/financial.types';
import { UserSelector } from '@/components/ui/user-selector';

interface Project {
  id: number;
  name: string;
}

interface DoubleEntryModalProps {
  isOpen: boolean;
  onClose: () => void;
  projects: Project[];
}

interface EntryFormData {
  account: string;
  party: string;
  description: string;
  debit: string;
  credit: string;
  project_id?: string;
  // For capital contributions, track selected user id separately for UI
  user_id?: string;
}

interface TransactionFormData {
  date: string;
  entries: EntryFormData[];
}

// Common transaction templates
const TRANSACTION_TEMPLATES = [
  {
    name: 'Góp vốn',
    entries: [
      { account: 'cash', debit: '0', credit: '0', party: '', description: 'Nhận vốn góp' },
      { account: 'capital', debit: '0', credit: '0', party: '', description: 'Ghi nhận vốn chủ sở hữu' },
    ],
  },
  {
    name: 'Thu tiền khách hàng',
    entries: [
      { account: 'cash', debit: '0', credit: '0', party: '', description: 'Thu tiền từ khách hàng' },
      { account: 'receivable', debit: '0', credit: '0', party: '', description: 'Giảm Phải thu' },
    ],
  },
  {
    name: 'Trả lương nhân viên',
    entries: [
      { account: 'expense', debit: '0', credit: '0', party: '', description: 'Chi trả lương' },
      { account: 'cash', debit: '0', credit: '0', party: '', description: 'Giảm tiền mặt' },
    ],
  },
  {
    name: 'Ghi nhận doanh thu dự án',
    entries: [
      { account: 'receivable', debit: '0', credit: '0', party: '', description: 'Phát sinh Phải thu' },
      { account: 'revenue', debit: '0', credit: '0', party: '', description: 'Ghi nhận doanh thu' },
    ],
  },
  {
    name: 'Chi phí vận hành',
    entries: [
      { account: 'expense', debit: '0', credit: '0', party: '', description: 'Chi phí hoạt động' },
      { account: 'cash', debit: '0', credit: '0', party: '', description: 'Giảm tiền mặt' },
    ],
  },
];

export function DoubleEntryModal({ isOpen, onClose, projects }: DoubleEntryModalProps) {
  const { createTransaction, isCreatingTransaction } = useLedgerManagement();
  const { accountOptions, isLoading: isLoadingAccounts } = useLedgerAccountOptions();
  const [selectedTemplate, setSelectedTemplate] = useState<string>('');

  const {
    control,
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    getValues,
    formState: { errors, isSubmitting },
  } = useForm<TransactionFormData>({
    defaultValues: {
      date: dateToString(new Date()),
      entries: [
        { account: '', party: '', description: '', debit: '0', credit: '0', project_id: 'none', user_id: '' },
        { account: '', party: '', description: '', debit: '0', credit: '0', project_id: 'none', user_id: '' },
      ],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: 'entries',
  });

  const watchedEntries = watch('entries');

  // Calculate totals
  const totalDebits = watchedEntries.reduce((sum, entry) => {
    const debit = parseFloat(entry.debit?.replace(/,/g, '') || '0');
    return sum + (isNaN(debit) ? 0 : debit);
  }, 0);

  const totalCredits = watchedEntries.reduce((sum, entry) => {
    const credit = parseFloat(entry.credit?.replace(/,/g, '') || '0');
    return sum + (isNaN(credit) ? 0 : credit);
  }, 0);

  const isBalanced = totalDebits === totalCredits && totalDebits > 0;

  useEffect(() => {
    if (isOpen) {
      reset({
        date: dateToString(new Date()),
        entries: [
          { account: '', party: '', description: '', debit: '0', credit: '0', project_id: 'none', user_id: '' },
          { account: '', party: '', description: '', debit: '0', credit: '0', project_id: 'none', user_id: '' },
        ],
      });
      setSelectedTemplate('');
    }
  }, [isOpen, reset]);

  const handleTemplateSelect = (templateName: string) => {
    const template = TRANSACTION_TEMPLATES.find(t => t.name === templateName);
    if (!template) return;

    setSelectedTemplate(templateName);

    // Reset entries with template data
    const newEntries = template.entries.map(entry => ({
      account: entry.account,
      party: entry.party,
      description: entry.description,
      debit: entry.debit,
      credit: entry.credit,
      project_id: 'none',
      user_id: '',
    }));

    setValue('entries', newEntries);
  };

  const maybeSyncCapital = useCallback((index: number, field: 'debit' | 'credit', formattedValue: string) => {
    if (selectedTemplate !== 'Góp vốn') return;

    const entriesNow = getValues('entries');
    const cashIndex = entriesNow.findIndex(e => e.account === 'cash');
    const capitalIndex = entriesNow.findIndex(e => e.account === 'capital');
    if (cashIndex === -1 || capitalIndex === -1) return;

    if (index === cashIndex && field === 'debit') {
      setValue(`entries.${cashIndex}.credit`, '0');
      setValue(`entries.${capitalIndex}.debit`, '0');
      setValue(`entries.${capitalIndex}.credit`, formattedValue || '0');
    } else if (index === capitalIndex && field === 'credit') {
      setValue(`entries.${cashIndex}.credit`, '0');
      setValue(`entries.${capitalIndex}.debit`, '0');
      setValue(`entries.${cashIndex}.debit`, formattedValue || '0');
    }
  }, [getValues, selectedTemplate, setValue]);

  const handleAmountChange = (index: number, field: 'debit' | 'credit', value: string) => {
    let formattedValue = value.replace(/[^0-9]/g, '');
    if (formattedValue) {
      formattedValue = parseInt(formattedValue).toLocaleString('vi-VN');
    }
    setValue(`entries.${index}.${field}`, formattedValue);

    // Auto-sync opposite side for capital contribution template
    maybeSyncCapital(index, field, formattedValue);
  };

  const addEntry = () => {
    append({ account: '', party: '', description: '', debit: '0', credit: '0', project_id: 'none', user_id: '' });
  };

  const removeEntry = (index: number) => {
    if (fields.length > 2) {
      remove(index);
    }
  };

  const onSubmit = async (data: TransactionFormData) => {
    try {
      if (!isBalanced) {
        throw new Error('Tổng nợ phải bằng tổng có');
      }

      const entries: CreateLedgerEntry[] = data.entries.map(entry => {
        const debit = parseFloat(entry.debit.replace(/,/g, ''));
        const credit = parseFloat(entry.credit.replace(/,/g, ''));

        return {
          date: data.date,
          account: entry.account,
          party: entry.party,
          description: entry.description,
          debit: isNaN(debit) ? 0 : debit,
          credit: isNaN(credit) ? 0 : credit,
          project_id: entry.project_id && entry.project_id !== 'none' ? parseInt(entry.project_id) : undefined,
        };
      });

      // Validate all entries
      for (const entry of entries) {
        const validation = ledgerService.validateEntry(entry);
        if (!validation.isValid) {
          throw new Error(validation.errors.join(', '));
        }
      }

      const transactionData: CreateTransactionRequest = { entries };
      await createTransaction(transactionData);
      onClose();
    } catch (error: unknown) {
      console.error('Transaction submission error:', error);
    }
  };

  const isLoading = isCreatingTransaction || isLoadingAccounts;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="typography-h3">Tạo Bút Toán Kép</DialogTitle>
          <DialogDescription className="typography-body-medium text-muted-foreground">
            Tạo giao dịch với nhiều bút toán, đảm bảo nguyên tắc cân bằng kế toán
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Templates */}
          <div className="space-y-3">
            <Label className="typography-body-medium font-medium">Mẫu giao dịch thông dụng</Label>
            <div className="flex flex-wrap gap-2">
              {TRANSACTION_TEMPLATES.map((template) => (
                <Button
                  key={template.name}
                  type="button"
                  variant={selectedTemplate === template.name ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => handleTemplateSelect(template.name)}
                  className="typography-body-small"
                >
                  {template.name}
                </Button>
              ))}
            </div>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
            {/* Date */}
            <div className="space-y-2">
              <Label htmlFor="date" className="typography-body-medium font-medium">
                Ngày Ghi Sổ *
              </Label>
              <Input
                id="date"
                type="date"
                {...register('date', { required: 'Ngày là bắt buộc' })}
                className="max-w-xs typography-body-medium"
              />
              {errors.date && (
                <p className="typography-body-small text-destructive">{errors.date.message}</p>
              )}
            </div>

            {/* Entries */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Label className="typography-body-medium font-medium">Các Bút Toán</Label>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={addEntry}
                  className="flex items-center gap-1 typography-body-small"
                >
                  <Plus className="h-4 w-4" />
                  Thêm bút toán
                </Button>
              </div>

              <div className="border rounded-xl overflow-hidden">
                <div className="bg-muted/50 p-3 border-b">
                  <div className="grid grid-cols-12 gap-3 typography-body-small font-medium text-muted-foreground">
                    <div className="col-span-2">Tài khoản</div>
                    <div className="col-span-2">Đối tượng</div>
                    <div className="col-span-3">Diễn giải</div>
                    <div className="col-span-2">Nợ (đ)</div>
                    <div className="col-span-2">Có (đ)</div>
                    <div className="col-span-1">Dự án</div>
                  </div>
                </div>

                <div className="divide-y">
                  {fields.map((field, index) => (
                    <div key={field.id} className="p-3">
                      <div className="grid grid-cols-12 gap-3 items-start">
                        {/* Account */}
                        <div className="col-span-2">
                          <Select
                            value={watch(`entries.${index}.account`)}
                            onValueChange={(value) => setValue(`entries.${index}.account`, value)}
                          >
                            <SelectTrigger className="min-h-11 typography-body-small">
                              <SelectValue placeholder="Chọn tài khoản" />
                            </SelectTrigger>
                            <SelectContent>
                              {accountOptions.map((option) => (
                                <SelectItem key={option.value} value={option.value} className="typography-body-small">
                                  {option.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>

                        {/* Party / User (capital contributions fetch admin only) */}
                        <div className="col-span-2">
                          {watch(`entries.${index}.account`) === 'capital' ? (
                            <UserSelector
                              value={watch(`entries.${index}.user_id`) || ''}
                              onValueChange={(val) => setValue(`entries.${index}.user_id`, val)}
                              onUserChange={(user) => {
                                // Sync party name from selected user for API
                                setValue(`entries.${index}.party`, user?.fullname || '');
                              }}
                              placeholder="Chọn người góp vốn"
                              className="min-h-11 typography-body-small"
                              filterRole="admin"
                            />
                          ) : (
                            <Input
                              {...register(`entries.${index}.party`, { required: 'Đối tượng là bắt buộc' })}
                              placeholder="Đối tượng"
                              className="min-h-11 typography-body-small"
                            />
                          )}
                        </div>

                        {/* Description */}
                        <div className="col-span-3">
                          <Textarea
                            {...register(`entries.${index}.description`, { required: 'Diễn giải là bắt buộc' })}
                            placeholder="Diễn giải"
                            rows={1}
                            className="min-h-11 resize-none typography-body-small"
                          />
                        </div>

                        {/* Debit */}
                        <div className="col-span-2">
                          <Input
                            value={watch(`entries.${index}.debit`)}
                            onChange={(e) => handleAmountChange(index, 'debit', e.target.value)}
                            placeholder="0"
                            className="min-h-11 text-right typography-body-small"
                          />
                        </div>

                        {/* Credit */}
                        <div className="col-span-2">
                          <Input
                            value={watch(`entries.${index}.credit`)}
                            onChange={(e) => handleAmountChange(index, 'credit', e.target.value)}
                            placeholder="0"
                            className="min-h-11 text-right typography-body-small"
                          />
                        </div>

                        {/* Project */}
                        <div className="col-span-1 flex items-center gap-2">
                          <Select
                            value={watch(`entries.${index}.project_id`)}
                            onValueChange={(value) => setValue(`entries.${index}.project_id`, value)}
                          >
                            <SelectTrigger className="min-h-11 typography-body-small">
                              <SelectValue placeholder="-" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="none" className="typography-body-small">Không</SelectItem>
                              {projects.map((project) => (
                                <SelectItem key={project.id} value={project.id.toString()} className="typography-body-small">
                                  {project.name}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>

                          {fields.length > 2 && (
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => removeEntry(index)}
                              className="h-11 w-11 p-0 text-destructive hover:text-destructive"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* Balance Summary */}
            <div className="bg-muted/50 p-4 rounded-xl">
              <div className="flex items-center justify-between mb-3">
                <span className="typography-body-medium font-medium">Tổng cộng:</span>
                {isBalanced ? (
                  <Badge variant="default" className="bg-green-100 text-green-800 typography-body-small">
                    Cân bằng
                  </Badge>
                ) : (
                  <Badge variant="destructive" className="typography-body-small">
                    Chưa cân bằng
                  </Badge>
                )}
              </div>
              <div className="grid grid-cols-2 gap-4 typography-body-small">
                <div className="flex justify-between">
                  <span>Tổng Nợ:</span>
                  <span className="font-medium">{totalDebits.toLocaleString('vi-VN')} đ</span>
                </div>
                <div className="flex justify-between">
                  <span>Tổng Có:</span>
                  <span className="font-medium">{totalCredits.toLocaleString('vi-VN')} đ</span>
                </div>
              </div>
              {!isBalanced && totalDebits + totalCredits > 0 && (
                <Alert className="mt-3">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription className="typography-body-small">
                    Tổng nợ phải bằng tổng có để đảm bảo nguyên tắc cân bằng kế toán
                  </AlertDescription>
                </Alert>
              )}
            </div>
          </form>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isLoading}
            className="typography-body-medium"
          >
            Đóng
          </Button>
          <Button
            type="submit"
            disabled={isLoading || !isBalanced}
            onClick={handleSubmit(onSubmit)}
            className="typography-body-medium"
            variant="default"
          >
            {isLoading ? 'Đang tạo...' : 'Tạo giao dịch'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
