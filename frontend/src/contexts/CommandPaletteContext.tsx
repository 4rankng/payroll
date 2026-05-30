import React, { createContext, useContext, useState, useCallback, useEffect, useMemo } from 'react';
import { useAuth } from './AuthContext';
import { useUserPreferences } from './UserPreferencesContext';
import { useNavigate } from 'react-router-dom';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

interface Command {
  id: string;
  label: string;
  description?: string;
  icon: string;
  category: 'navigation' | 'action' | 'search';
  keywords: string[];
  action: () => void | Promise<void>;
  role?: 'admin' | 'partner' | 'both';
}

interface SearchResult {
  id: string;
  type: 'employee' | 'project' | 'timesheet' | 'command';
  title: string;
  subtitle?: string;
  action: () => void;
}

interface CommandPaletteContextValue {
  isOpen: boolean;
  query: string;
  searchResults: SearchResult[];
  isSearching: boolean;

  // Actions
  open: () => void;
  close: () => void;
  toggle: () => void;
  setQuery: (query: string) => void;
  executeCommand: (commandId: string) => void;

  // Available commands based on user role
  availableCommands: Command[];
}

const CommandPaletteContext = createContext<CommandPaletteContextValue | undefined>(undefined);

interface CommandPaletteProviderProps {
  children: React.ReactNode;
}

