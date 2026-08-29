import { useState } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { 
  ChevronDown,
  ChevronUp,
  Edit,
  Trash2,
  Mail,
  Calendar,
  Clock,
  EllipsisVertical
} from "lucide-react";
import { User } from "@/types/user";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface UserCardRowProps {
  user: User;
  onEdit: (user: User) => void;
  onDelete: (user: User) => void;
}

export function UserCardRow({ user, onEdit, onDelete }: UserCardRowProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .slice(0, 2)
      .map(n => n[0])
      .join('')
      .toUpperCase();
  };


  const getRoleBadge = (role: string) => {
    switch (role) {
      case "admin":
        return <Badge variant="admin" className="typography-body-small">Admin</Badge>;
      case "partner":
        return <Badge variant="partner" className="typography-body-small">Partner</Badge>;
      case "adv_partner":
        return <Badge variant="manager" className="typography-body-small">Manager</Badge>;
      case "accountant":
        return <Badge variant="role" className="typography-body-small">Kế toán</Badge>;
      case "employee":
        return <Badge variant="role" className="typography-body-small">Nhân viên</Badge>;
      default:
        return <Badge variant="role" className="typography-body-small">{role}</Badge>;
    }
  };

  return (
    <Card className="mb-3 transition-all duration-200 hover:scale-[1.01] active:scale-[0.99] touch-manipulation">
      <CardContent className="p-5 touch-manipulation">
        {/* Main User Info - Always Visible */}
        <div className="space-y-3">
          {/* Header with Avatar and Primary Info */}
          <div className="flex items-start gap-3">
            <Avatar className="w-10 h-10 flex-shrink-0">
              <AvatarFallback className="bg-primary/10 text-primary font-semibold">
                {getInitials(user.username)}
              </AvatarFallback>
            </Avatar>
            
            <div className="flex-1 min-w-0">
              {/* Username */}
              <h4 className="typography-body-large text-foreground truncate">
                {user.username}
              </h4>
              
              {/* Email */}
              <div className="flex items-center gap-1 mt-1">
                <Mail className="w-3 h-3 text-muted-foreground flex-shrink-0" />
                <span className="typography-body-medium text-muted-foreground truncate">
                  {user.email}
                </span>
              </div>
            </div>

            {/* Actions Dropdown */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button 
                  variant="ghost" 
                  className="h-10 w-10 p-0 flex-shrink-0 touch-manipulation rounded-lg hover:bg-accent/60"
                  aria-label={`Hành động cho ${user.username}`}
                >
                  <EllipsisVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={() => onEdit(user)}>
                  <Edit className="mr-2 h-4 w-4" />
                  Chỉnh sửa
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => onDelete(user)}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Xóa
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {/* Role Badge */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 flex-wrap">
              {getRoleBadge(user.role)}
            </div>
          </div>
        </div>

        {/* Secondary Info - Collapsible */}
        <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
          <CollapsibleTrigger asChild>
            <Button 
              variant="ghost" 
              size="sm" 
              className="mt-4 w-full justify-between hover:bg-accent/50 min-h-[44px] rounded-lg touch-manipulation"
              aria-expanded={isExpanded}
              aria-label={isExpanded ? "Ẩn thông tin chi tiết" : "Xem thông tin chi tiết"}
            >
              <span className="typography-body-medium">
                {isExpanded ? "Ẩn chi tiết" : "Xem chi tiết"}
              </span>
              <div className="flex items-center gap-1">
                {isExpanded ? (
                  <ChevronUp className="h-4 w-4" />
                ) : (
                  <ChevronDown className="h-4 w-4" />
                )}
              </div>
            </Button>
          </CollapsibleTrigger>
          
          <CollapsibleContent>
            <div className="pt-3 space-y-3 border-t border-border/50">
              {/* Last Login */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Clock className="w-4 h-4" />
                  <span className="typography-body-medium">Đăng nhập cuối:</span>
                </div>
                <span className="typography-body-medium">
                  {user.last_login ? (() => {
                    const d = new Date(user.last_login);
                    return isNaN(d.getTime()) ? user.last_login : d.toLocaleString('vi-VN');
                  })() : "Chưa đăng nhập"}
                </span>
              </div>
              
              {/* Created Date */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="w-4 h-4" />
                  <span className="typography-body-medium">Ngày tạo:</span>
                </div>
                <span className="typography-body-medium">
                  {user.created_at ? (() => {
                    const d = new Date(user.created_at);
                    return isNaN(d.getTime()) ? user.created_at : d.toLocaleDateString('vi-VN');
                  })() : ""}
                </span>
              </div>
              
              {/* Quick Actions */}
              <div className="flex gap-3 pt-2">
                <Button 
                  variant="outline" 
                  size="sm" 
                  onClick={() => onEdit(user)}
                  className="flex-1 min-h-[44px] touch-manipulation rounded-lg"
                  aria-label={`Chỉnh sửa thông tin ${user.username}`}
                >
                  <Edit className="w-4 h-4 mr-2" />
                  Sửa
                </Button>
                <Button 
                  variant="destructive"
                  size="sm" 
                  onClick={() => onDelete(user)}
                  className="flex-1 min-h-[44px] touch-manipulation rounded-lg"
                  aria-label={`Xóa tài khoản ${user.username}`}
                >
                  <Trash2 className="w-4 h-4 mr-2" />
                  Xóa
                </Button>
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>
      </CardContent>
    </Card>
  );
}