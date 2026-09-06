import { ResponsiveTable } from '@/components/ui/responsive-table';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { EmployeeEmptyStates, EmployeeTableEmptyState } from './EmployeeEmptyStates';
import { EmployeeMobileCard } from './EmployeeMobileCard';
import type { ColumnDef, SortingState, OnChangeFn } from '@tanstack/react-table';
import type { MobileField, RowAction } from '@/components/ui/mobile-table';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeListContentProps {
  filteredEmployees: Employee[];
  dataToDisplay: Employee[];
  columns: ColumnDef<Employee>[];
  mobileFields: MobileField<Employee>[];
  rowTitle: (row: Employee) => React.ReactNode;
  rowSubtitle: (row: Employee) => React.ReactNode;
  rowActions: RowAction<Employee>[];
  searchTerm: string;
  isMobile: boolean;
  useCardView?: boolean;
  hasMore: boolean;
  isLoadingMore: boolean;
  loadMore: () => void;
  onRowClick: (employee: Employee) => void;
  onClearSearch: () => void;
  onAddEmployee: () => void;
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
  /** Render flush inside a parent operations card (no inner card wrapper). */
  embedded?: boolean;
}

export const EmployeeListContent = ({
  filteredEmployees,
  dataToDisplay,
  columns,
  mobileFields,
  rowTitle,
  rowSubtitle,
  rowActions,
  searchTerm,
  isMobile,
  useCardView = false,
  hasMore,
  isLoadingMore,
  loadMore,
  onRowClick,
  onClearSearch,
  onAddEmployee,
  sorting,
  onSortingChange,
  embedded = false,
}: EmployeeListContentProps) => {
  return (
    <div>
      {filteredEmployees.length === 0 ? (
        <div className="p-6">
          <EmployeeEmptyStates
            searchTerm={searchTerm}
            onClearSearch={onClearSearch}
            onAddEmployee={onAddEmployee}
          />
        </div>
      ) : isMobile && useCardView ? (
        <InfiniteScrollContainer
          onLoadMore={loadMore}
          hasMore={hasMore}
          isLoading={isLoadingMore}
          className="px-4 sm:px-0"
        >
          <div className="space-y-3">
            {dataToDisplay.map((employee) => (
              <EmployeeMobileCard
                key={employee.id}
                employee={employee}
                onClick={onRowClick}
              />
            ))}
          </div>
        </InfiniteScrollContainer>
      ) : isMobile ? (
        <InfiniteScrollContainer
          onLoadMore={loadMore}
          hasMore={hasMore}
          isLoading={isLoadingMore}
          className="px-4 sm:px-0"
        >
          <ResponsiveTable
            data={dataToDisplay}
            columns={columns}
            searchKey="name"
            searchPlaceholder="Tìm kiếm nhân viên..."
            mobileFields={mobileFields}
            rowTitle={rowTitle}
            rowSubtitle={rowSubtitle}
            rowActions={rowActions}
            getRowId={(row) => String(row.id)}
            onRowClick={onRowClick}
            emptyState={<EmployeeTableEmptyState onAddEmployee={onAddEmployee} />}
            sorting={sorting}
            onSortingChange={onSortingChange}
          />
        </InfiniteScrollContainer>
      ) : (
        <ResponsiveTable
          data={dataToDisplay}
          columns={columns}
          searchKey="name"
          searchPlaceholder="Tìm kiếm nhân viên..."
          mobileFields={mobileFields}
          rowTitle={rowTitle}
          rowSubtitle={rowSubtitle}
          rowActions={rowActions}
          getRowId={(row) => String(row.id)}
          onRowClick={onRowClick}
          emptyState={<EmployeeTableEmptyState onAddEmployee={onAddEmployee} />}
          className={embedded ? undefined : 'px-4 sm:px-0'}
          sorting={sorting}
          onSortingChange={onSortingChange}
          embedded={embedded}
        />
      )}
    </div>
  );
};