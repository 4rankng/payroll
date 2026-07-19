import { useMemo, useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Briefcase,
  Building2,
  Clock3,
  FolderKanban,
  Plus,
  Users,
  Calendar,
  ArrowRight,
  ChevronLeft,
  ChevronRight,
  CalendarDays,
  Pause,
  CheckCircle2,
  XCircle,
  FileText,
  FilterX,
} from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { PageHeader } from '@/components/shared/PageHeader';
import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { useProjects } from '@/hooks/api/useProjects';
import { useProjectFilters } from '@/hooks/projects/useProjectFilters';
import { useProjectModals } from '@/hooks/useModalNavigation';
import type { Project } from '@/types/api/project.types';
import { cn } from '@/lib/utils';

// ─── Status meta ──────────────────────────────────────────────────────────
type ProjectStatus = Project['status'];

const STATUS_META: Record<
  ProjectStatus,
  { label: string; pill: string; dot: string; icon: typeof Briefcase }
> = {
  active: {
    label: 'Đang hoạt động',
    pill: 'bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-200/60',
    dot: 'bg-emerald-500',
    icon: CheckCircle2,
  },
  draft: {
    label: 'Bản nháp',
    pill: 'bg-slate-50 text-slate-700 ring-1 ring-inset ring-slate-200',
    dot: 'bg-slate-400',
    icon: FileText,
  },
  paused: {
    label: 'Tạm dừng',
    pill: 'bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-200/60',
    dot: 'bg-amber-500',
    icon: Pause,
  },
  completed: {
    label: 'Hoàn thành',
    pill: 'bg-blue-50 text-blue-700 ring-1 ring-inset ring-blue-200/60',
    dot: 'bg-blue-500',
    icon: CheckCircle2,
  },
  cancelled: {
    label: 'Đã hủy',
    pill: 'bg-rose-50 text-rose-700 ring-1 ring-inset ring-rose-200/60',
    dot: 'bg-rose-500',
    icon: XCircle,
  },
};

// ─── Project avatar (color identity) ───────────────────────────────────────
// Restrained project identities: distinct enough to scan, cohesive enough to
// remain part of the TingTing partner workspace.
const AVATAR_GRADIENTS = [
  'from-emerald-600 to-teal-600',
  'from-teal-600 to-cyan-700',
  'from-green-700 to-emerald-600',
  'from-slate-600 to-slate-700',
];

function hashToGradient(input: string): string {
  let h = 0;
  for (let i = 0; i < input.length; i++) h = (h * 31 + input.charCodeAt(i)) >>> 0;
  return AVATAR_GRADIENTS[h % AVATAR_GRADIENTS.length];
}

function projectInitials(project: Project): string {
  // Prefer the first word of the project name if it looks like a short acronym
  // (e.g. "EPE KCN Vsip" → "EPE", "PQC Nam Định" → "PQC"). This is more
  // distinguishing than `code` since codes often share prefixes
  // (EX003, EX004 → both render "EX0" which is visually indistinct).
  const words = project.name.trim().split(/\s+/).filter(Boolean);
  if (words.length > 0) {
    const first = words[0];
    if (first.length <= 4) return first.toUpperCase();
    if (words.length >= 2) return (first[0] + words[words.length - 1][0]).toUpperCase();
    return first.slice(0, 3).toUpperCase();
  }
  if (project.code) return project.code.slice(0, 3).toUpperCase();
  return '?';
}

function ProjectAvatar({ project, size = 'md' }: { project: Project; size?: 'sm' | 'md' }) {
  const gradient = hashToGradient(project.code || project.name);
  const initials = projectInitials(project);
  return (
    <div
      className={cn(
        'flex items-center justify-center rounded-2xl font-bold text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.18),0_4px_12px_-4px_rgba(15,15,30,0.18)] shrink-0',
        'bg-gradient-to-br',
        gradient,
        size === 'md' && 'h-12 w-12 text-[13px] tracking-tight',
        size === 'sm' && 'h-9 w-9 text-[11px] tracking-tight',
      )}
    >
      {initials}
    </div>
  );
}

