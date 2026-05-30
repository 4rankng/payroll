import { Button } from '@/components/ui/button';
import { Plus, Users } from 'lucide-react';
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
    <div className="text-center py-12">
      <Users className="mx-auto h-12 w-12 text-muted-foreground/50" />
      <h3 className="mt-4 typography-title-large">Không có nhân viên nào</h3>
      <p className="mt-2 typography-body-medium text-muted-foreground">
        Hãy thêm nhân viên đầu tiên để bắt đầu quản lý.
      </p>
      <Button onClick={onAddEmployee} className="mt-4" variant="default">
        <Plus className="mr-2 h-4 w-4" />
        Thêm nhân viên
      </Button>
    </div>
  );
};