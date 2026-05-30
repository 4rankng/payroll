import { useMemo, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Activity, ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "@/lib/utils";
import { useRecentErrors } from "@/hooks/api/useSystemHealth";
import { useUsersByIds } from "@/hooks/api/useUsers";
import { EndpointLabel } from "./EndpointLabel";
import { AllClear } from "./AllClear";
import { roleLabel, relativeTime } from "./utils";

interface Props {
  days?: number;
  defaultVisible?: number;
}

export function RecentErrorsTable({ days = 1, defaultVisible = 10 }: Props) {
  const [open, setOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const { data, isLoading } = useRecentErrors(days);

  const userIds = useMemo(
    () => [...new Set(data?.map((d) => d.user_id).filter((id): id is number => id != null && id !== 0) ?? [])],
    [data]
  );
  const { data: userRecord } = useUsersByIds(userIds, { enabled: userIds.length > 0 });

  const visible = expanded ? data : data?.slice(0, defaultVisible);

  return (
    <Card>
      <button onClick={() => setOpen((o) => !o)} className="w-full text-left hover:bg-muted/50 transition-colors">
        <CardHeader className="pb-3 border-b">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Activity className="h-4 w-4 text-red-500 shrink-0" />
              <CardTitle className="text-sm font-semibold">
                Lỗi Gần Đây ({days === 1 ? "24h" : `${days}d`})
              </CardTitle>
            </div>
            <div className="flex items-center gap-2">
              {!isLoading && data && data.length > 0 && (
                <Badge variant="destructive" className="text-xs">
                  {data.length} lỗi
                </Badge>
              )}
              <ChevronDown className={cn("h-4 w-4 transition-transform", open && "rotate-180")} />
            </div>
          </div>
        </CardHeader>
      </button>
      {open && (
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-4 space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : !data || data.length === 0 ? (
            <AllClear text="Không có lỗi trong 24h qua" />
          ) : (
            <>
              <div className="divide-y">
                {visible?.map((err, idx) => {
                  const user = err.user_id != null ? userRecord?.[err.user_id] : null;
                  return (
                    <div
                      key={`${err.endpoint}-${err.called_at}-${idx}`}
                      className={cn(
                        "flex items-start gap-3 px-4 py-3",
                        err.status_code >= 500 ? "bg-red-50/50" : ""
                      )}
                    >
                      {/* Left: endpoint + user */}
                      <div className="flex-1 min-w-0">
                        <div className="truncate">
                          <EndpointLabel endpoint={err.endpoint} />
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5 truncate">
                          {user ? (
                            <span>
                              <span className="font-medium text-foreground">{user.fullname}</span>
                              {" · "}
                              <span>{roleLabel(user.role)}</span>
                            </span>
                          ) : err.user_id != null ? (
                            `#${err.user_id}`
                          ) : (
                            "—"
                          )}
                        </div>
                      </div>

                      {/* Right: badge + time + duration */}
                      <div className="flex flex-col items-end gap-1 shrink-0">
                        <Badge
                          variant={err.status_code >= 500 ? "destructive" : "outline"}
                          className="text-[10px] font-mono px-1.5 py-0 h-5"
                        >
                          {err.status_code}
                        </Badge>
                        <span className="text-[10px] text-muted-foreground tabular-nums">
                          {relativeTime(err.called_at)}
                        </span>
                        <span className="text-[10px] text-muted-foreground tabular-nums">
                          {err.duration_ms.toLocaleString()}ms
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>
              {data.length > defaultVisible && (
                <button
                  onClick={() => setExpanded((e) => !e)}
                  className="w-full flex items-center justify-center gap-1.5 py-2.5 text-xs
                             text-muted-foreground hover:text-foreground border-t transition-colors
                             hover:bg-muted/30"
                >
                  {expanded ? (
                    <><ChevronUp className="h-3 w-3" /> Thu gọn</>
                  ) : (
                    <><ChevronDown className="h-3 w-3" /> Xem thêm {data.length - defaultVisible} lỗi</>
                  )}
                </button>
              )}
            </>
          )}
        </CardContent>
      )}
    </Card>
  );
}
