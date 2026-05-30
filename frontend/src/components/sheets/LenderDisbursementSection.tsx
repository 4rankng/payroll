import { Label } from '@/components/ui/label';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
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
        <Select value={lenderId} onValueChange={onLenderChange}>
          <SelectTrigger className={`h-11 ${errors.lender_id ? 'input-error' : ''}`}>
            <SelectValue placeholder="Chọn chủ nợ" />
          </SelectTrigger>
          <SelectContent>
            {lenders.map((l) => (
              <SelectItem key={l.id} value={l.id.toString()}>{l.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.lender_id && <p className="typography-body-small text-financial-negative mt-1">{errors.lender_id}</p>}
      </div>
    </div>
  );
}
