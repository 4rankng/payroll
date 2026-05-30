import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Search } from "lucide-react";

interface ResponsiveContainerProps {
  children: React.ReactNode;
  className?: string;
}

export function ResponsiveContainer({ children, className }: ResponsiveContainerProps) {
  return (
    <div className={cn("p-4 lg:p-6", className)}>
      {children}
    </div>
  );
}

interface ResponsiveHeaderProps {
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

export function ResponsiveHeader({ title, description, action, className }: ResponsiveHeaderProps) {
  return (
    <div className={cn("flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 mb-6", className)}>
      <div className="space-y-1">
        <h1 className="typography-headline-large lg:typography-display-small text-foreground">{title}</h1>
        {description && (
          <p className="typography-body-medium lg:typography-body-large text-muted-foreground">{description}</p>
        )}
      </div>
      {action && (
        <div className="flex-shrink-0">
          {action}
        </div>
      )}
    </div>
  );
}

interface ResponsiveGridProps {
  children: React.ReactNode;
  columns?: "1" | "2" | "3" | "4";
  className?: string;
}

export function ResponsiveGrid({ children, columns = "3", className }: ResponsiveGridProps) {
  const gridClasses = {
    "1": "grid-cols-1",
    "2": "grid-cols-1 md:grid-cols-2",
    "3": "grid-cols-1 md:grid-cols-2 xl:grid-cols-3",
    "4": "grid-cols-1 sm:grid-cols-2 lg:grid-cols-4"
  };

  return (
    <div className={cn("grid gap-4 lg:gap-6", gridClasses[columns], className)}>
      {children}
    </div>
  );
}

interface ResponsiveSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  children?: React.ReactNode;
  className?: string;
}

export function ResponsiveSearch({ 
  value, 
  onChange, 
  placeholder = "Tìm kiếm...", 
  children,
  className 
}: ResponsiveSearchProps) {
  return (
    <div className={cn("flex flex-col sm:flex-row items-stretch sm:items-center gap-4", className)}>
      <div className="relative flex-1 sm:max-w-sm">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="pl-10 w-full"
        />
      </div>
      {children}
    </div>
  );
}