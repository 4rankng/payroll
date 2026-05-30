import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Callout } from '@/components/ui/callout';
import { EmptyState } from '@/components/ui/loading-states';
import type { PayrateStructure, ValidationResult } from '../types';
import { createDefaultPayrateStructure, createFlexiblePayrateStructure } from '../types';

// Import hooks
import { usePayrateEditor } from '../hooks/usePayrateEditor';
import { usePositionManager } from '../hooks/usePositionManager';
import { useHourTypeManager } from '../hooks/useHourTypeManager';

// Import components
import { ValidationAlert } from './matrix-editor/ValidationAlert';
import { PayrateTable } from './matrix-editor/PayrateTable';
import { QuickActions } from './matrix-editor/QuickActions';
import { SummaryInfo } from './matrix-editor/SummaryInfo';

interface PayrateMatrixEditorProps {
  rates: PayrateStructure;
  originalRates?: PayrateStructure;
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
  validation?: ValidationResult;
  isFlexible?: boolean;
}

export function PayrateMatrixEditor({
  rates,
  originalRates,
  onChange,
  readOnly = false,
  validation,
  isFlexible = false,
}: PayrateMatrixEditorProps) {
  // Use custom hooks for state management
  const { displayRates, handleRateChange, isEmpty } = usePayrateEditor(rates, onChange, readOnly);

  const positionManager = usePositionManager(displayRates, onChange, readOnly);
  const hourTypeManager = useHourTypeManager(displayRates, onChange, readOnly);

  // Handle empty state
  if (isEmpty) {
    if (readOnly) {
      return (
        <div className="text-center py-8">
          <p className="typography-body-large text-muted-foreground mb-4">
            Chưa có dữ liệu cấu hình mức lương
          </p>
          <p className="typography-body-medium text-muted-foreground">
            Vui lòng tạo cấu hình mức lương để hiển thị thông tin tại đây
          </p>
        </div>
      );
    }

    return (
      <Card>
        <CardContent>
          <div className="text-center py-8">
            <p className="typography-body-large text-muted-foreground mb-4">Chưa có cấu hình mức lương nào</p>
            <Button onClick={() => onChange(isFlexible ? createFlexiblePayrateStructure() : createDefaultPayrateStructure())} variant="default">
              Tạo cấu hình mặc định
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  // For flexible projects with no shifts yet — show a welcoming empty state
  const noShiftsYet = isFlexible && hourTypeManager.hourTypes.length === 0;

  return (
    <div className="space-y-6 w-full">
      {/* Validation Messages */}
      <ValidationAlert validation={validation} />

      {/* Hour type edit error — shown above table, not inside it */}
      {hourTypeManager.editingHourTypeError && (
        <Callout variant="error">
          <span className="font-medium">{hourTypeManager.editingHourTypeError}</span>
          {hourTypeManager.editingHourTypeSuggestion && (
            <button
              type="button"
              onClick={hourTypeManager.handleApplySuggestion}
              className="ml-2 text-xs font-semibold text-red-700 underline underline-offset-2 hover:text-red-900 transition-colors"
            >
              Dùng: {hourTypeManager.editingHourTypeSuggestion}
            </button>
          )}
        </Callout>
      )}

      {/* Matrix Editor Table */}
      <div className="space-y-4">
        {noShiftsYet ? (
          <div className="rounded-xl border border-dashed border-border bg-muted/10 px-6">
            <EmptyState
              title="Chưa có ca làm việc"
              description="Thêm ca từ bảng bên dưới để bắt đầu nhập mức lương"
            />
          </div>
        ) : (
        <PayrateTable
          rates={displayRates}
          originalRates={originalRates}
          positions={positionManager.positions}
          hourTypes={hourTypeManager.hourTypes}
          readOnly={readOnly}
          validation={validation}
          isFlexible={isFlexible}

          // Hour type management props
          editingHourType={hourTypeManager.editingHourType}
          editingHourTypeValue={hourTypeManager.editingHourTypeValue}
          editingHourTypeError={hourTypeManager.editingHourTypeError}
          editingHourTypeSuggestion={hourTypeManager.editingHourTypeSuggestion}
          onEditHourType={hourTypeManager.handleEditHourType}
          onSaveHourType={hourTypeManager.handleSaveHourType}
          onRemoveHourType={hourTypeManager.handleRemoveHourType}
          onApplySuggestion={hourTypeManager.handleApplySuggestion}
          onCancelEditHourType={hourTypeManager.handleCancelEditHourType}
          setEditingHourTypeValue={hourTypeManager.setEditingHourTypeValue}

          // Position management props
          editingPosition={positionManager.editingPosition}
          editingPositionValue={positionManager.editingPositionValue}
          onEditPosition={positionManager.handleEditPosition}
          onSavePosition={positionManager.handleSavePosition}
          onRemovePosition={positionManager.handleRemovePosition}
          onCopyRates={onChange}
          setEditingPositionValue={positionManager.setEditingPositionValue}
          setEditingPosition={positionManager.setEditingPosition}

          // Rate management props
          onRateChange={handleRateChange}
        />
        )}
      </div>

      {/* Quick Add Sections */}
      <QuickActions
        rates={displayRates}
        onChange={onChange}
        readOnly={readOnly}
        isFlexible={isFlexible}
        hourTypes={hourTypeManager.hourTypes}

        // Position management props
        newPosition={positionManager.newPosition}
        showAddPosition={positionManager.showAddPosition}
        setNewPosition={positionManager.setNewPosition}
        setShowAddPosition={positionManager.setShowAddPosition}
        handleAddPosition={positionManager.handleAddPosition}

        // Hour type management props
        newHourType={hourTypeManager.newHourType}
        showAddHourType={hourTypeManager.showAddHourType}
        setNewHourType={hourTypeManager.setNewHourType}
        setShowAddHourType={hourTypeManager.setShowAddHourType}
        handleAddHourType={hourTypeManager.handleAddHourType}
      />

      {/* Summary Info */}
      <SummaryInfo rates={displayRates} isFlexible={isFlexible} />
    </div>
  );
}
