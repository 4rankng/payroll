import { useState, useMemo, useCallback, useEffect } from 'react';
import { User } from '@/types/user';
import { useUsersInfinite } from '@/hooks/api/useUsers';
import { useDebounce } from '@/hooks/useDebounce';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Loader2, Search } from 'lucide-react';

interface UserSelectorProps {
  value?: string;
  onValueChange?: (value: string) => void;
  // Optional callback to expose selected user object
  onUserChange?: (user: User | null) => void;
  placeholder?: string;
  className?: string;
  excludeUserIds?: number[];
  filterRole?: 'admin' | 'partner' | 'employee';
}

export function UserSelector({
  value,
  onValueChange,
  onUserChange,
  placeholder = "Chọn người dùng...",
  className,
  excludeUserIds = [],
  filterRole,
}: UserSelectorProps) {
  const [search, setSearch] = useState('');
  const [isOpen, setIsOpen] = useState(false);

  // Debounce search input to avoid excessive API calls
  const debouncedSearch = useDebounce(search, 300);

  // Track if we're waiting for debounced search
  const isSearching = search !== debouncedSearch;

  // Clear search when select closes
  useEffect(() => {
    if (!isOpen) {
      setSearch('');
    }
  }, [isOpen]);

  // Get users list with API filtering and infinite scroll support
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    isLoading,
  } = useUsersInfinite({
    pageSize: 20,
    role: filterRole,
    search: debouncedSearch.trim() || undefined,
  });

  // Flatten all pages into a single array
  const allUsers = useMemo(() => {
    if (!data?.pages) return [];
    return data.pages.flatMap((page) => page.data);
  }, [data?.pages]);

  // Filter users based on exclude list only (search and role filtering is done by API)
  const filteredUsers = useMemo(() => {
    return allUsers.filter((user: User) => !excludeUserIds.includes(user.id));
  }, [allUsers, excludeUserIds]);

  // Get selected user for display
  const selectedUser = useMemo(() => {
    if (!value) return null;
    return allUsers.find((user: User) => user.id.toString() === value) || null;
  }, [value, allUsers]);

  // Handle scroll to load more
  const handleScroll = useCallback(
    (e: React.UIEvent<HTMLDivElement>) => {
      const target = e.target as HTMLDivElement;
      const scrolledToBottom =
        target.scrollHeight - target.scrollTop <= target.clientHeight + 50; // 50px threshold

      if (scrolledToBottom && hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
      }
    },
    [hasNextPage, isFetchingNextPage, fetchNextPage]
  );

  if (isLoading) {
    return (
      <div className="flex h-10 w-full items-center justify-center rounded-md border border-input bg-background px-3 py-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="ml-2 typography-body-medium text-muted-foreground">
          Đang tải...
        </span>
      </div>
    );
  }

  const handleValueChange = (val: string) => {
    if (onValueChange) onValueChange(val);
    if (onUserChange) {
      const user = allUsers.find((u: User) => u.id.toString() === val) || null;
      onUserChange(user);
    }
  };

  return (
    <Select value={value} onValueChange={handleValueChange} open={isOpen} onOpenChange={setIsOpen}>
      <SelectTrigger className={className} aria-label={placeholder}>
        <SelectValue placeholder={placeholder}>
          {selectedUser ? (
            <div className="flex items-center gap-2">
              <div className="flex flex-col items-start">
                <span className="typography-body-medium">{selectedUser.fullname}</span>
                <span className="typography-caption text-muted-foreground">
                  {selectedUser.email}
                </span>
              </div>
            </div>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent onScroll={handleScroll}>
        <div className="p-2 border-b sticky top-0 bg-background z-10">
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
            <Input
              aria-label="Tìm kiếm người dùng"
              placeholder="Tìm kiếm người dùng..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                // Prevent Select from closing when typing
                e.stopPropagation();
              }}
              onClick={(e) => {
                // Prevent Select from closing when clicking input
                e.stopPropagation();
              }}
              onMouseDown={(e) => {
                // Prevent Select from closing when clicking input
                e.stopPropagation();
              }}
              onFocus={(e) => {
                // Prevent Select from closing when focusing input
                e.stopPropagation();
              }}
              className="h-9 pl-8 pr-8"
              autoComplete="off"
              autoFocus
            />
            {isSearching && (
              <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 h-4 w-4 animate-spin text-muted-foreground pointer-events-none" />
            )}
          </div>
        </div>

        {filteredUsers.length === 0 && !isLoading ? (
          <div className="p-2 text-center typography-body-medium text-muted-foreground">
            {search.trim() ? 'Không tìm thấy người dùng phù hợp' : 'Không có người dùng khả dụng'}
          </div>
        ) : (
          <>
            {filteredUsers.map((user: User) => (
              <SelectItem key={user.id} value={user.id.toString()}>
                <div className="flex flex-col items-start w-full">
                  <span className="typography-body-medium">{user.fullname}</span>
                  <span className="typography-caption text-muted-foreground">
                    {user.email} • {
                      user.role === 'admin' ? 'Quản trị viên' :
                      user.role === 'partner' ? 'Quản lý' :
                      user.role === 'adv_partner' ? 'Quản lý ứng lương' :
                      user.role === 'accountant' ? 'Kế toán' :
                      'Nhân viên'
                    }
                  </span>
                </div>
              </SelectItem>
            ))}
            {isFetchingNextPage && (
              <div className="flex items-center justify-center p-2 typography-body-small text-muted-foreground">
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang tải thêm...
              </div>
            )}
          </>
        )}
      </SelectContent>
    </Select>
  );
}
