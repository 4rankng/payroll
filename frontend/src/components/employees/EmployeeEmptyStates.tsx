import { Button } from '@/components/ui/button';
import { Plus } from 'lucide-react';
import { EmptyState } from '@/components/shared/EmptyState';
import { NoEmployeesFound, NoSearchResults } from '@/components/ui/loading-states';

interface EmployeeEmptyStatesProps {
  searchTerm: string;
  onClearSearch: () => void;
  onAddEmployee: () => void;
}

export const EmployeeEmptyStates = ({
  searchTerm,
  onClearSearch,
  onAddEmployee,
}: EmployeeEmptyStatesProps) => {
  if (searchTerm) {
    return (
      <NoSearchResults
        query={searchTerm}
        onClearSearch={onClearSearch}
      />
    );
  }

  return <NoEmployeesFound onAddEmployee={onAddEmployee} />;
};

export const EmployeeTableEmptyState = ({ onAddEmployee }: { onAddEmployee: () => void }) => {
  return (
    <EmptyState
      title="Không có nhân viên nào"
      description="Hãy thêm nhân viên đầu tiên để bắt đầu quản lý."
      className="py-8"
    >
      <Button onClick={onAddEmployee} variant="default">
        <Plus className="mr-2 h-4 w-4" />
        Thêm nhân viên
      </Button>
    </EmptyState>
  );
};
