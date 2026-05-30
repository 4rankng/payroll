import { cn } from '@/lib/utils';

export interface PremiumLoadingStateProps {
  type?: 'skeleton' | 'spinner' | 'progress';
  message?: string;
  className?: string;
}

export const PremiumLoadingState = ({
  type = 'skeleton',
  message,
  className,
}: PremiumLoadingStateProps) => {
  if (type === 'spinner') {
    return (
      <div className={cn('flex flex-col items-center justify-center py-12', className)}>
        <div className="relative">
          {/* Outer ring */}
          <div className="absolute inset-0 border-4 border-primary/20 rounded-full" />
          {/* Spinning arc */}
          <div className="w-12 h-12 border-4 border-transparent border-t-primary rounded-full animate-spin" />
        </div>
        {message && (
          <p className="text-sm text-muted-foreground mt-4 font-medium animate-pulse">
            {message}
          </p>
        )}
      </div>
    );
  }

  if (type === 'progress') {
    return (
      <div className={cn('flex flex-col items-center justify-center py-12 space-y-4', className)}>
        <div className="w-64 h-1 bg-muted rounded-full overflow-hidden">
          <div className="h-full bg-primary animate-pulse w-2/3 rounded-full" />
        </div>
        {message && (
          <p className="text-sm text-muted-foreground font-medium">{message}</p>
        )}
      </div>
    );
  }

  // Default skeleton type
  return (
    <div className={cn('space-y-4 animate-fade-in', className)}>
      <div className="space-y-2">
        <div className="h-4 bg-muted/50 rounded w-1/3 animate-pulse" />
        <div className="h-3 bg-muted/30 rounded w-1/2 animate-pulse delay-100" />
      </div>
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center space-x-4">
            <div className="h-12 w-12 bg-muted/50 rounded-lg animate-pulse" style={{ animationDelay: `${i * 100}ms` }} />
            <div className="flex-1 space-y-2">
              <div className="h-4 bg-muted/40 rounded w-3/4 animate-pulse" style={{ animationDelay: `${i * 100 + 50}ms` }} />
              <div className="h-3 bg-muted/30 rounded w-1/2 animate-pulse" style={{ animationDelay: `${i * 100 + 100}ms` }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
