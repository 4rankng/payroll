import { useCallback, useEffect, useMemo, useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useUsersByIds, useUsersInfinite } from '@/hooks/api/useUsers';
import { useDebounce } from '@/hooks/useDebounce';
import { cn } from '@/lib/utils';
import { Check, ChevronsUpDown, Loader2, Users, X } from 'lucide-react';
import type { User } from '@/types/user';

interface UserMultiSelectorProps {
  value?: string[];
  onValueChange: (values: string[]) => void;
  placeholder?: string;
  className?: string;
  excludeUserIds?: number[];
  disabled?: boolean;
}

const getRoleLabel = (role: User['role']) => {
  if (role === 'admin') return 'Quản trị viên';
  if (role === 'partner') return 'Đối tác';
  if (role === 'adv_partner') return 'Quản lý ứng lương';
  if (role === 'accountant') return 'Kế toán';
  return 'Nhân viên';
};

const buildSelectedUsers = (
  selectedIds: number[],
  userMap?: Record<number, User>
) => {
  return selectedIds.map((id) => {
    const user = userMap?.[id];
    if (user) return user;
    return { id, fullname: `Người dùng #${id}`, email: '', username: '', role: 'employee', created_at: '', updated_at: '' };
  });
};

export function UserMultiSelector({
  value = [],
  onValueChange,
  placeholder = 'Chọn người nhận...',
  className,
  excludeUserIds = [],
  disabled = false,
}: UserMultiSelectorProps) {
  const [open, setOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const debouncedSearch = useDebounce(searchValue, 300);

  const normalizedSelectedIds = useMemo(() => {
    const ids = value
      .map((id) => Number(id))
      .filter((parsed): parsed is number => !Number.isNaN(parsed));
    return Array.from(new Set(ids));
  }, [value]);

  const { data: selectedUsersById } = useUsersByIds(normalizedSelectedIds, {
    enabled: normalizedSelectedIds.length > 0,
  });

  const usersQuery = useUsersInfinite({
    pageSize: 20,
    search: debouncedSearch.trim() ? debouncedSearch.trim() : undefined,
  });

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = usersQuery;

  const allUsers = useMemo(() => {
    if (!data?.pages) return [];
    return data.pages.flatMap((page) => page.data);
  }, [data?.pages]);

  const filteredUsers = useMemo(() => {
    if (excludeUserIds.length === 0) return allUsers;
    const excluded = new Set(excludeUserIds);
    return allUsers.filter((user) => !excluded.has(user.id));
  }, [allUsers, excludeUserIds]);

  const selectedUsersDetails = useMemo(() => {
    return buildSelectedUsers(normalizedSelectedIds, selectedUsersById);
  }, [normalizedSelectedIds, selectedUsersById]);

  const selectedValueSet = useMemo(() => new Set(value), [value]);

  const triggerLabel = useMemo(() => {
    if (normalizedSelectedIds.length === 0) {
      return placeholder;
    }
    const firstUser = selectedUsersDetails[0];
    if (!firstUser) {
      return `Đã chọn ${normalizedSelectedIds.length} người`;
    }
    if (normalizedSelectedIds.length === 1) {
      return firstUser.fullname;
    }
    return `${firstUser.fullname} và ${normalizedSelectedIds.length - 1} người khác`;
  }, [normalizedSelectedIds.length, selectedUsersDetails, placeholder]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    setOpen(nextOpen);
  }, []);

  useEffect(() => {
    if (!open) {
      setSearchValue('');
    }
  }, [open]);

  const handleSearchValueChange = useCallback((nextValue: string) => {
    setSearchValue(nextValue);
  }, []);

  const handleScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const target = event.currentTarget;
      if (target.scrollHeight - target.scrollTop - target.clientHeight < 50) {
        if (hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      }
    },
    [fetchNextPage, hasNextPage, isFetchingNextPage]
  );

  const updateSelection = useCallback(
    (nextValues: string[]) => {
      onValueChange(nextValues);
    },
    [onValueChange]
  );

  const toggleUserSelection = useCallback(
    (userId: number) => {
      const stringId = userId.toString();
      if (selectedValueSet.has(stringId)) {
        updateSelection(value.filter((current) => current !== stringId));
        return;
      }
      updateSelection([...value, stringId]);
    },
    [selectedValueSet, updateSelection, value]
  );

  const removeSelectedUser = useCallback(
    (userId: number) => {
      const stringId = userId.toString();
      updateSelection(value.filter((current) => current !== stringId));
    },
    [updateSelection, value]
  );

  const handleRemoveSelectedUserClick = useCallback(
    (event: React.MouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      const target = event.currentTarget;
      const userId = target.dataset.userId;
      if (!userId) return;
      const parsedId = Number(userId);
      if (Number.isNaN(parsedId)) return;
      removeSelectedUser(parsedId);
    },
    [removeSelectedUser]
  );

  const handleCommandItemSelect = useCallback(
    (itemValue: string) => {
      const parsedId = Number(itemValue);
      if (Number.isNaN(parsedId)) return;
      toggleUserSelection(parsedId);
    },
    [toggleUserSelection]
  );

  const isSearching = searchValue !== debouncedSearch;

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            'min-h-11 w-full justify-between rounded-md border border-input bg-background px-3 py-2 text-left',
            className
          )}
          disabled={disabled}
          role="combobox"
          aria-expanded={open}
          aria-label="Chọn người nhận"
        >
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex min-w-0 items-center gap-2">
              <Users className="h-4 w-4 shrink-0 opacity-60" />
              <span className="min-w-0 truncate typography-body-medium">{triggerLabel}</span>
            </div>
            {normalizedSelectedIds.length > 0 && (
              <span className="min-w-0 truncate typography-caption text-muted-foreground">
                {normalizedSelectedIds.length === 1
                  ? 'Một người đã được chọn'
                  : `${normalizedSelectedIds.length} người được chọn`}
              </span>
            )}
          </div>
          <ChevronsUpDown className="h-4 w-4 opacity-60" />
        </Button>
      </PopoverTrigger>

      <PopoverContent
        align="start"
        side="bottom"
        className="w-[calc(100vw-2rem)] max-w-[480px] p-0 sm:w-[--radix-popover-trigger-width]"
      >
        <div className="flex flex-col gap-3">
          <div className="space-y-2 border-b px-4 pt-4 pb-2">
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4 text-muted-foreground" />
              <span className="typography-body-medium text-muted-foreground">
                {normalizedSelectedIds.length === 0
                  ? 'Chưa có người nhận nào'
                  : 'Người nhận đã chọn'}
              </span>
            </div>
            {normalizedSelectedIds.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {selectedUsersDetails.map((user) => (
                  <Badge
                    key={user.id}
                    variant="secondary"
                    className="inline-flex min-h-9 items-center gap-1.5 whitespace-normal rounded-full py-1"
                  >
                    <span className="break-words typography-body-small">{user.fullname}</span>
                    <button
                      type="button"
                      data-user-id={user.id}
                      onClick={handleRemoveSelectedUserClick}
                      className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground transition hover:bg-muted/80"
                      aria-label="Xóa người nhận"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            )}
          </div>

          <Command shouldFilter={false}>
            <CommandInput
              aria-label="Tìm kiếm người nhận"
              value={searchValue}
              onValueChange={handleSearchValueChange}
              placeholder="Tìm kiếm người dùng..."
            />
            <CommandList className="space-y-2 px-2" onScroll={handleScroll}>
              {isSearching && (
                <div className="flex items-center gap-2 px-2 py-2 typography-body-small text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang tìm kiếm người dùng...
                </div>
              )}

              {isLoading && !isSearching && (
                <div className="flex items-center gap-2 px-2 py-2 typography-body-small text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang tải người dùng đang hoạt động...
                </div>
              )}

              {!isLoading && filteredUsers.length === 0 && (
                <CommandEmpty>
                  {searchValue.trim()
                    ? 'Không tìm thấy người dùng phù hợp'
                    : 'Không có người dùng khả dụng'}
                </CommandEmpty>
              )}

              {!isLoading && filteredUsers.length > 0 && (
                <CommandGroup>
                  {filteredUsers.map((user) => {
                    const isSelected = selectedValueSet.has(user.id.toString());
                    return (
                      <CommandItem
                        key={user.id}
                        value={user.id.toString()}
                        onSelect={handleCommandItemSelect}
                        className="min-h-12 items-start py-2.5"
                      >
                        <Check
                          className={cn(
                            'mr-2 h-4 w-4 transition-opacity',
                            isSelected ? 'opacity-100' : 'opacity-0'
                          )}
                        />
                        <div className="flex min-w-0 flex-col items-start leading-tight">
                          <span className="break-words typography-body-medium">{user.fullname}</span>
                          <span className="break-words typography-caption text-muted-foreground">
                            {user.email} • {getRoleLabel(user.role)}
                          </span>
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              )}

              {isFetchingNextPage && (
                <div className="flex items-center gap-2 px-2 py-3 typography-body-small text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang tải thêm người dùng...
                </div>
              )}
            </CommandList>
          </Command>
        </div>
      </PopoverContent>
    </Popover>
  );
}
