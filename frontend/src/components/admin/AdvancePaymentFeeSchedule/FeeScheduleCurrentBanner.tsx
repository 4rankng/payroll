import { useMemo, useState } from "react";
import {
  CalendarClock,
  CheckCircle2,
  CircleHelp,
  Pencil,
  Sparkles,
  Trash2,
} from "lucide-react";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import {
  describeEntryFragments,
  formatActiveRange,
  formatDaysFromNow,
  formatVietnameseDate,
  isLikelyBootstrap,
  type RuleFragment,
} from "./rule-formatters";
import { daysFromTodayISO } from "./form-helpers";
import { isEntryEditable } from "./form-helpers";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  active: FeeScheduleEntry | null;
  upcoming: FeeScheduleEntry[];
  onEdit: (entry: FeeScheduleEntry) => void;
  onDelete: (entry: FeeScheduleEntry) => void;
}

const RuleText = ({ fragments }: { fragments: RuleFragment[] }) => (
  <p className="text-base sm:text-lg leading-snug text-foreground">
    {fragments.map((f, i) =>
      f.type === "emphasis" ? (
        <strong key={i} className="font-semibold">
          {f.value}
        </strong>
      ) : (
        <span key={i}>{f.value}</span>
      ),
    )}
  </p>
);

export const FeeScheduleCurrentBanner = ({
  active,
  upcoming,
  onEdit,
  onDelete,
}: Props) => {
  const [showAllUpcoming, setShowAllUpcoming] = useState(false);

  const sortedUpcoming = useMemo(
    () =>
      [...upcoming].sort((a, b) =>
        a.effectiveDate < b.effectiveDate ? -1 : 1,
      ),
    [upcoming],
  );

  const nextEntry = sortedUpcoming[0] ?? null;
  const nextAfterNext = sortedUpcoming.slice(1);

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        <ActiveCard active={active} nextEntry={nextEntry} />
        {nextEntry ? (
          <UpcomingCard
            entry={nextEntry}
            onEdit={onEdit}
            onDelete={onDelete}
            additionalCount={nextAfterNext.length}
            showAdditional={showAllUpcoming}
            onToggleAdditional={() => setShowAllUpcoming((v) => !v)}
            additional={nextAfterNext}
          />
        ) : (
          <NoUpcomingCard />
        )}
      </div>
    </div>
  );
};

