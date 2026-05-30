interface PaytypeHierarchyProps {
  paytype: string;
  className?: string;
}

export function PaytypeHierarchy({ paytype, className = '' }: PaytypeHierarchyProps) {
  // Parse the paytype string (e.g., "phổ thông.ngày thường.ca ngày")
  const levels = paytype.split('.');

  if (levels.length < 2) {
    return (
      <span className={`text-sm font-medium text-foreground ${className}`}>{paytype}</span>
    );
  }

  const position = levels[0];
  const dayType = levels.length >= 3 ? levels[1] : '';
  const hourType = levels.length >= 3 ? levels[2] : levels[1];

  if (dayType) {
    return (
      <div className={`flex flex-col gap-0.5 min-w-0 ${className}`}>
        <span className="text-sm font-medium text-foreground leading-tight capitalize">{position}</span>
        <span className="text-xs text-muted-foreground leading-tight">{dayType} · {hourType}</span>
      </div>
    );
  }

  return (
    <div className={`flex flex-col gap-0.5 min-w-0 ${className}`}>
      <span className="text-sm font-medium text-foreground leading-tight capitalize">{position}</span>
      <span className="text-xs text-muted-foreground leading-tight">{hourType}</span>
    </div>
  );
}
