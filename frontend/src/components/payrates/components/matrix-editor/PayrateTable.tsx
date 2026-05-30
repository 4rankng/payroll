import { Table, TableHeader as ShadcnTableHeader } from '@/components/ui/table';
import { TableHeader } from './TableHeader';
import { TableBody } from './TableBody';
import type { PayrateStructure, DayType, ValidationResult } from '../../types';

interface PayrateTableProps {
  rates: PayrateStructure;
  originalRates?: PayrateStructure;
  positions: string[];
  hourTypes: string[];
  readOnly?: boolean;
  validation?: ValidationResult;
  isFlexible?: boolean;
  editingHourType: string | null;
  editingHourTypeValue: string;
  editingHourTypeError?: string | null;
  editingHourTypeSuggestion?: string | null;
  onEditHourType: (hourType: string) => void;
  onSaveHourType: () => void;
  onRemoveHourType: (hourType: string) => void;
  onApplySuggestion?: () => void;
  onCancelEditHourType: () => void;
  setEditingHourTypeValue: (value: string) => void;
  editingPosition: string | null;
  editingPositionValue: string;
  onEditPosition: (position: string) => void;
  onSavePosition: () => void;
  onRemovePosition: (position: string) => void;
  onCopyRates: (rates: PayrateStructure) => void;
  setEditingPositionValue: (value: string) => void;
  setEditingPosition: (position: string | null) => void;
  onRateChange: (position: string, dayType: DayType, hourType: string, value: string) => void;
}

export function PayrateTable({
  rates, originalRates, positions, hourTypes, readOnly = false, validation, isFlexible = false,
  editingHourType, editingHourTypeValue, editingHourTypeError, editingHourTypeSuggestion,
  onEditHourType, onSaveHourType, onRemoveHourType, onApplySuggestion, onCancelEditHourType, setEditingHourTypeValue,
  editingPosition, editingPositionValue, onEditPosition, onSavePosition, onRemovePosition, onCopyRates,
  setEditingPositionValue, setEditingPosition, onRateChange,
}: PayrateTableProps) {
  return (
    <div className="rounded-xl border border-border overflow-hidden">
      <div className="overflow-x-auto">
        <Table className="w-full border-collapse [&_td]:p-0 [&_th]:p-0">
          <ShadcnTableHeader>
            <TableHeader
              hourTypes={hourTypes}
              readOnly={readOnly}
              isFlexible={isFlexible}
              editingHourType={editingHourType}
              editingHourTypeValue={editingHourTypeValue}
              editingHourTypeError={editingHourTypeError}
              editingHourTypeSuggestion={editingHourTypeSuggestion}
              onEditHourType={onEditHourType}
              onSaveHourType={onSaveHourType}
              onRemoveHourType={onRemoveHourType}
              onApplySuggestion={onApplySuggestion}
              onCancelEditHourType={onCancelEditHourType}
              setEditingHourTypeValue={setEditingHourTypeValue}
            />
          </ShadcnTableHeader>
          <TableBody
            positions={positions}
            hourTypes={hourTypes}
            rates={rates}
            originalRates={originalRates}
            readOnly={readOnly}
            validation={validation}
            isFlexible={isFlexible}
            editingPosition={editingPosition}
            editingPositionValue={editingPositionValue}
            onEditPosition={onEditPosition}
            onSavePosition={onSavePosition}
            onRemovePosition={onRemovePosition}
            onCopyRates={onCopyRates}
            onRateChange={onRateChange}
            setEditingPositionValue={setEditingPositionValue}
            setEditingPosition={setEditingPosition}
          />
        </Table>
      </div>
    </div>
  );
}
