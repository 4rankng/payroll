import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Calendar, CheckCircle } from "lucide-react";
import { PartnerProject } from "@/types/partnerProject";
import { getProjectStatusColor } from "@/utils/partnerProjectHelpers";

interface ProjectInfoProps {
  project: PartnerProject;
}

export const ProjectInfo = ({ project }: ProjectInfoProps) => {
  return (
    <Card className="bg-gradient-to-br from-slate-50 to-gray-100 border-0 shadow-sm">
      <CardHeader>
        <CardTitle>Thông tin dự án</CardTitle>
        <CardDescription>Thông tin cơ bản về dự án hiện tại</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <Label className="typography-body-medium typography-label-medium text-muted-foreground">Tên dự án</Label>
            <p className="text-foreground font-medium mt-1">{project.name}</p>
          </div>
          <div>
            <Label className="typography-body-medium typography-label-medium text-muted-foreground">Ngày bắt đầu</Label>
            <div className="flex items-center gap-2 mt-1">
              <Calendar className="w-4 h-4 text-muted-foreground" />
              <span className="text-foreground font-medium">{project.startDate}</span>
            </div>
          </div>
          <div>
            <Label className="typography-body-medium typography-label-medium text-muted-foreground">Trạng thái</Label>
            <div className="mt-1">
              <Badge className={getProjectStatusColor(project.status)}>
                <CheckCircle className="w-3 h-3 mr-1" />
                {project.status}
              </Badge>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};