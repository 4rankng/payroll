import type { ReactNode } from "react";

export function ActionBar({ children }: { children: ReactNode }) {
  return (
    <div className="flex flex-wrap gap-1.5 items-center">
      {children}
    </div>
  );
}
