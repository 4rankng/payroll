import { 
  RateCategory, 
  PayrateConfiguration,
  PayrateStructureType, 
  POSITION_RATE_TEMPLATES,
  PositionConfig 
} from '../types';

export function generateRateTemplate(
  rateType: PayrateStructureType,
  positions: string[]
): PayrateConfiguration {
  const template = POSITION_RATE_TEMPLATES.find(t => t.type === rateType);
  if (!template) {
    throw new Error(`Template not found for rate type: ${rateType}`);
  }

  const baseStructure = template.baseStructure;

  // If no positions provided, use "tất cả" as default
  if (positions.length === 0) {
    return {
      "tất cả": baseStructure
    };
  }

  // Generate structure for each position
  const result: PayrateConfiguration = {};
  positions.forEach(position => {
    result[position] = JSON.parse(JSON.stringify(baseStructure));
  });

  return result;
}

export function addPositionToTemplate(
  currentRates: PayrateConfiguration,
  newPosition: string,
  rateType: PayrateStructureType
): PayrateConfiguration {
  const template = POSITION_RATE_TEMPLATES.find(t => t.type === rateType);
  if (!template) {
    throw new Error(`Template not found for rate type: ${rateType}`);
  }

  return {
    ...currentRates,
    [newPosition]: JSON.parse(JSON.stringify(template.baseStructure))
  };
}

export function removePositionFromTemplate(
  currentRates: PayrateConfiguration,
  positionToRemove: string
): PayrateConfiguration {
  const { [positionToRemove]: removed, ...remaining } = currentRates;
  
  // If no positions left, add "tất cả" as default
  if (Object.keys(remaining).length === 0) {
    return { "tất cả": {} };
  }
  
  return remaining;
}

export function copyRatesBetweenPositions(
  currentRates: PayrateConfiguration,
  fromPosition: string,
  toPosition: string
): PayrateConfiguration {
  if (!currentRates[fromPosition]) {
    throw new Error(`Source position "${fromPosition}" not found`);
  }
  
  if (!currentRates[toPosition]) {
    throw new Error(`Target position "${toPosition}" not found`);
  }

  return {
    ...currentRates,
    [toPosition]: JSON.parse(JSON.stringify(currentRates[fromPosition]))
  };
}

export function getPositionsFromRateCategory(rates: PayrateConfiguration): string[] {
  return Object.keys(rates);
}

export function detectRateType(rates: PayrateConfiguration): PayrateStructureType {
  // Get the first position's structure to analyze
  const firstPosition = Object.keys(rates)[0];
  if (!firstPosition || !rates[firstPosition]) {
    return 'category'; // Default fallback
  }

  const firstDayType = Object.keys(rates[firstPosition])[0];
  if (!firstDayType) {
    return 'category'; // Default fallback
  }

  const dayStructure = rates[firstPosition][firstDayType as keyof typeof rates[string]];
  if (typeof dayStructure !== 'object') {
    return 'category'; // Invalid structure
  }

  const firstKey = Object.keys(dayStructure)[0];
  
  // Check if the key looks like a time range (contains ":")
  if (firstKey && firstKey.includes(':')) {
    return 'hourly';
  }
  
  return 'category';
}

export function validatePositionConfig(config: PositionConfig): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // Check positions
  if (config.positions.length === 0) {
    errors.push('Phải có ít nhất một vị trí');
  }

  // Check for duplicate positions
  const uniquePositions = new Set(config.positions);
  if (uniquePositions.size !== config.positions.length) {
    errors.push('Không được có vị trí trùng lặp');
  }

  // Check for empty position names
  config.positions.forEach(position => {
    if (!position.trim()) {
      errors.push('Tên vị trí không được để trống');
    }
  });

  // Validate rate structure exists
  if (!config.rates || Object.keys(config.rates).length === 0) {
    errors.push('Cấu trúc lương không được để trống');
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

export function createPositionConfigFromRates(
  rates: PayrateConfiguration,
  rateType?: PayrateStructureType
): PositionConfig {
  return {
    positions: getPositionsFromRateCategory(rates),
    rateType: rateType || detectRateType(rates),
    rates
  };
}