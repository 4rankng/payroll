import { format } from "date-fns";
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Checkbox } from '@/components/ui/checkbox';
import { Eye, MessageSquare, FileText } from 'lucide-react';
import { ApprovalItem } from '@/types/approval';
import { 
  getPriorityColor, 
  getStatusColor, 
  getTypeLabel, 
  getTypeColor,
  getInitials 
} from '@/utils/approvalHelpers';

interface ApprovalItemCardProps {
  item: ApprovalItem;
  isSelected: boolean;
  onToggleSelection: () => void;
  onViewDetails: () => void;
}

export const ApprovalItemCard = ({ 
  item, 
  isSelected, 
  onToggleSelection, 
  onViewDetails 
}: ApprovalItemCardProps) => {
  return (
    <Card className="transition-shadow">
      <CardContent className="p-4">
        <div className="flex items-start gap-3">
          <Checkbox
            checked={isSelected}
            onCheckedChange={onToggleSelection}
            className="mt-1"
          />
          
          <Avatar className="w-10 h-10 mt-1">
            <AvatarImage src={item.submittedBy.avatar} />
            <AvatarFallback>
              {getInitials(item.submittedBy.name)}
            </AvatarFallback>
          </Avatar>

          <div className="flex-1 min-w-0">
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1 min-w-0">
                <h3 className="typography-title-large truncate">{item.title}</h3>
                <p className="text-muted-foreground typography-body-medium mt-1">
                  {item.description}
                </p>
                
                <div className="flex flex-wrap items-center gap-2 mt-2">
                  <Badge className={getTypeColor(item.type)}>
                    {getTypeLabel(item.type)}
                  </Badge>
                  <Badge className={getPriorityColor(item.priority)}>
                    {item.priority}
                  </Badge>
                  <Badge className={getStatusColor(item.status)}>
                    {item.status}
                  </Badge>
                </div>

                <div className="flex items-center gap-4 mt-3 typography-body-small text-muted-foreground">
                  <span>Gửi bởi: {item.submittedBy.name}</span>
                  <span>•</span>
                  <span>{item.submittedAt}</span>
                  {item.dueDate && (
                    <>
                      <span>•</span>
                      <span>Hạn: {item.dueDate && !isNaN(new Date(item.dueDate).getTime()) ? format(new Date(item.dueDate), 'dd/MM/yyyy') : item.dueDate}</span>
                    </>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-2">
                {item.attachments && item.attachments > 0 && (
                  <Badge variant="outline" className="typography-body-small">
                    <FileText className="w-3 h-3 mr-1" />
                    {item.attachments}
                  </Badge>
                )}
                {item.comments && item.comments > 0 && (
                  <Badge variant="outline" className="typography-body-small">
                    <MessageSquare className="w-3 h-3 mr-1" />
                    {item.comments}
                  </Badge>
                )}
                
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={onViewDetails}
                >
                  <Eye className="w-4 h-4" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};