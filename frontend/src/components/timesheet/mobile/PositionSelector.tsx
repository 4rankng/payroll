import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { GraduationCap } from 'lucide-react';
import { PayrateStructure, DEFAULT_POSITIONS } from '@/types/api/payrate.types';

interface PositionSelectorProps {
  position: string;
  onPositionChange: (position: string) => void;
  payrateConfig?: PayrateStructure;
}

export function PositionSelector({
  position,
  onPositionChange,
  payrateConfig
}: PositionSelectorProps) {
  // Get available positions from payrate config, with fallback to default positions
  const availablePositions = payrateConfig ? Object.keys(payrateConfig) : [...DEFAULT_POSITIONS];

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <GraduationCap className="h-4 w-4 text-muted-foreground" />
        <Label className="typography-body-medium">Vị trí</Label>
      </div>

      <Select value={position} onValueChange={onPositionChange}>
        <SelectTrigger className="h-11 border-2 border-border/50 hover:border-border transition-colors">
          <SelectValue>{position}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          {availablePositions.map((pos) => (
            <SelectItem key={pos} value={pos}>
              {pos}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {!payrateConfig && (
        <div className="bg-muted/50 rounded-xl p-3">
          <p className="typography-body-small text-muted-foreground">
            <strong>Lưu ý:</strong> Hiển thị vị trí mặc định. Chọn dự án để xem vị trí cụ thể.
          </p>
        </div>
      )}
    </div>
  );
}