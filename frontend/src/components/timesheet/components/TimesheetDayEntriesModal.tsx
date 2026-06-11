import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { Calendar, Clock, Banknote } from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { formatCurrency } from '@/utils/formatters';
import { TimesheetEntryCard } from './TimesheetEntryCard';

interface TimesheetDayEntriesModalProps {
  isOpen: boolean;
  onClose: () => void;
  date: Date;
  entries: Timesheet[];
  onEntrySelect: (entry: Timesheet) => void;
}

export function TimesheetDayEntriesModal({
  isOpen,
  onClose,
  date,
  entries,
  onEntrySelect
}: TimesheetDayEntriesModalProps) {
  const handleEntryClick = (entry: Timesheet) => {
    onEntrySelect(entry);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[700px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Chấm công ngày {format(date, 'dd/MM/yyyy', { locale: vi })}
          </DialogTitle>
          <p className="typography-body-small text-muted-foreground">
            {format(date, 'EEEE', { locale: vi })} • {entries.length} bản ghi
          </p>
        </DialogHeader>

        {/* Timeline Layout */}
        <div className="space-y-3 max-h-[65vh] overflow-y-auto pr-2">
          {entries.map((entry, index) => (
            <div key={entry.id} className="relative">
              {/* Timeline connector line - except for last item */}
              {index < entries.length - 1 && (
                <div className="absolute left-[11px] top-[60px] bottom-[-12px] w-0.5 bg-border" />
              )}

              {/* Timeline dot */}
              <div className="absolute left-2 top-6 w-3 h-3 rounded-full bg-primary border-2 border-background shadow-sm z-10" />

              {/* Entry Card with offset for timeline */}
              <div className="ml-8">
                <TimesheetEntryCard
                  entry={entry}
                  variant="timeline"
                  onClick={() => handleEntryClick(entry)}
                />
              </div>
            </div>
          ))}
        </div>

        {/* Enhanced Summary footer */}
        <div className="border-t pt-4 mt-4 bg-muted/30 -mx-6 px-6 -mb-6 pb-6 rounded-b-lg">
          <div className="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
            <span className="typography-title-medium font-semibold">Tổng cộng ngày {format(date, 'dd/MM')}:</span>
            <div className="flex gap-4">
              <div className="flex items-center gap-2 bg-green-50 px-3 py-2 rounded-xl border border-green-200">
                <Clock className="h-5 w-5 text-green-600" />
                <div>
                  <p className="typography-body-small text-muted-foreground">Ca làm việc</p>
                  <Badge variant="secondary" className="typography-data font-bold text-green-700 bg-green-100 mt-0.5">
                    {entries.reduce((sum, e) => sum + e.hours_worked, 0)}h
                  </Badge>
                </div>
              </div>
              <div className="flex items-center gap-2 bg-primary/5 px-3 py-2 rounded-xl border border-primary/20">
                <Banknote className="h-5 w-5 text-primary" />
                <div>
                  <p className="typography-body-small text-muted-foreground">Thành tiền</p>
                  <p className="typography-currency font-bold tabular-nums mt-0.5">
                    {formatCurrency(entries.reduce((sum, e) => sum + e.amount, 0))}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
