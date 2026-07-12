import { cn } from '@/lib/utils';

// Bitmask: bit0=Sun, bit1=Mon, bit2=Tue, bit3=Wed, bit4=Thu, bit5=Fri, bit6=Sat
const DAYS = [
  { bit: 0, label: 'CN' },
  { bit: 1, label: 'T2' },
  { bit: 2, label: 'T3' },
  { bit: 3, label: 'T4' },
  { bit: 4, label: 'T5' },
  { bit: 5, label: 'T6' },
  { bit: 6, label: 'T7' },
];

export function isOffDay(offDays: number, dayOfWeek: number): boolean {
  return (offDays & (1 << dayOfWeek)) !== 0;
}

interface OffDaysPickerProps {
  value: number; // bitmask
  onChange: (value: number) => void;
  disabled?: boolean;
}

export function OffDaysPicker({ value, onChange, disabled }: OffDaysPickerProps) {
  const toggle = (bit: number) => {
    if (disabled) return;
    onChange(value ^ (1 << bit));
  };

  return (
    <div className="space-y-1.5">
      <div className="flex flex-wrap gap-1.5">
        {DAYS.map(({ bit, label }) => {
          const active = (value & (1 << bit)) !== 0;
          return (
            <button
              key={bit}
              type="button"
              onClick={() => toggle(bit)}
              disabled={disabled}
              className={cn(
                'w-8 h-8 sm:w-9 sm:h-9 rounded-xl text-xs font-semibold border transition-colors shrink-0',
                active
                  ? 'bg-red-100 border-red-400 text-red-600'
                  : 'bg-emerald-50 border-emerald-300 text-emerald-700 hover:bg-emerald-100',
                disabled && 'opacity-50 cursor-not-allowed'
              )}
            >
              {label}
            </button>
          );
        })}
      </div>
      <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
        <span className="flex items-center gap-1">
          <span className="inline-block w-2.5 h-2.5 rounded-sm bg-emerald-100 border border-emerald-300" />
          Ngày thường
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block w-2.5 h-2.5 rounded-sm bg-red-100 border border-red-400" />
          Ngày nghỉ
        </span>
      </div>
    </div>
  );
}
