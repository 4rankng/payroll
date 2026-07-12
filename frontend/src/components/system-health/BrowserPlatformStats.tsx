import React, { useState, useCallback } from "react";
import { Monitor, Users, Zap, Smartphone, Globe, ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "@/lib/utils";
import { useOSStats, useBrowserStats } from "@/hooks/api/useSystemHealth";
import { SectionLabel } from "./SectionLabel";
import { AllClear } from "./AllClear";
import { OSUsersSheet, BrowserUsersSheet } from "./GroupedUsersSheet";
import type { OSGroupStat, BrowserGroupStat, VersionStat } from "@/types/api/system-health.types";

// ─── Icons ────────────────────────────────────────────────────────────────────

function osIcon(family: string): string {
  const f = family.toLowerCase();
  if (f === "ios") return "📱";
  if (f === "android") return "🤖";
  if (f === "macos") return "🍎";
  if (f === "windows") return "🪟";
  if (f === "linux") return "🐧";
  return "💻";
}

function browserIcon(family: string): string {
  const f = family.toLowerCase();
  if (f === "safari") return "🧭";
  if (f === "chrome") return "🌐";
  if (f === "firefox") return "🦊";
  if (f === "edge") return "🔷";
  if (f === "opera") return "🔴";
  return "🌍";
}

const RANK_COLORS = [
  "bg-amber-400 text-amber-900",
  "bg-slate-300 text-slate-700",
  "bg-orange-300 text-orange-900",
];

// ─── Version breakdown row ────────────────────────────────────────────────────

function VersionRow({ v, maxUsers }: { v: VersionStat; maxUsers: number }) {
  const pct = Math.round((v.unique_users / maxUsers) * 100);
  return (
    <div className="flex items-center gap-2 text-xs">
      <div className="w-2 h-2 rounded-full bg-primary/30 shrink-0" />
      <span className="text-muted-foreground truncate flex-1 min-w-0">{v.version}</span>
      <div className="flex items-center gap-1 shrink-0">
        <div className="w-12 h-1 bg-muted rounded-full overflow-hidden">
          <div className="h-full bg-primary/50 rounded-full" style={{ width: `${pct}%` }} />
        </div>
        <span className="tabular-nums font-medium text-foreground w-6 text-right">{v.unique_users}</span>
      </div>
    </div>
  );
}

// ─── OS group card ────────────────────────────────────────────────────────────

const MAX_VERSIONS_COLLAPSED = 3;

function OSGroupCard({
  stat,
  rank,
  maxUsers,
  onClickUsers,
}: {
  stat: OSGroupStat;
  rank: number;
  maxUsers: number;
  onClickUsers: (family: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const barPct = Math.round((stat.unique_users / maxUsers) * 100);
  const rankClass = RANK_COLORS[rank] ?? "bg-muted text-muted-foreground";
  const isTop = rank === 0;
  const versions = stat.versions ?? [];
  const showToggle = versions.length > MAX_VERSIONS_COLLAPSED;
  const visibleVersions = expanded ? versions : versions.slice(0, MAX_VERSIONS_COLLAPSED);
  const versionMax = versions[0]?.unique_users ?? 1;

  return (
    <div
      className={cn(
        "relative rounded-xl border bg-card p-3 overflow-hidden flex flex-col gap-2",
        isTop && "ring-1 ring-primary/20",
      )}
    >
      {/* Progress bar background */}
      <div
        className="absolute inset-y-0 left-0 bg-primary/5 transition-all duration-500"
        style={{ width: `${barPct}%` }}
      />

      {/* Header row */}
      <div className="relative flex items-center gap-2.5">
        <span className={cn("h-6 w-6 rounded-full text-xs font-bold flex items-center justify-center shrink-0", rankClass)}>
          {rank + 1}
        </span>

        <div className="flex-1 min-w-0 flex items-center gap-1.5">
          <span className="text-base leading-none">{osIcon(stat.os_family)}</span>
          <span className="text-sm font-semibold text-foreground">{stat.os_family}</span>
          <span className="text-xs text-muted-foreground">· {versions.length} phiên bản</span>
        </div>

        {/* User count — clickable */}
        <button
          type="button"
          onClick={() => onClickUsers(stat.os_family)}
          className="shrink-0 text-right group hover:opacity-75 transition-opacity"
          title="Xem người dùng"
        >
          <div className="flex items-center gap-1 justify-end">
            <Users className="h-3 w-3 text-muted-foreground group-hover:text-primary transition-colors" />
            <span className={cn("text-sm font-bold tabular-nums", isTop ? "text-primary" : "text-foreground")}>
              {stat.unique_users.toLocaleString()}
            </span>
          </div>
          <span className="text-[11px] text-muted-foreground">người dùng</span>
        </button>
      </div>

      {/* Actions bar */}
      <div className="relative flex items-center gap-2">
        <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
          <div
            className={cn("h-full rounded-full transition-all duration-500", isTop ? "bg-primary" : "bg-primary/40")}
            style={{ width: `${barPct}%` }}
          />
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Zap className="h-3 w-3 text-muted-foreground" />
          <span className="text-xs text-muted-foreground tabular-nums">
            {stat.total_actions.toLocaleString()} hành động
          </span>
        </div>
      </div>

      {/* Version breakdown */}
      {versions.length > 0 && (
        <div className="relative flex flex-col gap-1 pt-1 border-t border-border/50">
          {visibleVersions.map((v) => (
            <VersionRow key={v.version} v={v} maxUsers={versionMax} />
          ))}
          {showToggle && (
            <button
              type="button"
              onClick={() => setExpanded((e) => !e)}
              className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors mt-0.5 w-fit"
            >
              {expanded ? (
                <><ChevronUp className="h-3 w-3" /> Thu gọn</>
              ) : (
                <><ChevronDown className="h-3 w-3" /> +{versions.length - MAX_VERSIONS_COLLAPSED} phiên bản nữa</>
              )}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Browser group card ───────────────────────────────────────────────────────

function BrowserGroupCard({
  stat,
  rank,
  maxUsers,
  onClickUsers,
}: {
  stat: BrowserGroupStat;
  rank: number;
  maxUsers: number;
  onClickUsers: (family: string) => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const barPct = Math.round((stat.unique_users / maxUsers) * 100);
  const rankClass = RANK_COLORS[rank] ?? "bg-muted text-muted-foreground";
  const isTop = rank === 0;
  const versions = stat.versions ?? [];
  const showToggle = versions.length > MAX_VERSIONS_COLLAPSED;
  const visibleVersions = expanded ? versions : versions.slice(0, MAX_VERSIONS_COLLAPSED);
  const versionMax = versions[0]?.unique_users ?? 1;

  return (
    <div
      className={cn(
        "relative rounded-xl border bg-card p-3 overflow-hidden flex flex-col gap-2",
        isTop && "ring-1 ring-primary/20",
      )}
    >
      <div
        className="absolute inset-y-0 left-0 bg-primary/5 transition-all duration-500"
        style={{ width: `${barPct}%` }}
      />

      <div className="relative flex items-center gap-2.5">
        <span className={cn("h-6 w-6 rounded-full text-xs font-bold flex items-center justify-center shrink-0", rankClass)}>
          {rank + 1}
        </span>

        <div className="flex-1 min-w-0 flex items-center gap-1.5">
          <span className="text-base leading-none">{browserIcon(stat.browser_family)}</span>
          <span className="text-sm font-semibold text-foreground">{stat.browser_family}</span>
          <span className="text-xs text-muted-foreground">· {versions.length} phiên bản</span>
        </div>

        <button
          type="button"
          onClick={() => onClickUsers(stat.browser_family)}
          className="shrink-0 text-right group hover:opacity-75 transition-opacity"
          title="Xem người dùng"
        >
          <div className="flex items-center gap-1 justify-end">
            <Users className="h-3 w-3 text-muted-foreground group-hover:text-primary transition-colors" />
            <span className={cn("text-sm font-bold tabular-nums", isTop ? "text-primary" : "text-foreground")}>
              {stat.unique_users.toLocaleString()}
            </span>
          </div>
          <span className="text-[11px] text-muted-foreground">người dùng</span>
        </button>
      </div>

      <div className="relative flex items-center gap-2">
        <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
          <div
            className={cn("h-full rounded-full transition-all duration-500", isTop ? "bg-primary" : "bg-primary/40")}
            style={{ width: `${barPct}%` }}
          />
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <Zap className="h-3 w-3 text-muted-foreground" />
          <span className="text-xs text-muted-foreground tabular-nums">
            {stat.total_actions.toLocaleString()} hành động
          </span>
        </div>
      </div>

      {versions.length > 0 && (
        <div className="relative flex flex-col gap-1 pt-1 border-t border-border/50">
          {visibleVersions.map((v) => (
            <VersionRow key={v.version} v={v} maxUsers={versionMax} />
          ))}
          {showToggle && (
            <button
              type="button"
              onClick={() => setExpanded((e) => !e)}
              className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors mt-0.5 w-fit"
            >
              {expanded ? (
                <><ChevronUp className="h-3 w-3" /> Thu gọn</>
              ) : (
                <><ChevronDown className="h-3 w-3" /> +{versions.length - MAX_VERSIONS_COLLAPSED} phiên bản nữa</>
              )}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ─── Loading skeleton ─────────────────────────────────────────────────────────

function Skeleton() {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="h-32 rounded-xl bg-muted animate-pulse" />
      ))}
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

interface Props {
  days?: number;
}

export function BrowserPlatformStats({ days = 30 }: Props) {
  const { data: osData, isLoading: osLoading } = useOSStats(days);
  const { data: browserData, isLoading: browserLoading } = useBrowserStats(days);

  const [selectedOS, setSelectedOS] = useState<string | null>(null);
  const [selectedBrowser, setSelectedBrowser] = useState<string | null>(null);
  const handleCloseOS = useCallback(() => setSelectedOS(null), []);
  const handleCloseBrowser = useCallback(() => setSelectedBrowser(null), []);

  const osMax = osData?.[0]?.unique_users ?? 1;
  const browserMax = browserData?.[0]?.unique_users ?? 1;

  return (
    <section className="flex flex-col gap-6">
      {/* OS section */}
      <div>
        <SectionLabel icon={Smartphone}>Hệ điều hành</SectionLabel>
        <p className="text-xs text-muted-foreground mb-3">{days} ngày qua · nhấn vào số người dùng để xem chi tiết</p>

        {osLoading ? (
          <Skeleton />
        ) : !osData?.length ? (
          <AllClear text="Không có dữ liệu" />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {osData.map((stat, idx) => (
              <OSGroupCard
                key={stat.os_family}
                stat={stat}
                rank={idx}
                maxUsers={osMax}
                onClickUsers={setSelectedOS}
              />
            ))}
          </div>
        )}
      </div>

      {/* Browser section */}
      <div>
        <SectionLabel icon={Globe}>Trình duyệt</SectionLabel>
        <p className="text-xs text-muted-foreground mb-3">{days} ngày qua · nhấn vào số người dùng để xem chi tiết</p>

        {browserLoading ? (
          <Skeleton />
        ) : !browserData?.length ? (
          <AllClear text="Không có dữ liệu" />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {browserData.map((stat, idx) => (
              <BrowserGroupCard
                key={stat.browser_family}
                stat={stat}
                rank={idx}
                maxUsers={browserMax}
                onClickUsers={setSelectedBrowser}
              />
            ))}
          </div>
        )}
      </div>

      <OSUsersSheet osFamily={selectedOS} days={days} onClose={handleCloseOS} />
      <BrowserUsersSheet browserFamily={selectedBrowser} days={days} onClose={handleCloseBrowser} />
    </section>
  );
}
