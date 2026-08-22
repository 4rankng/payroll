import { useState, useEffect } from 'react';
import { DndContext, DragEndEvent, DragOverlay, DragStartEvent, closestCenter } from '@dnd-kit/core';
import { SortableContext, arrayMove, rectSortingStrategy } from '@dnd-kit/sortable';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Grip, Plus, Settings, RotateCw } from 'lucide-react';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/components/shared/EmptyState';

export interface WidgetConfig {
  id: string;
  type: string;
  title: string;
  description: string;
  visible: boolean;
  position: number;
  settings?: Record<string, unknown>;
}

interface SortableWidgetProps {
  id: string;
  children: React.ReactNode;
  isDragging?: boolean;
}

const SortableWidget: React.FC<SortableWidgetProps> = ({ id, children, isDragging }) => {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging: isSortableDragging,
  } = useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isSortableDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        'relative',
        isDragging && 'z-50'
      )}
    >
      <div
        className="absolute top-2 left-2 z-10 opacity-0 hover:opacity-100 transition-opacity cursor-move p-2 bg-background/80 backdrop-blur-sm rounded-xl"
        {...attributes}
        {...listeners}
      >
        <Grip className="h-4 w-4 text-muted-foreground" />
      </div>
      {children}
    </div>
  );
};

interface WidgetManagerProps {
  widgets: WidgetConfig[];
  onWidgetUpdate: (widgets: WidgetConfig[]) => void;
  availableWidgetTypes: {
    type: string;
    title: string;
    description: string;
    defaultSettings?: Record<string, unknown>;
  }[];
  renderWidget: (config: WidgetConfig) => React.ReactNode;
  className?: string;
}

export const WidgetManager: React.FC<WidgetManagerProps> = ({
  widgets,
  onWidgetUpdate,
  availableWidgetTypes,
  renderWidget,
  className
}) => {
  const [localWidgets, setLocalWidgets] = useState(widgets);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [showManagerDialog, setShowManagerDialog] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);

  useEffect(() => {
    setLocalWidgets(widgets);
  }, [widgets]);

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = localWidgets.findIndex(w => w.id === active.id);
      const newIndex = localWidgets.findIndex(w => w.id === over.id);

      const newWidgets = arrayMove(localWidgets, oldIndex, newIndex).map((w, i) => ({
        ...w,
        position: i
      }));

      setLocalWidgets(newWidgets);
      onWidgetUpdate(newWidgets);
    }

    setActiveId(null);
  };

  const toggleWidgetVisibility = (widgetId: string) => {
    const newWidgets = localWidgets.map(w =>
      w.id === widgetId ? { ...w, visible: !w.visible } : w
    );
    setLocalWidgets(newWidgets);
    onWidgetUpdate(newWidgets);
  };

  const addWidget = (type: string) => {
    const widgetType = availableWidgetTypes.find(w => w.type === type);
    if (!widgetType) return;

    const newWidget: WidgetConfig = {
      id: `widget-${Date.now()}`,
      type: widgetType.type,
      title: widgetType.title,
      description: widgetType.description,
      visible: true,
      position: localWidgets.length,
      settings: widgetType.defaultSettings || {}
    };

    const newWidgets = [...localWidgets, newWidget];
    setLocalWidgets(newWidgets);
    onWidgetUpdate(newWidgets);
    setShowAddDialog(false);
  };

  const removeWidget = (widgetId: string) => {
    const newWidgets = localWidgets
      .filter(w => w.id !== widgetId)
      .map((w, i) => ({ ...w, position: i }));
    setLocalWidgets(newWidgets);
    onWidgetUpdate(newWidgets);
  };

  const resetLayout = () => {
    // Reset to default layout
    const defaultWidgets = localWidgets.map((w, i) => ({
      ...w,
      visible: true,
      position: i
    }));
    setLocalWidgets(defaultWidgets);
    onWidgetUpdate(defaultWidgets);
  };

  const visibleWidgets = localWidgets
    .filter(w => w.visible)
    .sort((a, b) => a.position - b.position);

  return (
    <>
      <div className={cn('space-y-4', className)}>
        {/* Widget Controls */}
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowAddDialog(true)}
          >
            <Plus className="h-4 w-4 mr-2" />
            Thêm widget
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowManagerDialog(true)}
          >
            <Settings className="h-4 w-4 mr-2" />
            Quản lý widget
          </Button>
        </div>

        {/* Widget Grid */}
        <DndContext
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={visibleWidgets.map(w => w.id)}
            strategy={rectSortingStrategy}
          >
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {visibleWidgets.map((widget) => (
                <SortableWidget
                  key={widget.id}
                  id={widget.id}
                  isDragging={activeId === widget.id}
                >
                  {renderWidget(widget)}
                </SortableWidget>
              ))}
            </div>
          </SortableContext>

          <DragOverlay>
            {activeId ? (
              <div className="opacity-50">
                {renderWidget(localWidgets.find(w => w.id === activeId)!)}
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>

        {visibleWidgets.length === 0 && (
          <div className="border-2 border-dashed rounded-xl px-4">
            <EmptyState
              title="Không có widget nào"
              description="Thêm widget để tùy chỉnh dashboard của bạn"
              action={{ label: 'Thêm widget đầu tiên', onClick: () => setShowAddDialog(true) }}
              size="sm"
            />
          </div>
        )}
      </div>

      {/* Widget Manager Dialog */}
      <Dialog open={showManagerDialog} onOpenChange={setShowManagerDialog}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Quản lý Widget Dashboard</DialogTitle>
            <DialogDescription>
              Tùy chỉnh hiển thị và thứ tự các widget trên dashboard
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {localWidgets.map((widget) => (
              <div
                key={widget.id}
                className="flex items-center justify-between p-3 border rounded-xl"
              >
                <div className="flex items-center gap-3">
                  <Grip className="h-4 w-4 text-muted-foreground" />
                  <div>
                    <p className="font-medium">{widget.title}</p>
                    <p className="typography-body-medium text-muted-foreground">
                      {widget.description}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <Label htmlFor={`widget-${widget.id}`} className="typography-body-medium">
                    Hiển thị
                  </Label>
                  <Switch
                    id={`widget-${widget.id}`}
                    checked={widget.visible}
                    onCheckedChange={() => toggleWidgetVisibility(widget.id)}
                  />
                </div>
              </div>
            ))}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={resetLayout}>
              <RotateCw className="h-4 w-4 mr-2" />
              Đặt lại mặc định
            </Button>
            <Button onClick={() => setShowManagerDialog(false)}>
              Đóng
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add Widget Dialog */}
      <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Thêm Widget Mới</DialogTitle>
            <DialogDescription>
              Chọn widget để thêm vào dashboard
            </DialogDescription>
          </DialogHeader>

          <div className="grid grid-cols-1 gap-4 py-4">
            {availableWidgetTypes
              .filter(type => !localWidgets.some(w => w.type === type.type))
              .map((type) => (
                <button
                  key={type.type}
                  onClick={() => addWidget(type.type)}
                  className="text-left p-4 border rounded-xl hover:bg-accent transition-colors"
                >
                  <h4 className="font-medium">{type.title}</h4>
                  <p className="typography-body-medium text-muted-foreground mt-1">
                    {type.description}
                  </p>
                </button>
              ))}
          </div>

          {availableWidgetTypes.filter(type => !localWidgets.some(w => w.type === type.type)).length === 0 && (
            <p className="text-center py-4 text-muted-foreground">
              Tất cả widget đã được thêm vào dashboard
            </p>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
};
