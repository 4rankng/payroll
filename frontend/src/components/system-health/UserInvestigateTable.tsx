import { useMemo, useState } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { useErrorsByUser } from "@/hooks/api/useSystemHealth";
import { useUsersByIds } from "@/hooks/api/useUsers";
import { EndpointLabel } from "./EndpointLabel";
import { AllClear } from "./AllClear";
import { roleLabel, relativeTime } from "./utils";

interface UserRow {
  userId: number;
  totalCount: number;
  codes: Set<number>;
  breakdown: Array<{
    endpoint: string;
    statusCode: number;
    count: number;
    lastSeen: string;
  }>;
}

interface Props {
  days?: number;
  limit?: number;
}

export function UserInvestigateTable({ days = 14, limit = 10 }: Props) {
  const { data: errorsByUser, isLoading } = useErrorsByUser(days);
  const [expandedUser, setExpandedUser] = useState<number | null>(null);

  const userIds = useMemo(
    () => [...new Set(errorsByUser?.map((d) => d.user_id).filter((id): id is number => id != null && id !== 0) ?? [])],
    [errorsByUser]
  );
  const { data: userRecord } = useUsersByIds(userIds, { enabled: userIds.length > 0 });

  const rows = useMemo((): UserRow[] => {
    if (!errorsByUser) return [];
    const map = new Map<number, UserRow>();
    for (const row of errorsByUser) {
      if (row.user_id == null) continue;
      const existing = map.get(row.user_id) ?? {
        userId: row.user_id,
        totalCount: 0,
        codes: new Set<number>(),
        breakdown: [],
      };
      existing.totalCount += row.count;
      existing.codes.add(row.status_code);
      existing.breakdown.push({
        endpoint: row.endpoint,
        statusCode: row.status_code,
        count: row.count,
        lastSeen: row.last_seen,
      });
      map.set(row.user_id, existing);
    }
    return [...map.values()]
      .sort((a, b) => b.totalCount - a.totalCount)
      .slice(0, limit);
  }, [errorsByUser, limit]);

  return (
    <div>
      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <AllClear text="Không có người dùng bất thường" />
      ) : (
        <div className="divide-y border bg-card rounded-xl overflow-hidden">
          {rows.map((row) => {
              const user = userRecord?.[row.userId];
              const isExpanded = expandedUser === row.userId;
              const isCritical = row.totalCount > 50;
              const isWarn = row.totalCount > 10;

              return (
                <div key={row.userId}>
                  {/* Summary row */}
                  <button
                    onClick={() => setExpandedUser(isExpanded ? null : row.userId)}
                    className={cn(
                      "w-full flex items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/30",
                      isCritical ? "bg-red-50/50"
                        : isWarn ? "bg-amber-50/30"
                        : ""
                    )}
                  >
                    {/* Chevron */}
                    <span className="text-muted-foreground shrink-0">
                      {isExpanded
                        ? <ChevronDown className="h-4 w-4" />
                        : <ChevronRight className="h-4 w-4" />}
                    </span>

                    {/* Name + meta */}
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-sm truncate">
                        {user?.fullname ?? `#${row.userId}`}
                      </div>
                      {user && (
                        <div className="text-xs text-muted-foreground truncate">
                          @{user.username} · {roleLabel(user.role)}
                        </div>
                      )}
                    </div>

                    {/* Right side: labeled total + per-code breakdown */}
                    <div className="flex flex-col items-end gap-1.5 shrink-0">
                      {/* Total error count — clearly labeled */}
                      <div className="flex items-center gap-1">
                        <span className="text-xs text-muted-foreground">Tổng lỗi:</span>
                        <span className={cn(
                          "text-base font-bold tabular-nums",
                          isCritical ? "text-red-700" : isWarn ? "text-amber-700" : "text-foreground"
                        )}>
                          {row.totalCount.toLocaleString()}
                        </span>
                      </div>
                      {/* Per HTTP-code pills: code + count together */}
                      <div className="flex flex-wrap justify-end gap-1">
                        {(() => {
                          const codeMap = new Map<number, number>();
                          for (const b of row.breakdown) {
                            codeMap.set(b.statusCode, (codeMap.get(b.statusCode) ?? 0) + b.count);
                          }
                          return [...codeMap.entries()]
                            .sort((a, b) => b[1] - a[1])
                            .slice(0, 3)
                            .map(([code, count]) => (
                              <span
                                key={code}
                                className={cn(
                                  "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium border",
                                  code >= 500
                                    ? "bg-red-100 border-red-200 text-red-700"
                                    : "bg-orange-50 border-orange-200 text-orange-700"
                                )}
                              >
                                <span className="font-mono font-bold">{code}</span>
                                <span className="opacity-50">×</span>
                                <span>{count}</span>
                              </span>
                            ));
                        })()}
                      </div>
                    </div>
                  </button>

                  {/* Expanded: per-endpoint breakdown */}
                  {isExpanded && (
                    <div className="bg-muted/20 border-t border-dashed divide-y divide-dashed">
                      {row.breakdown
                        .sort((a, b) => b.count - a.count)
                        .map((ep, i) => (
                          <div
                            key={`${ep.endpoint}-${ep.statusCode}-${i}`}
                            className="flex items-start gap-3 px-4 py-2.5 pl-11"
                          >
                            <div className="flex-1 min-w-0">
                              <div className="truncate">
                                <EndpointLabel endpoint={ep.endpoint} className="text-xs" />
                              </div>
                              <div className="text-xs text-muted-foreground mt-0.5">
                                {relativeTime(ep.lastSeen)}
                              </div>
                            </div>
                            <div className="flex items-center gap-2 shrink-0">
                              <span
                                className={cn(
                                  "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium border",
                                  ep.statusCode >= 500
                                    ? "bg-red-100 border-red-200 text-red-700"
                                    : "bg-orange-50 border-orange-200 text-orange-700"
                                )}
                              >
                                <span className="font-mono font-bold">{ep.statusCode}</span>
                                <span className="opacity-50">×</span>
                                <span>{ep.count.toLocaleString()}</span>
                              </span>
                            </div>
                          </div>
                        ))}
                    </div>
                  )}
                </div>
              );
            })}
        </div>
      )}
    </div>
  );
}
