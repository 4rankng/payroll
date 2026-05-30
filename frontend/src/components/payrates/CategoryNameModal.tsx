import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogClose,
} from '@/components/ui/dialog';
import { Plus, X } from 'lucide-react';

interface CategoryNameModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (categoryName: string) => void;
  title?: string;
  description?: string;
  placeholder?: string;
  initialValue?: string;
}

export function CategoryNameModal({
  open,
  onOpenChange,
  onConfirm,
  title = "Thêm danh mục mới",
  description = "Nhập tên cho danh mục mới",
  placeholder = "Tên danh mục",
  initialValue = ""
}: CategoryNameModalProps) {
  const [categoryName, setCategoryName] = useState(initialValue);

  const handleConfirm = () => {
    if (categoryName.trim()) {
      onConfirm(categoryName.trim());
      setCategoryName('');
      onOpenChange(false);
    }
  };

  const handleCancel = () => {
    setCategoryName(initialValue);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
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
          <DialogTitle className="flex items-center gap-2">
            <Plus className="w-5 h-5 text-white/80" />
            {title}
          </DialogTitle>
          <DialogDescription>
            {description}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="category-name">Tên danh mục</Label>
            <Input
              id="category-name"
              value={categoryName}
              onChange={(e) => setCategoryName(e.target.value)}
              placeholder={placeholder}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  handleConfirm();
                }
                if (e.key === 'Escape') {
                  e.preventDefault();
                  handleCancel();
                }
              }}
              autoFocus
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>
            Đóng
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={!categoryName.trim()}
            className="flex items-center gap-2"
            variant="default"
          >
            <Plus className="w-4 h-4" />
            Thêm
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
