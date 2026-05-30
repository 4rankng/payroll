import type { ReactNode } from "react";

export function ButtonGroup({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-center gap-px rounded-xl border border-border overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      {children}
    </div>
  );
}
