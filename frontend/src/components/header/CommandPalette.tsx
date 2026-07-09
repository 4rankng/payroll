import { useEffect, useRef, useState } from 'react';
import { Search, Command as CommandIcon } from 'lucide-react';
import {
  Dialog,
  DialogContent,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCommandPalette } from '@/contexts';
import { cn } from '@/lib/utils';

// Dynamic icon imports
const iconMap: Record<string, () => Promise<React.ComponentType<{ className?: string }>>> = {
  Home: () => import('lucide-react').then(mod => mod.Home),
  Users: () => import('lucide-react').then(mod => mod.Users),
  FolderOpen: () => import('lucide-react').then(mod => mod.FolderOpen),
  UserCheck: () => import('lucide-react').then(mod => mod.UserCheck),
  Clock: () => import('lucide-react').then(mod => mod.Clock),
  DollarSign: () => import('lucide-react').then(mod => mod.DollarSign),
  CheckSquare: () => import('lucide-react').then(mod => mod.CheckSquare),
  BarChart3: () => import('lucide-react').then(mod => mod.BarChart3),
  Settings: () => import('lucide-react').then(mod => mod.Settings),
  HelpCircle: () => import('lucide-react').then(mod => mod.HelpCircle),
  UserPlus: () => import('lucide-react').then(mod => mod.UserPlus),
  FolderPlus: () => import('lucide-react').then(mod => mod.FolderPlus),
  Calculator: () => import('lucide-react').then(mod => mod.Calculator),
  Send: () => import('lucide-react').then(mod => mod.Send),
};

interface CommandItemProps {
  id: string;
  label: string;
  description?: string;
  icon: string;
  isSelected: boolean;
  onSelect: () => void;
}

const CommandItem = ({ label, description, icon, isSelected, onSelect }: CommandItemProps) => {
  const [IconComponent, setIconComponent] = useState<React.ComponentType<{ className?: string }> | null>(null);

  useEffect(() => {
    const loadIcon = async () => {
      try {
        const iconLoader = iconMap[icon];
        if (iconLoader) {
        const iconModule = await iconLoader();
        setIconComponent(() => iconModule);
        }
      } catch (error) {
        console.warn(`Failed to load icon: ${icon}`);
        // Fallback to CommandIcon
        setIconComponent(() => CommandIcon);
      }
    };

    loadIcon();
  }, [icon]);

  return (
    <button
      className={cn(
        "flex w-full items-center gap-3 px-3 py-2 text-left typography-body-medium transition-colors",
        "hover:bg-accent hover:text-accent-foreground",
        isSelected && "bg-accent text-accent-foreground"
      )}
      onClick={onSelect}
    >
      {IconComponent && (
        <IconComponent className="h-4 w-4 flex-shrink-0" />
      )}
      <div className="flex-1 min-w-0">
        <div className="font-medium truncate">{label}</div>
        {description && (
          <div className="typography-body-small text-muted-foreground truncate">
            {description}
          </div>
        )}
      </div>
    </button>
  );
};

export const CommandPalette = () => {
  const {
    isOpen,
    query,
    searchResults,
    isSearching,
    close,
    setQuery,
    executeCommand,
    availableCommands
  } = useCommandPalette();

  const inputRef = useRef<HTMLInputElement>(null);
  const [selectedIndex, setSelectedIndex] = useState(0);

  // Focus input when dialog opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 0);
      setSelectedIndex(0);
    }
  }, [isOpen]);

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [searchResults]);

  // Keyboard navigation
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const results = query.trim() ? searchResults : availableCommands.slice(0, 8);

      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex(prev => (prev + 1) % results.length);
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex(prev => prev === 0 ? results.length - 1 : prev - 1);
          break;
        case 'Enter':
          e.preventDefault();
          if (results[selectedIndex]) {
            const selectedResult = results[selectedIndex];
            if (selectedResult && 'action' in selectedResult) {
              selectedResult.action();
              close();
            }
          }
          break;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, query, searchResults, availableCommands, selectedIndex, executeCommand, close]);

  const displayResults = query.trim()
    ? searchResults
    : availableCommands.slice(0, 8).map(cmd => ({
        id: cmd.id,
        type: 'command' as const,
        title: cmd.label,
        subtitle: cmd.description,
        action: () => executeCommand(cmd.id)
      }));

  return (
    <Dialog open={isOpen} onOpenChange={close}>
      <DialogContent className="max-w-2xl" contentPadding="none" hideCloseButton>
        <div className="flex flex-col">
          {/* Search Input */}
          <div className="flex items-center border-b px-3">
            <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
            <Input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Tìm kiếm lệnh hoặc điều hướng..."
              className="border-0 focus-visible:ring-0 focus-visible:ring-offset-0"
            />
          </div>

          {/* Results */}
          <ScrollArea className="max-h-96">
            <div className="p-2">
              {!query.trim() && (
                <div className="px-2 py-1.5 typography-body-small text-muted-foreground">
                  Lệnh phổ biến
                </div>
              )}

              {displayResults.length > 0 ? (
                <div className="space-y-1">
                  {displayResults.map((result, index) => {
                    const command = availableCommands.find(cmd => cmd.id === result.id);

                    return (
                      <CommandItem
                        key={result.id}
                        id={result.id}
                        label={result.title}
                        description={result.subtitle}
                        icon={command?.icon || 'CommandIcon'}
                        isSelected={index === selectedIndex}
                        onSelect={result.action}
                      />
                    );
                  })}
                </div>
              ) : query.trim() && !isSearching ? (
                <div className="py-6 text-center typography-body-medium text-muted-foreground">
                  Không tìm thấy kết quả cho "{query}"
                </div>
              ) : isSearching ? (
                <div className="py-6 text-center typography-body-medium text-muted-foreground">
                  Đang tìm kiếm...
                </div>
              ) : null}
            </div>
          </ScrollArea>

          {/* Footer */}
          <div className="flex items-center justify-between border-t px-3 py-2 typography-body-small text-muted-foreground">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-1">
                <kbd className="h-4 w-4 rounded border bg-muted flex items-center justify-center">
                  ↑
                </kbd>
                <kbd className="h-4 w-4 rounded border bg-muted flex items-center justify-center">
                  ↓
                </kbd>
                <span>điều hướng</span>
              </div>
              <div className="flex items-center gap-1">
                <kbd className="px-1 h-4 rounded border bg-muted flex items-center justify-center">
                  ↵
                </kbd>
                <span>chọn</span>
              </div>
            </div>
            <div className="flex items-center gap-1">
              <kbd className="px-1 h-4 rounded border bg-muted flex items-center justify-center">
                esc
              </kbd>
              <span>đóng</span>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
