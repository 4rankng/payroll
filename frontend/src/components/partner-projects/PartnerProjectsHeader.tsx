interface PartnerProjectsHeaderProps {
  totalProjects: number;
}

export const PartnerProjectsHeader = ({ totalProjects }: PartnerProjectsHeaderProps) => {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div className="space-y-1">
        <h1 className="typography-display-small text-foreground">
          Dự án
        </h1>
        <p className="typography-body-large text-muted-foreground">
          Quản lý {totalProjects} dự án đang hoạt động
        </p>
      </div>
    </div>
  );
};
