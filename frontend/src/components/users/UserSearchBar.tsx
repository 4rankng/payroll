import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useDebouncedInput } from "@/hooks/useDebouncedInput";

interface UserSearchBarProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  placeholder?: string;
}

export const UserSearchBar = ({
  searchTerm,
  onSearchChange,
  placeholder = "Tìm kiếm theo tên, email hoặc vai trò...",
}: UserSearchBarProps) => {
  const { localValue, handleChange } = useDebouncedInput(searchTerm, onSearchChange);

  return (
    <div className="relative max-w-md mb-6">
      <Search className="absolute left-3 top-3 h-4 w-4 text-gray-500" />
      <Input
        placeholder={placeholder}
        value={localValue}
        onChange={handleChange}
        className="pl-9"
      />
    </div>
  );
};
