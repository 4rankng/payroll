import type { ReactNode } from 'react';

export interface DashboardCardData {
  id: string;
  content: ReactNode;
}

export const DashboardMasonryCard = ({ index, data }: { index: number; data: DashboardCardData; width: number }) => (
  <div
    className="w-full opacity-0 animate-fade-in-up [animation-fill-mode:forwards]"
    style={{ animationDelay: `${200 + index * 50}ms` }}
  >
    {data.content}
  </div>
);
