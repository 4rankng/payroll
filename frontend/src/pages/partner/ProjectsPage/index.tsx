import { useMemo, useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Briefcase,
  Plus,
  Search,
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
} from 'lucide-react';
import { Skeleton } from '@/components/ui/skeleton';
import { PageHeader } from '@/components/shared/PageHeader';
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
// 8 curated soft-gradient pairings, chosen for variety without clashing
const AVATAR_GRADIENTS = [
  'from-indigo-500 to-purple-600',
  'from-emerald-500 to-teal-600',
  'from-blue-500 to-cyan-600',
  'from-amber-500 to-orange-600',
  'from-pink-500 to-rose-600',
  'from-violet-500 to-fuchsia-600',
  'from-sky-500 to-blue-600',
  'from-lime-500 to-green-600',
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

// ─── Project card (replaces table row) ─────────────────────────────────────
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
        'group relative w-full rounded-2xl bg-card cursor-pointer',
        'shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_2px_8px_-4px_rgba(15,15,30,0.04)]',
        'hover:shadow-[0_2px_4px_-1px_rgba(15,15,30,0.08),0_12px_24px_-8px_rgba(15,15,30,0.10)]',
        'hover:-translate-y-0.5 transition-all duration-200',
      )}
    >
      <div className="grid grid-cols-1 md:grid-cols-[auto_1fr_auto] gap-4 p-4 md:p-5 items-center">
        {/* Avatar */}
        <ProjectAvatar project={project} />

        {/* Identity + meta */}
        <div className="min-w-0 space-y-1.5">
          <div className="flex items-baseline gap-2 flex-wrap">
            <h3 className="text-[15px] font-bold text-foreground tracking-tight leading-tight truncate">
              {project.name}
            </h3>
            {project.code && (
              <span className="text-[11px] font-mono font-semibold text-muted-foreground/80 px-1.5 py-0.5 rounded bg-muted/60">
                {project.code}
              </span>
            )}
          </div>

          <p className="text-[12.5px] text-muted-foreground truncate">
            <span className="text-foreground/80 font-medium">{project.client_name}</span>
            {project.description ? <span> · {project.description}</span> : null}
          </p>

          <div className="flex items-center gap-2 flex-wrap pt-1">
            {/* Status indicator — icon only, with tooltip via title attr */}
            <span
              className={cn(
                'inline-flex items-center justify-center h-5 w-5 rounded-full',
                status.pill,
              )}
              title={status.label}
              aria-label={status.label}
            >
              <StatusIcon className="h-3 w-3" />
            </span>

            <SalaryPeriodChip project={project} />

            <span className="inline-flex items-center gap-1 text-[11.5px] text-muted-foreground">
              <Users className="h-3 w-3" />
              <span className="font-semibold text-foreground tabular-nums">
                {project.employee_count}
              </span>
              nhân viên
            </span>
          </div>
        </div>

        {/* Quick actions (right side) — Bảng công only, hover-revealed.
            Card itself is the "Mở" target (whole-row click handler). */}
        <div className="flex items-center justify-end shrink-0">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onTimesheet();
            }}
            className={cn(
              'inline-flex items-center gap-1.5 rounded-xl px-3 py-2 text-[12px] font-semibold transition-all',
              'bg-muted/50 text-foreground/70 hover:bg-muted hover:text-foreground',
              'md:opacity-0 md:group-hover:opacity-100',
            )}
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

  const allProjects = response?.data || [];
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

  return (
    <div className="p-4 lg:p-8 max-w-[1320px] mx-auto space-y-6">
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

      {/* ── UNIFIED TOOLBAR ── */}
      <div className="rounded-2xl bg-card p-2.5 shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06),0_2px_6px_-2px_rgba(15,15,30,0.04)]">
        <div className="flex items-center gap-2 flex-wrap">
          {/* Status filter pills */}
          <div className="flex items-center gap-1 flex-wrap">
            {STATUS_FILTERS.map((f) => {
              const isActive = statusFilter === f.value;
              return (
                <button
                  key={f.value}
                  type="button"
                  onClick={() => setStatusFilter(f.value)}
                  className={cn(
                    'px-3 py-1.5 rounded-full text-[12px] font-semibold transition-all shrink-0',
                    isActive
                      ? 'bg-foreground text-background shadow-sm'
                      : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
                  )}
                >
                  {f.label}
                </button>
              );
            })}
          </div>

          <div className="hidden md:block h-5 w-px bg-border mx-1" />

          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <input
              type="text"
              value={filterControls.searchTerm}
              onChange={(e) => filterControls.setSearchTerm(e.target.value)}
              placeholder="Tìm tên dự án, mã, khách hàng…"
              className="w-full h-9 pl-9 pr-3 rounded-xl bg-muted/40 text-[12.5px] text-foreground placeholder:text-muted-foreground/70 border border-transparent focus:border-primary/40 focus:bg-card focus:outline-none focus:ring-2 focus:ring-primary/10 transition-all"
            />
          </div>

          {filterControls.hasFilters && (
            <button
              type="button"
              onClick={filterControls.clearFilters}
              className="text-[11.5px] font-medium text-muted-foreground hover:text-foreground transition-colors px-2"
            >
              Xóa lọc
            </button>
          )}
        </div>
      </div>

      {/* ── PROJECT CARDS LIST ── */}
      {isLoading ? (
        <div className="space-y-2.5">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[108px] w-full rounded-2xl" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <div className="rounded-2xl bg-card py-16 px-6 text-center shadow-[0_1px_2px_-1px_rgba(15,15,30,0.06)]">
          <div className="inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-muted/60 mb-3">
            <Briefcase className="h-6 w-6 text-muted-foreground/50" />
          </div>
          <p className="text-[14px] font-semibold text-foreground">Không có dự án nào</p>
          <p className="text-[12px] text-muted-foreground mt-1 max-w-sm mx-auto">
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
  );
};

export default ProjectsPage;