export const CommandPaletteProvider = ({ children }: CommandPaletteProviderProps) => {
  const { user } = useAuth();
  const { addRecentAction, addRecentSearch } = useUserPreferences();
  const navigate = useNavigate();

  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);

  // Define available commands based on user role
  const getAvailableCommands = useCallback((): Command[] => {
    if (!user) return [];
    const baseCommands: Command[] = [
      // Navigation commands
      {
        id: 'nav-dashboard',
        label: 'Đi đến Tổng quan',
        description: 'Xem trang tổng quan chính',
        icon: 'Home',
        category: 'navigation',
        keywords: ['dashboard', 'tổng quan', 'home'],
        action: () => navigate(user?.role === 'admin' ? '/admin' : '/partner'),
        role: 'both'
      },
      {
        id: 'nav-help',
        label: 'Trợ giúp',
        description: 'Xem tài liệu trợ giúp',
        icon: 'HelpCircle',
        category: 'navigation',
        keywords: ['help', 'trợ giúp', 'hướng dẫn'],
        action: () => navigate(user?.role === 'admin' ? '/admin/help' : '/partner/help'),
        role: 'both'
      }
    ];

    const adminCommands: Command[] = [
      // Admin navigation
      {
        id: 'nav-users',
        label: 'Người dùng',
        description: 'Xem danh sách người dùng',
        icon: 'Users',
        category: 'navigation',
        keywords: ['users', 'người dùng', 'user'],
        action: () => navigate('/admin/users'),
        role: 'admin'
      },
      {
        id: 'nav-projects',
        label: 'Dự án',
        description: 'Xem danh sách dự án',
        icon: 'FolderOpen',
        category: 'navigation',
        keywords: ['projects', 'dự án', 'project'],
        action: () => navigate('/admin/projects'),
        role: 'admin'
      },
      {
        id: 'nav-employees',
        label: 'Nhân viên',
        description: 'Xem danh sách nhân viên',
        icon: 'UserCheck',
        category: 'navigation',
        keywords: ['employees', 'nhân viên', 'employee'],
        action: () => navigate('/admin/employees'),
        role: 'admin'
      },
      {
        id: 'nav-timesheet',
        label: 'Bảng công',
        description: 'Bảng công',
        icon: 'Clock',
        category: 'navigation',
        keywords: ['timesheet', 'bảng công', 'chấm công'],
        action: () => navigate('/admin/timesheet'),
        role: 'admin'
      },
      {
        id: 'nav-approvals',
        label: 'Phê duyệt',
        description: 'Xử lý các yêu cầu phê duyệt',
        icon: 'CheckSquare',
        category: 'navigation',
        keywords: ['approvals', 'phê duyệt', 'duyệt'],
        action: () => navigate('/admin/approvals'),
        role: 'admin'
      },
      // Admin actions
      {
        id: 'action-add-employee',
        label: 'Thêm nhân viên mới',
        description: 'Tạo hồ sơ nhân viên mới',
        icon: 'UserPlus',
        category: 'action',
        keywords: ['add employee', 'thêm nhân viên', 'tạo nhân viên'],
        action: () => navigate('/admin/employees?modal=add_employee'),
        role: 'admin'
      },
      {
        id: 'action-add-project',
        label: 'Tạo dự án mới',
        description: 'Khởi tạo dự án mới',
        icon: 'FolderPlus',
        category: 'action',
        keywords: ['add project', 'tạo dự án', 'thêm dự án'],
        action: () => navigate('/admin/projects?modal=project_create'),
        role: 'admin'
      },
    ];

    const partnerCommands: Command[] = [
      // Partner navigation
      {
        id: 'nav-partner-projects',
        label: 'Dự án',
        description: 'Xem các dự án',
        icon: 'FolderOpen',
        category: 'navigation',
        keywords: ['projects', 'dự án'],
        action: () => navigate('/partner'),
        role: 'partner'
      },
      {
        id: 'nav-partner-employees',
        label: 'Nhân viên dự án',
        description: 'Xem nhân viên trong dự án',
        icon: 'Users',
        category: 'navigation',
        keywords: ['employees', 'nhân viên', 'team'],
        action: () => navigate('/partner/employees'),
        role: 'partner'
      },
      {
        id: 'nav-partner-timesheet',
        label: 'Bảng công',
        description: 'Bảng công cá nhân',
        icon: 'Clock',
        category: 'navigation',
        keywords: ['timesheet', 'bảng công', 'chấm công'],
        action: () => navigate('/partner/timesheet'),
        role: 'partner'
      },
      // Partner actions
      {
        id: 'action-submit-timesheet',
        label: 'Nộp bảng công',
        description: 'Gửi bảng công để phê duyệt',
        icon: 'Send',
        category: 'action',
        keywords: ['submit timesheet', 'nộp bảng công', 'gửi bảng công'],
        action: () => navigate('/partner/timesheet?action=submit'),
        role: 'partner'
      }
    ];

    // Filter commands based on user role
    const allCommands = [...baseCommands, ...adminCommands, ...partnerCommands];
    return allCommands.filter(cmd =>
      cmd.role === 'both' || cmd.role === user?.role
    );
  }, [user, navigate]);

  const availableCommands = useMemo(() => getAvailableCommands(), [getAvailableCommands]);

  // Handle search
  useEffect(() => {
    if (!user || !query.trim()) {
      setSearchResults([]);
      setIsSearching(false);
      return;
    }

    setIsSearching(true);

    // Debounced search
    const timeoutId = setTimeout(() => {
      const searchTerm = query;

      // Search through available commands
      const commandResults = availableCommands
        .filter(cmd =>
          vietnameseIncludes(cmd.label, searchTerm) ||
          (cmd.description && vietnameseIncludes(cmd.description, searchTerm)) ||
          cmd.keywords.some(keyword => vietnameseIncludes(keyword, searchTerm))
        )
        .slice(0, 5)
        .map(cmd => ({
          id: cmd.id,
          type: 'command' as const,
          title: cmd.label,
          subtitle: cmd.description,
          action: cmd.action
        }));

      setSearchResults(commandResults);
      setIsSearching(false);
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [query, availableCommands, user]);

  // Keyboard shortcuts
  useEffect(() => {
    if (!user) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      // Cmd+K or Ctrl+K to open
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        setIsOpen(true);
      }

      // Escape to close
      if (event.key === 'Escape' && isOpen) {
        setIsOpen(false);
        setQuery('');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, user]);

  const open = useCallback(() => {
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
    setQuery('');
  }, []);

  const toggle = useCallback(() => {
    setIsOpen(prev => !prev);
    if (isOpen) {
      setQuery('');
    }
  }, [isOpen]);

  const executeCommand = useCallback(async (commandId: string) => {
    if (!user) return;

    const command = availableCommands.find(cmd => cmd.id === commandId);
    if (!command) return;

    try {
      await command.action();

      // Track action usage
      await addRecentAction({
        type: 'command',
        label: command.label,
        icon: command.icon,
        handler: commandId
      });

      // Add to recent searches if there was a query
      if (query.trim()) {
        await addRecentSearch(query.trim());
      }

      close();
    } catch (error) {
      console.error('Failed to execute command:', error);
    }
  }, [availableCommands, addRecentAction, addRecentSearch, query, close, user]);

  const value: CommandPaletteContextValue = useMemo(() => ({
    isOpen: user ? isOpen : false,
    query: user ? query : '',
    searchResults: user ? searchResults : [],
    isSearching: user ? isSearching : false,
    open: user ? open : () => {},
    close: user ? close : () => {},
    toggle: user ? toggle : () => {},
    setQuery: user ? setQuery : () => {},
    executeCommand: user ? executeCommand : () => {},
    availableCommands: user ? availableCommands : []
  }), [
    isOpen,
    query,
    searchResults,
    isSearching,
    open,
    close,
    toggle,
    setQuery,
    executeCommand,
    availableCommands,
    user
  ]);

  return (
    <CommandPaletteContext.Provider value={value}>
      {children}
    </CommandPaletteContext.Provider>
  );
};

export const useCommandPalette = (): CommandPaletteContextValue => {
  const context = useContext(CommandPaletteContext);
  if (context === undefined) {
    throw new Error('useCommandPalette must be used within a CommandPaletteProvider');
  }
  return context;
};
