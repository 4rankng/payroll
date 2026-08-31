import { Label } from '@/components/ui/label';
import { SearchableSelect } from '@/components/ui/searchable-select';
import type { Lender } from '@/types/api/loan.types';

interface LenderDisbursementSectionProps {
  lenderId: string;
  onLenderChange: (id: string) => void;
  lenders: Lender[];
  errors: Record<string, string>;
}

export function LenderDisbursementSection({
  lenderId,
  onLenderChange,
  lenders,
  errors
}: LenderDisbursementSectionProps) {
  return (
    <div className="flex flex-col gap-3">
      <div className="space-y-2">
        <Label className="typography-label-medium">Chủ nợ *</Label>
        <SearchableSelect
          value={lenderId}
          onChange={onLenderChange}
          options={lenders.map((l) => ({ value: l.id.toString(), label: l.name }))}
          placeholder="Chọn chủ nợ"
          searchPlaceholder="Tìm chủ nợ..."
          triggerClassName={errors.lender_id ? 'input-error' : undefined}
        />
        {errors.lender_id && <p className="typography-body-small text-financial-negative mt-1">{errors.lender_id}</p>}
      </div>
    </div>
  );
}