const ActiveCard = ({
  active,
  nextEntry,
}: {
  active: FeeScheduleEntry | null;
  nextEntry: FeeScheduleEntry | null;
}) => {
  if (!active) {
    return (
      <div className="rounded-xl border border-dashed bg-muted/30 p-5 flex items-center justify-center text-center">
        <p className="text-sm text-muted-foreground">
          Chưa có cấu hình nào đang hoạt động
        </p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border-2 border-primary/30 bg-primary/[0.04] p-4 sm:p-5 relative overflow-hidden">
      <div className="absolute inset-x-0 top-0 h-1 bg-primary/60" />
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/15">
            <CheckCircle2 className="h-4 w-4 text-primary" />
          </div>
          <span className="text-xs font-semibold uppercase tracking-wider text-primary">
            Đang áp dụng
          </span>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          {isLikelyBootstrap(active) && (
            <Badge variant="outline" className="text-[10px] font-normal">
              Cấu hình mặc định
            </Badge>
          )}
          <HelpPopover />
        </div>
      </div>

      <RuleText fragments={describeEntryFragments(active)} />

      <p className="text-xs text-muted-foreground mt-2">
        {formatActiveRange(active.effectiveDate, nextEntry)}
      </p>
    </div>
  );
};

const UpcomingCard = ({
  entry,
  onEdit,
  onDelete,
  additionalCount,
  showAdditional,
  onToggleAdditional,
  additional,
}: {
  entry: FeeScheduleEntry;
  onEdit: (entry: FeeScheduleEntry) => void;
  onDelete: (entry: FeeScheduleEntry) => void;
  additionalCount: number;
  showAdditional: boolean;
  onToggleAdditional: () => void;
  additional: FeeScheduleEntry[];
}) => {
  const days = daysFromTodayISO(entry.effectiveDate);
  const editable = isEntryEditable(entry);

  return (
    <div className="rounded-xl border-2 border-amber-400/40 bg-amber-50/50 dark:bg-amber-500/[0.06] p-4 sm:p-5 relative overflow-hidden">
      <div className="absolute inset-x-0 top-0 h-1 bg-amber-400/70" />
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/15">
            <CalendarClock className="h-4 w-4 text-amber-600" />
          </div>
          <span className="text-xs font-semibold uppercase tracking-wider text-amber-700 dark:text-amber-400">
            Đang chờ
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            aria-label="Sửa cấu hình sắp có hiệu lực"
            disabled={!editable}
            onClick={() => onEdit(entry)}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            aria-label="Xóa cấu hình sắp có hiệu lực"
            disabled={!editable}
            onClick={() => onDelete(entry)}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <RuleText fragments={describeEntryFragments(entry)} />

      <p className="text-xs text-muted-foreground mt-2">
        Sẽ thay thế cấu hình hiện tại từ{" "}
        <strong className="font-medium text-foreground/80">
          {formatVietnameseDate(entry.effectiveDate)}
        </strong>{" "}
        ({formatDaysFromNow(days)})
      </p>

      {additionalCount > 0 && (
        <div className="mt-3 pt-3 border-t border-amber-400/20">
          <button
            type="button"
            onClick={onToggleAdditional}
            className="text-xs text-amber-700 dark:text-amber-400 hover:underline"
          >
            {showAdditional
              ? "Ẩn các cấu hình khác"
              : `Xem ${additionalCount} cấu hình khác sắp có hiệu lực`}
          </button>
          {showAdditional && (
            <ul className="mt-2 space-y-1.5">
              {additional.map((e) => (
                <li
                  key={e.id}
                  className="text-xs text-muted-foreground flex items-baseline justify-between gap-2"
                >
                  <span className="truncate">{e.summary}</span>
                  <span className="shrink-0">
                    {formatVietnameseDate(e.effectiveDate)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
};

const NoUpcomingCard = () => (
  <div
    className={cn(
      "rounded-xl border border-dashed bg-muted/20 p-4 sm:p-5",
      "flex flex-col justify-center text-center gap-2",
    )}
  >
    <Sparkles className="h-5 w-5 text-muted-foreground/60 mx-auto" />
    <p className="text-sm text-muted-foreground leading-relaxed max-w-sm mx-auto">
      Không có cấu hình sắp có hiệu lực. Thêm cấu hình mới để thay đổi phí từ
      một ngày trong tương lai.
    </p>
  </div>
);

const HelpPopover = () => (
  <Popover>
    <PopoverTrigger asChild>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-7 w-7"
        aria-label="Hướng dẫn cấu hình phí"
      >
        <CircleHelp className="h-4 w-4 text-muted-foreground" />
      </Button>
    </PopoverTrigger>
    <PopoverContent className="w-80" align="end">
      <div className="space-y-2 text-xs leading-relaxed">
        <p className="font-semibold text-sm">Cấu hình phí ứng lương</p>
        <p>
          Mỗi cấu hình áp dụng từ ngày hiệu lực cho đến khi có cấu hình mới
          thay thế. Cấu hình quá khứ và đang hoạt động không thể sửa hay xóa
          để giữ tính minh bạch của lịch sử giao dịch.
        </p>
        <p>
          Để thay đổi phí, hãy thêm cấu hình mới với ngày hiệu lực trong tương
          lai. Cấu hình đó sẽ tự động áp dụng khi đến ngày.
        </p>
      </div>
    </PopoverContent>
  </Popover>
);
