import { format } from "date-fns";
import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  History,
  Eye,
  Copy,
  Edit3,
  Trash2
} from 'lucide-react';
import {
  FlexiblePayrateConfig,
  STATUS_LABELS
} from '../types';
import { payRateService } from '@/services/api/payrate.service';
import { EmptyState } from '@/components/shared/EmptyState';

interface ConfigurationHistoryProps {
  configs: FlexiblePayrateConfig[];
  isLoading: boolean;
  onEdit: (config: FlexiblePayrateConfig) => void;
  onView: (config: FlexiblePayrateConfig) => void;
  onCopy: (config: FlexiblePayrateConfig) => void;
  onDelete: (config: FlexiblePayrateConfig) => void;
}

export function ConfigurationHistory({ 
  configs, 
  isLoading, 
  onEdit, 
  onView, 
  onCopy, 
  onDelete 
}: ConfigurationHistoryProps) {
  const [timesheetAttachments, setTimesheetAttachments] = useState<Record<number, boolean>>({});

  // Check timesheet attachments for all configs
  useEffect(() => {
    const checkTimesheetAttachments = async () => {
      const attachmentChecks: Record<number, boolean> = {};
      
      for (const config of configs) {
        try {
          const hasTimesheets = await payRateService.hasTimesheets(config.id);
          attachmentChecks[config.id] = hasTimesheets;
        } catch (error) {
          console.warn(`Failed to check timesheets for config ${config.id}:`, error);
          // If we can't check, assume it has timesheets if it's active (safer approach)
          attachmentChecks[config.id] = config.status === 'active';
        }
      }
      
      setTimesheetAttachments(attachmentChecks);
    };

    if (configs.length > 0 && !isLoading) {
      checkTimesheetAttachments();
    }
  }, [configs, isLoading]);

  const getStatusBadge = (status: string) => {
    const colors = {
      active: 'bg-green-100 text-green-800 border-green-200',
      pending_approval: 'bg-yellow-100 text-yellow-800 border-yellow-200',
      rejected: 'bg-red-100 text-red-800 border-red-200',
      inactive: 'bg-muted text-gray-800 border-border'
    };
    
    return (
      <Badge variant="outline" className={colors[status as keyof typeof colors] || colors.inactive}>
        {STATUS_LABELS[status as keyof typeof STATUS_LABELS]}
      </Badge>
    );
  };

  const canEdit = (config: FlexiblePayrateConfig) => {
    // Can edit if config is not active and has no attached timesheets
    return config.status !== 'active' && !timesheetAttachments[config.id];
  };

  const canDelete = (config: FlexiblePayrateConfig) => {
    // Can delete if config is not active and has no attached timesheets
    return config.status !== 'active' && !timesheetAttachments[config.id];
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-40" />
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-xl bg-secondary/50 flex items-center justify-center">
              <History className="w-4 h-4 text-secondary-foreground" />
            </div>
            <div>
              <CardTitle>Lịch sử cấu hình</CardTitle>
              <p className="typography-body-medium text-muted-foreground mt-1">
                {configs.length} cấu hình đã tạo
              </p>
            </div>
          </div>
        </div>
      </CardHeader>
      
      <CardContent>
        {configs.length === 0 ? (
          <EmptyState
            title="Chưa có lịch sử cấu hình"
            description="Các cấu hình đã tạo sẽ xuất hiện ở đây"
            size="sm"
          />
        ) : (
          <div className="space-y-3">
            {configs.map((config) => (
              <div 
                key={config.id} 
                className="flex items-center justify-between p-4 border rounded-xl hover:bg-muted/30 transition-colors"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 mb-2">
                    <div className="typography-body-medium">
                      {format(new Date(config.fromDate), 'dd/MM/yyyy')} -
                      {config.toDate ? format(new Date(config.toDate), 'dd/MM/yyyy') : 'Không giới hạn'}
                    </div>
                    {getStatusBadge(config.status)}
                  </div>
                  
                  <div className="flex items-center gap-4 typography-body-small text-muted-foreground">
                    <span>Tạo ngày {format(new Date(config.created_at || ''), 'dd/MM/yyyy')}</span>
                    <span>•</span>
                    <span>{Object.keys(config.rates).length} trình độ</span>
                    <span>•</span>
                    <span>
                      {Object.values(config.rates).reduce((total, rates) => total + Object.keys(rates).length, 0)} mức lương
                    </span>
                  </div>
                </div>
                
                <div className="flex items-center gap-2 ml-4">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onView(config)}
                    className="h-8 px-2"
                  >
                    <Eye className="h-3 w-3" />
                  </Button>
                  
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onCopy(config)}
                    className="h-8 px-2"
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                  
                  {canEdit(config) && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onEdit(config)}
                      className="h-8 px-2"
                    >
                      <Edit3 className="h-3 w-3" />
                    </Button>
                  )}
                  
                  {canDelete(config) && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onDelete(config)}
                      className="h-8 px-2 text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
