import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

interface PartnerDashboardHeaderProps {
  onAddProject: () => void;
}

export const PartnerDashboardHeader = ({ onAddProject }: PartnerDashboardHeaderProps) => {
  return (
    <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
      <div className="space-y-1">
        <h1 className="typography-headline-medium sm:typography-headline-large lg:typography-display-small text-foreground">
          Danh sách dự án
        </h1>
        <p className="typography-body-small sm:typography-body-medium lg:typography-body-large text-muted-foreground">
          Quản lý và theo dõi các dự án của bạn
        </p>
      </div>
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 sm:gap-3">
        <Button
          onClick={onAddProject}
          className="w-full sm:w-auto min-h-[44px] touch-manipulation"
        >
          <Plus className="h-4 w-4 mr-2" />
          Thêm dự án
        </Button>
      </div>
    </div>
  );
};