import { cn } from "@/lib/utils";

export type EmptyStateIllustrationVariant =
  | 'activity'
  | 'employees'
  | 'finance'
  | 'projects'
  | 'records'
  | 'search';

interface EmptyStateIllustrationProps {
  className?: string;
  variant?: EmptyStateIllustrationVariant;
}

const ILLUSTRATION_SOURCES: Record<EmptyStateIllustrationVariant, string> = {
  activity: '/images/empty-states/activity-empty-state-illustration.webp',
  employees: '/images/empty-states/employees-empty-state-illustration.webp',
  finance: '/images/empty-states/finance-empty-state-illustration.webp',
  projects: '/images/empty-states/projects-empty-state-illustration.webp',
  records: '/images/empty-states/records-empty-state-illustration.webp',
  search: '/images/empty-states/search-empty-state-illustration.webp',
};

/** A context-specific visual language for data, search, and history empty states. */
export function EmptyStateIllustration({ className, variant = 'records' }: EmptyStateIllustrationProps) {
  return (
    <img
      src={ILLUSTRATION_SOURCES[variant]}
      alt=""
      aria-hidden="true"
      className={cn("h-20 w-20 object-contain sm:h-24 sm:w-24", className)}
    />
  );
}
