import { cn } from "@/lib/utils";
import { EmptyState as SharedEmptyState } from "@/components/shared/EmptyState";

interface SkeletonProps {
  className?: string;
}

export function Skeleton({ className }: SkeletonProps) {
  return (
    <div
      className={cn(
        "animate-pulse rounded-md bg-muted",
        className
      )}
    />
  );
}

export function TableSkeleton() {
  return (
    <div className="space-y-3">
      <div className="flex space-x-4">
        <Skeleton className="h-4 w-[250px]" />
        <Skeleton className="h-4 w-[200px]" />
        <Skeleton className="h-4 w-[150px]" />
        <Skeleton className="h-4 w-[100px]" />
      </div>
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex space-x-4">
          <Skeleton className="h-4 w-[250px]" />
          <Skeleton className="h-4 w-[200px]" />
          <Skeleton className="h-4 w-[150px]" />
          <Skeleton className="h-4 w-[100px]" />
        </div>
      ))}
    </div>
  );
}

export function CardSkeleton() {
  return (
    <div className="rounded-lg border bg-card p-6 shadow-card">
      <div className="space-y-3">
        <Skeleton className="h-4 w-[250px]" />
        <Skeleton className="h-4 w-[200px]" />
        <Skeleton className="h-20 w-full" />
      </div>
    </div>
  );
}

export function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {Array.from({ length: 4 }).map((_, i) => (
          <CardSkeleton key={i} />
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {Array.from({ length: 2 }).map((_, i) => (
          <CardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function DashboardLoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Skeleton className="h-8 w-[300px]" />
        <Skeleton className="h-4 w-[500px]" />
      </div>
      <DashboardSkeleton />
    </div>
  );
}

export function EmptyState({ 
  title, 
  description, 
  children 
}: { 
  title: string; 
  description: string; 
  children?: React.ReactNode;
}) {
  return (
    <SharedEmptyState
      title={title}
      description={description}
      className="py-8 sm:py-10"
    >
      {children}
    </SharedEmptyState>
  );
}

export function TableLoadingSkeleton({ rows = 5, columns = 4 }: { rows?: number; columns?: number }) {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <Skeleton className="h-8 w-[200px]" />
        <Skeleton className="h-10 w-[120px]" />
      </div>
      <div className="space-y-3">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex space-x-4">
            {Array.from({ length: columns }).map((_, j) => (
              <Skeleton key={j} className="h-4 flex-1" />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export function NoEmployeesFound({ onAddEmployee }: { onAddEmployee?: () => void }) {
  return (
    <EmptyState
      title="Chưa có nhân viên"
      description="Hệ thống chưa có nhân viên nào. Hãy thêm nhân viên đầu tiên."
    />
  );
}

export function NoSearchResults({ query, onClearSearch }: { query?: string; onClearSearch?: () => void }) {
  return (
    <EmptyState
      title="Không tìm thấy kết quả"
      description="Không có nhân viên nào phù hợp với từ khóa tìm kiếm."
    />
  );
}
