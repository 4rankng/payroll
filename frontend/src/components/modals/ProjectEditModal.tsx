import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useState, useEffect } from "react";
import { Project, ProjectFormData, ProjectPriority } from "@/types/api/project.types";
import { X } from "lucide-react";

export const modalConfig = {
  id: 'project-edit',
};

interface ProjectEditModalProps {
  project: Project | null;
  isOpen: boolean;
  onClose: () => void;
  onProjectUpdate: (projectId: number, projectData: ProjectFormData) => void;
}

export function ProjectEditModal({
  project,
  isOpen,
  onClose,
  onProjectUpdate
}: ProjectEditModalProps) {
  const [formData, setFormData] = useState<ProjectFormData>({
    name: "",
    client_name: "",
    code: "",
    manager: "",
    start_date: "",
    end_date: "",
    budget: 0,
    priority: "medium",
    description: "",
  });

  useEffect(() => {
    if (project) {
      setFormData({
        name: project.name,
        client_name: project.client_name,
        code: project.code,
        manager: project.manager,
        start_date: project.start_date || project.startDate || "",
        end_date: project.end_date || project.endDate || "",
        budget: project.budget,
        priority: ["cao", "high"].includes(project.priority?.toLowerCase() || "") ? "high" :
                 ["trung bình", "medium"].includes(project.priority?.toLowerCase() || "") ? "medium" :
                 ["thấp", "low"].includes(project.priority?.toLowerCase() || "") ? "low" : "medium",
        description: project.description || "",
      });
    }
  }, [project]);

  const handleSubmit = () => {
    if (project) {
      onProjectUpdate(project.id, formData);
      onClose();
    }
  };

  const handleInputChange = (field: keyof ProjectFormData, value: string | number) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  if (!project) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl">
        <DialogClose asChild>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-4 right-4 h-8 w-8 rounded-xl hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <X className="h-4 w-4" />
            <span className="sr-only">Đóng</span>
          </Button>
        </DialogClose>
        <DialogHeader>
          <DialogTitle>Chỉnh sửa dự án</DialogTitle>
          <DialogDescription>
            Cập nhật thông tin chi tiết cho dự án
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-2 gap-4 py-4">
          <div className="col-span-2 space-y-2">
            <Label htmlFor="editProjectName">Tên dự án</Label>
            <Input
              id="editProjectName"
              placeholder="Dự án ABC - Mô tả ngắn"
              value={formData.name}
              onChange={(e) => handleInputChange("name", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editClient">Khách hàng</Label>
            <Input
              id="editClient"
              placeholder="Tên công ty khách hàng"
              value={formData.client_name}
              onChange={(e) => handleInputChange("client_name", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editManager">Dự án</Label>
            <Input
              id="editManager"
              placeholder="Tên Dự án"
              value={formData.manager}
              onChange={(e) => handleInputChange("manager", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editStartDate">Ngày bắt đầu</Label>
            <Input
              id="editStartDate"
              type="date"
              value={formData.start_date}
              onChange={(e) => handleInputChange("start_date", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editEndDate">Ngày kết thúc</Label>
            <Input
              id="editEndDate"
              type="date"
              value={formData.end_date}
              onChange={(e) => handleInputChange("end_date", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editBudget">Ngân sách (₫)</Label>
            <Input
              id="editBudget"
              type="number"
              placeholder="500000000"
              value={formData.budget}
              onChange={(e) => handleInputChange("budget", parseInt(e.target.value) || 0)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="editPriority">Mức độ ưu tiên</Label>
            <Select value={formData.priority} onValueChange={(value: ProjectPriority) => handleInputChange("priority", value)}>
              <SelectTrigger>
                <SelectValue placeholder="Chọn mức độ" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="high">Cao</SelectItem>
                <SelectItem value="medium">Trung bình</SelectItem>
                <SelectItem value="low">Thấp</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="col-span-2 space-y-2">
            <Label htmlFor="editDescription">Mô tả</Label>
            <Textarea
              id="editDescription"
              placeholder="Mô tả chi tiết về dự án..."
              value={formData.description}
              onChange={(e) => handleInputChange("description", e.target.value)}
            />
          </div>
        </div>

        <div className="flex justify-end space-x-2">
          <Button variant="outline" onClick={onClose}>
            Đóng
          </Button>
          <Button onClick={handleSubmit} variant="default">
            Cập nhật
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
