import type { PayrateStructure } from '../../types';
import { getPositionsFromRates, getAllHourTypes } from '../../types';

interface SummaryInfoProps {
  rates: PayrateStructure;
  isFlexible?: boolean;
}

export function SummaryInfo({ rates, isFlexible = false }: SummaryInfoProps) {
  const positions = getPositionsFromRates(rates);
  const hourTypes = getAllHourTypes(rates);
  const totalCells = Object.values(rates).reduce((total, dayConfig) =>
    total + Object.values(dayConfig).reduce((dayTotal, hourConfig) =>
      dayTotal + Object.values(hourConfig).filter((r: number) => r > 0).length, 0), 0);

  return (
    <div className="flex flex-wrap items-center gap-3 border-t border-border/40 px-4 pt-3 text-xs text-muted-foreground sm:px-0">
      <span>{positions.length} vị trí</span>
      <span className="text-border">·</span>
      <span>{hourTypes.length} {isFlexible ? 'ca làm việc' : 'khung giờ'}</span>
      <span className="text-border">·</span>
      <span>{totalCells} {isFlexible ? 'ca có mức lương' : 'ô có giá trị'}</span>
    </div>
  );
}
