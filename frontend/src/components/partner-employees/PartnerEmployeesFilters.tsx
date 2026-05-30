import { SearchBar } from '@/components/shared/SearchBar';

interface PartnerEmployeesFiltersProps {
  searchTerm: string;
  onSearch: (term: string) => void;
  onClearSearch: () => void;
}

export const PartnerEmployeesFilters = ({
  searchTerm,
  onSearch,
}: PartnerEmployeesFiltersProps) => {
  return (
    <SearchBar
      searchTerm={searchTerm}
      onSearchChange={onSearch}
      placeholder="Tìm nhân viên..."
      className="w-48"
    />
  );
};
