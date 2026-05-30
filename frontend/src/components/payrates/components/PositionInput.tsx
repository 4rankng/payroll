import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import {
  Plus,
  X,
  Users,
  AlertTriangle
} from 'lucide-react';
import { COMMON_POSITIONS } from '../types';

interface PositionInputProps {
  positions: string[];
  onChange: (positions: string[]) => void;
  disabled?: boolean;
}

export function PositionInput({ positions, onChange, disabled }: PositionInputProps) {
  const [inputValue, setInputValue] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleAddPosition = () => {
    const trimmedValue = inputValue.trim();

    if (!trimmedValue) {
      setError('Tên vị trí không được để trống');
      return;
    }

    if (positions.includes(trimmedValue)) {
      setError('Vị trí đã tồn tại');
      return;
    }

    onChange([...positions, trimmedValue]);
    setInputValue('');
    setError(null);
  };

  const handleRemovePosition = (positionToRemove: string) => {
    onChange(positions.filter(pos => pos !== positionToRemove));
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddPosition();
    }
  };

  const handleQuickAdd = (position: string) => {
    if (positions.includes(position)) return;
    onChange([...positions, position]);
  };

  return (
    <div className="space-y-4">
      <div>
        <Label className="typography-body-large">Vị trí công việc</Label>
        <p className="typography-body-medium text-muted-foreground mt-1">
          Thêm các vị trí để cấu hình lương riêng biệt. Để trống để sử dụng "tất cả"
        </p>
      </div>

      {/* Quick add common positions */}
      <div className="p-4 bg-muted/50 rounded-xl">
        <div className="typography-body-medium mb-3 flex items-center gap-2">
          <Users className="w-4 h-4" />
          Vị trí phổ biến
        </div>
        <div className="flex flex-wrap gap-2">
          {COMMON_POSITIONS.map((position) => {
              const isSelected = positions.includes(position);
              return (
                <Button
                  key={position}
                  variant={isSelected ? "default" : "outline"}
                  size="sm"
                  onClick={() => {
                    if (isSelected) {
                      onChange(positions.filter(pos => pos !== position));
                    } else {
                      handleQuickAdd(position);
                    }
                  }}
                  disabled={disabled}
                  className="typography-body-small"
                >
                  {isSelected ? (
                    <>
                      <X className="w-3 h-3 mr-1" />
                      {position}
                    </>
                  ) : (
                    <>
                      <Plus className="w-3 h-3 mr-1" />
                      {position}
                    </>
                  )}
                </Button>
              );
            })}
        </div>
      </div>

      {/* Manual input */}
      <div className="flex gap-2">
        <div className="flex-1">
          <Input
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              if (error) setError(null);
            }}
            onKeyPress={handleKeyPress}
            placeholder="Nhập tên vị trí..."
            disabled={disabled}
            className={error ? 'border-red-500' : ''}
          />
          {error && (
            <div className="flex items-center gap-1 mt-1 typography-body-medium text-red-600">
              <AlertTriangle className="w-3 h-3" />
              {error}
            </div>
          )}
        </div>
        <Button
          onClick={handleAddPosition}
          disabled={disabled || !inputValue.trim()}
          size="default"
        >
          <Plus className="w-4 h-4 mr-1" />
          Thêm
        </Button>
      </div>
    </div>
  );
}
