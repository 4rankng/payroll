import { format } from "date-fns";
import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  MoreVertical,
  RefreshCw,
  Maximize2,
  Minimize2,
  Eye,
  EyeOff,
  Settings,
  Download,
  Trash2
} from 'lucide-react';

export interface DashboardWidgetProps {
  id: string;
  title: string;
  description?: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  refreshInterval?: number;
  onRefresh?: () => void | Promise<void>;
  onRemove?: () => void;
  onSettings?: () => void;
  onExport?: () => void;
  isLoading?: boolean;
  isExpanded?: boolean;
  isVisible?: boolean;
  customActions?: React.ReactNode;
  badge?: React.ReactNode;
  lastUpdated?: Date;
}

export const DashboardWidget: React.FC<DashboardWidgetProps> = ({
  id,
  title,
  description,
  icon,
  children,
  className,
  refreshInterval,
  onRefresh,
  onRemove,
  onSettings,
  onExport,
  isLoading = false,
  isExpanded = false,
  isVisible = true,
  customActions,
  badge,
  lastUpdated
}) => {
  const [expanded, setExpanded] = useState(isExpanded);
  const [refreshing, setRefreshing] = useState(false);
  const [visible, setVisible] = useState(isVisible);

  const handleRefresh = async () => {
    if (!onRefresh) return;
    
    setRefreshing(true);
    try {
      await onRefresh();
    } finally {
      setTimeout(() => setRefreshing(false), 500);
    }
  };

  const formatLastUpdated = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    
    if (seconds < 60) return 'Vừa xong';
    if (minutes < 60) return `${minutes} phút trước`;
    if (hours < 24) return `${hours} giờ trước`;
    return format(date, 'dd/MM/yyyy');
  };

  if (!visible) {
    return null;
  }

  return (
    <Card
      className={cn(
        'relative group transition-all duration-300',
        expanded && 'md:col-span-2 lg:col-span-3',
        className
      )}
      data-widget-id={id}
    >
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            {icon && (
              <div className="p-2 rounded-xl bg-primary/10 text-primary">
                {icon}
              </div>
            )}
            <div>
              <CardTitle className="flex items-center gap-2">
                {title}
                {badge}
              </CardTitle>
              {description && (
                <CardDescription className="mt-1">
                  {description}
                </CardDescription>
              )}
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            {customActions}
            
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-8 w-8 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <MoreVertical className="h-4 w-4" />
                  <span className="sr-only">Menu</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Hành động</DropdownMenuLabel>
                <DropdownMenuSeparator />
                
                {onRefresh && (
                  <DropdownMenuItem onClick={handleRefresh}>
                    <RefreshCw className={cn(
                      "mr-2 h-4 w-4",
                      refreshing && "animate-spin"
                    )} />
                    Làm mới
                  </DropdownMenuItem>
                )}
                
                <DropdownMenuItem onClick={() => setExpanded(!expanded)}>
                  {expanded ? (
                    <>
                      <Minimize2 className="mr-2 h-4 w-4" />
                      Thu nhỏ
                    </>
                  ) : (
                    <>
                      <Maximize2 className="mr-2 h-4 w-4" />
                      Mở rộng
                    </>
                  )}
                </DropdownMenuItem>
                
                {onExport && (
                  <DropdownMenuItem onClick={onExport}>
                    <Download className="mr-2 h-4 w-4" />
                    Xuất dữ liệu
                  </DropdownMenuItem>
                )}
                
                {onSettings && (
                  <DropdownMenuItem onClick={onSettings}>
                    <Settings className="mr-2 h-4 w-4" />
                    Cài đặt
                  </DropdownMenuItem>
                )}
                
                <DropdownMenuItem onClick={() => setVisible(false)}>
                  <EyeOff className="mr-2 h-4 w-4" />
                  Ẩn widget
                </DropdownMenuItem>
                
                {onRemove && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={onRemove}
                      className="text-destructive"
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      Xóa widget
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        
        {lastUpdated && (
          <div className="flex items-center gap-1 mt-2 typography-body-small text-muted-foreground">
            <RefreshCw className="h-3 w-3" />
            Cập nhật: {formatLastUpdated(lastUpdated)}
          </div>
        )}
      </CardHeader>
      
      <CardContent className="relative">
        {isLoading || refreshing ? (
          <div className="absolute inset-0 bg-background/50 backdrop-blur-sm z-10 flex items-center justify-center">
            <RefreshCw className="h-6 w-6 animate-spin text-primary" />
          </div>
        ) : null}
        
        {children}
      </CardContent>
    </Card>
  );
};