// ─── Salary period chip ────────────────────────────────────────────────────
function SalaryPeriodChip({ project }: { project: Project }) {
  if (project.is_weekly) {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-blue-50 px-2 py-0.5 text-[10.5px] font-semibold text-blue-700 ring-1 ring-inset ring-blue-200/60">
        <CalendarDays className="h-3 w-3" />
        Lương tuần
      </span>
    );
  }
  if (project.is_monthly) {
    const period =
      project.salary_period_from != null && project.salary_period_to != null
        ? `· ngày ${project.salary_period_from}–${project.salary_period_to}`
        : '';
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-violet-50 px-2 py-0.5 text-[10.5px] font-semibold text-violet-700 ring-1 ring-inset ring-violet-200/60">
        <Calendar className="h-3 w-3" />
        Lương tháng {period}
      </span>
    );
  }
  return null;
}

// ─── Project card ──────────────────────────────────────────────────────────
function ProjectCard({
  project,
  onOpen,
  onTimesheet,
}: {
  project: Project;
  onOpen: () => void;
  onTimesheet: () => void;
}) {
  const status = STATUS_META[project.status] ?? STATUS_META.active;
  const StatusIcon = status.icon;

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onOpen();
        }
      }}
      className={cn(
        'group relative w-full overflow-hidden rounded-2xl border border-border/70 bg-card cursor-pointer',
        'shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_8px_18px_-16px_rgba(15,15,30,0.18)]',
        'hover:-translate-y-0.5 hover:border-primary/30 hover:shadow-[0_18px_34px_-22px_rgba(6,101,52,0.38)] transition-all duration-200',
      )}
    >
      <div className="absolute inset-y-0 left-0 w-1 bg-gradient-to-b from-primary via-primary/65 to-emerald-300" />
      <div className="p-4 pl-5 md:p-5 md:pl-6">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <ProjectAvatar project={project} />
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className={cn('inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-[10px] font-bold', status.pill)}>
                  <StatusIcon className="h-3 w-3" />
                  {status.label}
                </span>
                {project.code && <span className="font-mono text-[10.5px] font-semibold tracking-wide text-muted-foreground">{project.code}</span>}
              </div>
              <h3 className="mt-2 line-clamp-2 text-[15px] font-bold leading-snug tracking-tight text-foreground">
                {project.name}
              </h3>
            </div>
          </div>
          <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform duration-200 group-hover:translate-x-1 group-hover:text-primary" aria-hidden="true" />
        </div>

        <div className="mt-4 border-y border-border/55 py-3">
          <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
            <Building2 className="h-3.5 w-3.5 shrink-0 text-primary/75" />
            <span className="truncate font-medium text-foreground/85">{project.client_name || 'Chưa cập nhật khách hàng'}</span>
          </div>
          {project.description && <p className="mt-2 line-clamp-1 text-[11.5px] text-muted-foreground">{project.description}</p>}
        </div>

        <div className="mt-3 flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2">
            <span className="inline-flex items-center gap-1.5 text-[11.5px] text-muted-foreground">
              <Users className="h-3.5 w-3.5" />
              <span className="font-bold tabular-nums text-foreground">{project.employee_count}</span>
              nhân sự
            </span>
            <span className="h-3.5 w-px bg-border" />
            <SalaryPeriodChip project={project} />
          </div>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onTimesheet();
            }}
            className="inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-lg bg-primary/8 px-2.5 text-[11.5px] font-bold text-primary transition-colors hover:bg-primary hover:text-primary-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
          >
            <CalendarDays className="h-3.5 w-3.5" />
            Bảng công
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Status filter tab ─────────────────────────────────────────────────────
const STATUS_FILTERS: Array<{ value: ProjectStatus | 'all'; label: string }> = [
  { value: 'all', label: 'Tất cả' },
  { value: 'active', label: 'Đang hoạt động' },
  { value: 'paused', label: 'Tạm dừng' },
  { value: 'completed', label: 'Hoàn thành' },
  { value: 'draft', label: 'Bản nháp' },
];

