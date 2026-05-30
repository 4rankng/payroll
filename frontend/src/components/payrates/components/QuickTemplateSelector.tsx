import { useState } from 'react';
import {
  Clock,
  Tag
} from 'lucide-react';
import {
  POSITION_RATE_TEMPLATES,
  PositionConfig,
  PayrateStructureType
} from '../types';
import { generateRateTemplate } from '../utils/templateGenerator';
import { PositionInput } from './PositionInput';
import { COMMON_HOUR_TYPES } from '@/types/api/payrate.types';

interface QuickTemplateSelectorProps {
  onSelectTemplate: (config: PositionConfig) => void;
}

export function QuickTemplateSelector({ 
  onSelectTemplate
}: QuickTemplateSelectorProps) {
  const [positions, setPositions] = useState<string[]>([]);

  const handleTemplateSelect = (templateType: PayrateStructureType) => {
    // Generate configuration with selected positions
    const rates = generateRateTemplate(templateType, positions);
    
    const config: PositionConfig = {
      positions,
      rateType: templateType,
      rates
    };
    
    onSelectTemplate(config);
  };

  return (
    <div className="space-y-6 w-full">
      {/* Header */}
      <div>
        <h3 className="typography-headline-medium">Tạo cấu hình lương</h3>
      </div>

      {/* Position Selection */}
      <PositionInput
        positions={positions}
        onChange={setPositions}
      />

      {/* Template Cards - Same Row */}
      <div>
        <h4 className="font-medium mb-4">Mẫu cấu hình có sẵn</h4>
        <div className="grid grid-cols-1 gap-0 border rounded-xl overflow-hidden">
        {POSITION_RATE_TEMPLATES.map((template, index) => (
          <div 
            key={template.type}
            className={`group p-4 cursor-pointer transition-all hover:bg-muted/50 bg-card ${
              index < POSITION_RATE_TEMPLATES.length - 1 ? 'border-b' : ''
            }`}
            onClick={() => handleTemplateSelect(template.type)}
          >
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 bg-primary/10 rounded-xl flex items-center justify-center group-hover:bg-primary/20 transition-colors">
                {template.type === 'hourly' ? (
                  <Clock className="w-6 h-6 text-primary" />
                ) : (
                  <Tag className="w-6 h-6 text-primary" />
                )}
              </div>
              
              <div className="flex-1">
                <h4 className="typography-title-large">{template.name}</h4>
                <p className="text-muted-foreground mt-1">
                  {template.description}
                </p>
                <p className="typography-body-small text-muted-foreground mt-2">
                  Ví dụ: {template.type === 'hourly'
                    ? COMMON_HOUR_TYPES.TIME_RANGES.slice(0, 3).join(', ')
                    : COMMON_HOUR_TYPES.CATEGORIES.join(', ')
                  }
                </p>
              </div>
            </div>
          </div>
        ))}
        </div>
      </div>

    </div>
  );
}