// ─── Main page ────────────────────────────────────────────────────────────
const ProjectsPage = () => {
  const navigate = useNavigate();
  const filterControls = useProjectFilters({
    initialFilters: { page: 1, pageSize: 10 },
  });
  const { data: response, isLoading } = useProjects(filterControls.apiFilters);
  const { openCreateProject, openPartnerProjectDetails } = useProjectModals();

  const [statusFilter, setStatusFilter] = useState<ProjectStatus | 'all'>('all');

  const allProjects = useMemo(() => response?.data || [], [response?.data]);
  // Client-side status filter (on top of search/server pagination)
  const projects = useMemo(() => {
    if (statusFilter === 'all') return allProjects;
    return allProjects.filter((p) => p.status === statusFilter);
  }, [allProjects, statusFilter]);

  const handleProjectClick = useCallback(
    (project: Project) => openPartnerProjectDetails(project.id.toString()),
    [openPartnerProjectDetails],
  );

  const handleTimesheet = useCallback(
    (project: Project) => navigate(`/partner/timesheet?project=${project.id}`),
    [navigate],
  );

  const pagination = response?.pagination;
  const totalPages = pagination?.totalPages ?? 1;
  const currentPage = pagination?.page ?? 1;
  const activeCount = allProjects.filter((p) => p.status === 'active').length;

  return (
    <div className="min-h-full bg-[radial-gradient(circle_at_top_right,rgba(8,120,62,0.10),transparent_31rem)] px-4 py-5 lg:px-8 lg:py-7">
      <div className="mx-auto max-w-[1320px] space-y-5">
        <div className="relative overflow-hidden rounded-3xl border border-emerald-100 bg-card p-5 shadow-[0_20px_54px_-42px_rgba(6,101,52,0.44)] md:p-6">
          <div className="pointer-events-none absolute -right-16 -top-16 h-48 w-48 rounded-full border-[28px] border-emerald-100/65" />
          <div className="relative">
          <PageHeader
            title="Dự án"
            description="Quản lý và theo dõi các dự án được phân quyền"
            actions={[
              {
                label: 'Thêm dự án',
                onClick: () => openCreateProject(),
                icon: Plus,
                variant: 'default' as const,
              },
            ]}
          />
          <div className="mt-6 grid gap-2 sm:grid-cols-3">
            {[
              { label: 'Tổng dự án', value: allProjects.length, icon: FolderKanban, tone: 'text-slate-600 bg-slate-100' },
              { label: 'Đang hoạt động', value: activeCount, icon: CheckCircle2, tone: 'text-emerald-700 bg-emerald-100' },
              { label: 'Đang xem', value: projects.length, icon: Clock3, tone: 'text-amber-700 bg-amber-100' },
            ].map((item) => (
              <div key={item.label} className="flex items-center gap-3 rounded-2xl border border-border/60 bg-muted/25 px-3.5 py-3">
                <div className={cn('flex h-9 w-9 items-center justify-center rounded-xl', item.tone)}>
                  <item.icon className="h-4 w-4" />
                </div>
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-[0.09em] text-muted-foreground">{item.label}</p>
                  <p className="mt-0.5 text-xl font-bold tabular-nums text-foreground">{item.value.toLocaleString('vi-VN')}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
        </div>

      {/* ── UNIFIED TOOLBAR ── */}
      <div className="rounded-2xl border border-border/60 bg-card p-3 shadow-[0_8px_22px_-20px_rgba(15,23,42,0.40)]">
        <div className="flex items-center gap-2.5 flex-wrap">
          <div className="hidden items-center gap-2 border-r border-border pr-3 lg:flex">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary"><FolderKanban className="h-4 w-4" /></div>
            <span className="text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">Lọc danh sách</span>
          </div>

          {/* Status filter (single-select, matches D3's FilterPill pattern) */}
          <FilterPill
            value={statusFilter}
            onChange={(v) => setStatusFilter(v as ProjectStatus | 'all')}
            placeholder="Trạng thái"
            options={STATUS_FILTERS.filter(f => f.value !== 'all').map(f => ({ value: f.value, label: f.label }))}
          />

          <div className="hidden md:block h-6 w-px bg-border mx-0.5" />

          {/* Search — shared SearchBar primitive (matches D3 Employees) */}
          <SearchBar
            searchTerm={filterControls.searchTerm}
            onSearchChange={filterControls.setSearchTerm}
            placeholder="Tìm tên dự án, mã, khách hàng…"
            className="h-10 min-w-[260px] flex-1 rounded-xl bg-muted/45 text-[12.5px]"
          />

          {filterControls.hasFilters && (
            <button
              type="button"
              onClick={filterControls.clearFilters}
              className="inline-flex min-h-9 items-center gap-1.5 rounded-lg px-2 text-[11.5px] font-semibold text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <FilterX className="h-3.5 w-3.5" />
              Xóa lọc
            </button>
          )}
        </div>
      </div>

      {/* ── PROJECT CARDS LIST ── */}
      {isLoading ? (
        <div className="space-y-2.5">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[108px] w-full rounded-2xl bg-white/70" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-slate-300/80 bg-white/78 py-20 px-6 text-center shadow-[0_18px_48px_-40px_rgba(15,23,42,0.45)] backdrop-blur">
          <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-slate-100 mb-4 ring-1 ring-slate-200">
            <Briefcase className="h-6 w-6 text-muted-foreground/50" />
          </div>
          <p className="text-base font-bold text-foreground">Không có dự án nào</p>
          <p className="text-sm text-muted-foreground mt-1 max-w-sm mx-auto">
            {statusFilter !== 'all' || filterControls.hasFilters
              ? 'Thử bỏ bộ lọc hoặc đổi từ khoá tìm kiếm.'
              : 'Bạn chưa được phân quyền truy cập vào dự án nào.'}
          </p>
          {(statusFilter !== 'all' || filterControls.hasFilters) && (
            <button
              type="button"
              onClick={() => {
                setStatusFilter('all');
                filterControls.clearFilters();
              }}
              className="mt-4 inline-flex items-center gap-1.5 rounded-xl bg-muted/60 px-3 py-1.5 text-[12px] font-semibold text-foreground hover:bg-muted transition-colors"
            >
              Xóa bộ lọc
            </button>
          )}
        </div>
      ) : (
        <div className="animate-fade-in-up">
          <div className="flex items-center gap-1.5 mb-2.5 px-1 text-[11.5px] text-muted-foreground">
            <ArrowRight className="h-3 w-3" />
            <span>Nhấn vào dự án để xem chi tiết</span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2.5">
            {projects.map((project) => (
              <ProjectCard
                key={project.id}
                project={project}
                onOpen={() => handleProjectClick(project)}
                onTimesheet={() => handleTimesheet(project)}
              />
            ))}
          </div>
        </div>
      )}

      {/* ── PAGINATION ── */}
      {pagination && pagination.totalRecords > pagination.pageSize && (
        <div className="flex items-center justify-between gap-3 px-1 flex-wrap">
          <p className="text-[12px] text-muted-foreground">
            Hiển thị{' '}
            <span className="font-semibold text-foreground tabular-nums">
              {(currentPage - 1) * pagination.pageSize + 1}
            </span>
            –
            <span className="font-semibold text-foreground tabular-nums">
              {Math.min(currentPage * pagination.pageSize, pagination.totalRecords)}
            </span>{' '}
            trong{' '}
            <span className="font-semibold text-foreground tabular-nums">
              {pagination.totalRecords}
            </span>{' '}
            dự án
          </p>

          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => filterControls.setPage(Math.max(1, currentPage - 1))}
              disabled={currentPage <= 1}
              className={cn(
                'inline-flex items-center justify-center h-8 w-8 rounded-lg text-foreground/70 transition-all',
                'hover:bg-muted/60 hover:text-foreground',
                'disabled:opacity-30 disabled:cursor-not-allowed',
              )}
              aria-label="Trang trước"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </button>

            {Array.from({ length: totalPages }, (_, i) => i + 1)
              .filter((p) => {
                if (totalPages <= 7) return true;
                if (p === 1 || p === totalPages) return true;
                if (Math.abs(p - currentPage) <= 1) return true;
                return false;
              })
              .map((p, idx, arr) => {
                const prev = arr[idx - 1];
                const gap = prev !== undefined && p - prev > 1;
                return (
                  <span key={p} className="inline-flex items-center gap-1">
                    {gap && <span className="text-muted-foreground/60 px-1 text-[11px]">…</span>}
                    <button
                      type="button"
                      onClick={() => filterControls.setPage(p)}
                      className={cn(
                        'min-w-8 h-8 px-2 rounded-lg text-[12px] font-semibold transition-all tabular-nums',
                        p === currentPage
                          ? 'bg-foreground text-background shadow-sm'
                          : 'text-foreground/70 hover:bg-muted/60 hover:text-foreground',
                      )}
                    >
                      {p}
                    </button>
                  </span>
                );
              })}

            <button
              type="button"
              onClick={() => filterControls.setPage(Math.min(totalPages, currentPage + 1))}
              disabled={currentPage >= totalPages}
              className={cn(
                'inline-flex items-center justify-center h-8 w-8 rounded-lg text-foreground/70 transition-all',
                'hover:bg-muted/60 hover:text-foreground',
                'disabled:opacity-30 disabled:cursor-not-allowed',
              )}
              aria-label="Trang sau"
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
      </div>
    </div>
  );
};

export default ProjectsPage;
