import { useState, useMemo, useEffect, useCallback } from 'react';
import { ColumnDef } from '@tanstack/react-table';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Card } from '@/components/ui/card';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { DataTable } from '@/components/ui/data-table';
import { AlertCircle, Info, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { PayrateStructure, DayType, DEFAULT_POSITIONS, COMMON_DAY_TYPES } from '@/types/api/payrate.types';
import { getAllHourTypes } from '@/components/payrates/types';
import { formatCurrency } from '@/utils/formatters';
import { VIETNAMESE_PAYRATE_LABELS } from '@/types/api/payrate.types';
import { getPositionBadgeStyle, getDayTypeBadgeStyle, getItemIndex } from '@/utils/badge-styles';
import { TimesheetDateNavigation } from './TimesheetDateNavigation';
import { useProjectModals } from '@/hooks/useModalNavigation';

interface Employee {
  id: number;
  fullname: string;
  cccd: string;
  position: string; // Default position from employee profile
  employee_code: string;
}

// New timesheet entry structure for 3-level payrate
interface TimesheetEntry {
  employee_id: number;
  position: string; // Selected position (can override employee default)
  hour_entries: Record<string, number>; // hour_type -> hours_worked
  existingEntries?: Array<{ // Store existing timesheet data for status display
    id: number;
    hour_type: string;
    status: 'draft' | 'pending_approval' | 'approved' | 'rejected';
    paymentStatus?: 'pending' | 'paid' | 'failed' | 'cancelled';
  }>;
}

// Row data structure for DataTable
interface TimesheetTableRow {
  id: number;
  employee_id: number;
  fullname: string;
  cccd: string;
  employee_code: string;
  position: string;
  hour_entries: Record<string, number>;
  totalHours: number;
  totalAmount: number;
  hourStatus: 'normal' | 'exceeded' | 'excessive';
  // Dynamic hour columns will be added at runtime
  [key: string]: unknown;
}


interface TimesheetEntryTableProps {
  employees: Employee[];
  entries: TimesheetEntry[];
  onChange: (entries: TimesheetEntry[]) => void;
  payrateConfig?: PayrateStructure;
  dayType: DayType;
  selectedDate: Date;
  onDateChange: (date: Date) => void;
  isLoading?: boolean;
  selectedProject?: { id: number; name: string; code: string } | null;
}

export function TimesheetEntryTable({
  employees,
  entries,
  onChange,
  payrateConfig,
  dayType,
  selectedDate,
  onDateChange,
  isLoading = false,
  selectedProject
}: TimesheetEntryTableProps) {
  // Project modals hook for navigation
  const { openProjectDetails } = useProjectModals();

  // Available hour types from payrate config
  const availableHourTypes = useMemo(() => {
    if (!payrateConfig) return [];
    return getAllHourTypes(payrateConfig);
  }, [payrateConfig]);

  // Get available positions from payrate config
  const availablePositions = useMemo(() => {
    if (!payrateConfig) return [];
    return Object.keys(payrateConfig);
  }, [payrateConfig]);



  const [localEntries, setLocalEntries] = useState<Record<number, TimesheetEntry>>(
    () => {
      const entryMap: Record<number, TimesheetEntry> = {};
      if (!employees || !Array.isArray(employees)) {
        return entryMap;
      }

      employees.forEach(emp => {
        const existingEntry = entries?.find(e => e.employee_id === emp.id);
        entryMap[emp.id] = existingEntry ? {
          ...existingEntry,
          hour_entries: existingEntry.hour_entries || {}
        } : {
          employee_id: emp.id,
          position: emp.position, // Default to employee's position
          hour_entries: {} // Empty hour entries
        };
      });
      return entryMap;
    }
  );

  // Update localEntries when employees or entries change
  useEffect(() => {
    if (!employees || !Array.isArray(employees)) return;

    const entryMap: Record<number, TimesheetEntry> = {};
    employees.forEach(emp => {
      const existingEntry = entries?.find(e => e.employee_id === emp.id);
      entryMap[emp.id] = existingEntry ? {
        ...existingEntry,
        hour_entries: existingEntry.hour_entries || {}
      } : {
        employee_id: emp.id,
        position: emp.position,
        hour_entries: {}
      };
    });
    setLocalEntries(entryMap);
  }, [employees, entries]);

  const updateEntry = useCallback((employeeId: number, field: 'hour_entries', value: Record<string, number>) => {
    const newEntries = {
      ...localEntries,
      [employeeId]: {
        ...localEntries[employeeId],
        [field]: value
      }
    };
    setLocalEntries(newEntries);
    onChange(Object.values(newEntries));
  }, [localEntries, onChange]);


  // Check if an employee has validation error

  const updateHourEntry = useCallback((employeeId: number, hourType: string, hours: number) => {
    const entry = localEntries[employeeId];
    if (!entry) return;

    const newHourEntries = {
      ...(entry.hour_entries || {}),
      [hourType]: hours
    };
    updateEntry(employeeId, 'hour_entries', newHourEntries);
  }, [localEntries, updateEntry]);

  // Calculate total hours for an employee
  const calculateTotalHours = (entry: TimesheetEntry | undefined): number => {
    if (!entry || !entry.hour_entries) return 0;
    return Object.values(entry.hour_entries).reduce((sum, hours) => sum + (hours || 0), 0);
  };


  // Calculate rate for an employee's position and hour type
  const getRate = useCallback((position: string, hourType: string): number => {
    if (!payrateConfig || !payrateConfig[position] || !payrateConfig[position][dayType]) {
      return 0;
    }
    return payrateConfig[position][dayType][hourType] || 0;
  }, [payrateConfig, dayType]);

  // Calculate total amount for an employee
  const calculateEmployeeTotal = useCallback((entry: TimesheetEntry | undefined): number => {
    if (!entry || !entry.hour_entries) return 0;

    let total = 0;
    Object.entries(entry.hour_entries).forEach(([hourType, hours]) => {
      const rate = getRate(entry.position, hourType);
      total += rate * hours;
    });
    return total;
  }, [getRate]);

  // Check if employee position is valid for payrate config
  const isPositionValid = useCallback((position: string): boolean => {
    return availablePositions.includes(position);
  }, [availablePositions]);

  // Transform data for DataTable
  const tableData: TimesheetTableRow[] = useMemo(() => {
    if (!employees || !Array.isArray(employees)) return [];

    return employees.map((employee) => {
      const entry = localEntries[employee.id];
      const totalHours = calculateTotalHours(entry);
      const totalAmount = calculateEmployeeTotal(entry);
      const hourStatus = (() => {
        if (totalHours > 16) return 'excessive'; // > 16 hours: red
        if (totalHours > 12) return 'exceeded'; // > 12 hours: yellow
        return 'normal';
      })();

      const row: TimesheetTableRow = {
        id: employee.id,
        employee_id: employee.id,
        fullname: employee.fullname,
        cccd: employee.cccd,
        employee_code: employee.employee_code,
        position: entry?.position || employee.position,
        hour_entries: entry?.hour_entries || {},
        totalHours,
        totalAmount,
        hourStatus,
      };

      // Add hour type columns dynamically
      availableHourTypes.forEach(hourType => {
        row[`hour_${hourType}`] = entry?.hour_entries?.[hourType] || 0;
      });

      return row;
    });
  }, [employees, localEntries, availableHourTypes, calculateEmployeeTotal]);

  // Utility functions for styling
  const getPositionBadgeClass = useCallback((position: string) => {
    const positionIndex = getItemIndex<string>(DEFAULT_POSITIONS as readonly string[], position);
    return getPositionBadgeStyle(positionIndex);
  }, []);

  const getDayTypeBadgeClass = (dayType: DayType) => {
    const dayTypeIndex = getItemIndex(COMMON_DAY_TYPES, dayType);
    return getDayTypeBadgeStyle(dayTypeIndex);
  };

  // Professional color sets with good contrast (from lowest to highest payrate)
  const colorSets = useMemo(() => [
    // Forest Green - Lowest cost (safest)
    {
      header: 'bg-[#e8f5e8]',
      cellFilled: 'bg-[#e8f5e8] text-[#1b4d3e]',
      cellEmpty: 'bg-[#f4faf4]'
    },
    // Ocean Blue - Low cost
    {
      header: 'bg-[#e3f2fd]',
      cellFilled: 'bg-[#e3f2fd] text-[#0d47a1]',
      cellEmpty: 'bg-[#f1f8ff]'
    },
    // Teal - Medium-low cost
    {
      header: 'bg-teal-100',
      cellFilled: 'bg-teal-100 text-teal-900',
      cellEmpty: 'bg-teal-50'
    },
    // Golden Yellow - Medium cost
    {
      header: 'bg-[#fff8e1]',
      cellFilled: 'bg-[#fff8e1] text-[#e65100]',
      cellEmpty: 'bg-[#fffcf0]'
    },
    // Coral Orange - High cost
    {
      header: 'bg-[#fce4ec]',
      cellFilled: 'bg-[#fce4ec] text-[#c2185b]',
      cellEmpty: 'bg-[#fef2f6]'
    },
    // Deep Red - Highest cost (warning)
    {
      header: 'bg-[#ffebee]',
      cellFilled: 'bg-[#ffebee] text-[#b71c1c]',
      cellEmpty: 'bg-[#fff5f6]'
    }
  ], []);

  // Get unique sorted payrates
  const uniquePayrates = useMemo(() => {
    if (!payrateConfig) return [];

    const rates: number[] = [];
    Object.keys(payrateConfig).forEach(position => {
      Object.keys(payrateConfig[position][dayType] || {}).forEach(hourType => {
        const rate = payrateConfig[position][dayType][hourType];
        if (rate > 0) rates.push(rate);
      });
    });

    return [...new Set(rates)].sort((a, b) => a - b);
  }, [payrateConfig, dayType]);

  const getHourTypeColorByPayrate = useCallback((hourType: string, position: string) => {
    const rate = getRate(position, hourType);

    if (!payrateConfig || uniquePayrates.length === 0 || rate === 0) {
      return { header: 'bg-muted/50', cellFilled: 'bg-muted/50 text-muted-foreground', cellEmpty: 'bg-gray-25' };
    }

    const rateIndex = uniquePayrates.indexOf(rate);

    // Use predefined color sets based on index, default to white/black if exceeds 6
    if (rateIndex < colorSets.length) {
      return colorSets[rateIndex];
    } else {
      // Default for rates beyond 6 different levels
      return { header: 'bg-card', cellFilled: 'bg-card text-black', cellEmpty: 'bg-gray-25' };
    }
  }, [getRate, payrateConfig, uniquePayrates, colorSets]);

  const getHourTypeInputClass = useCallback((hourType: string, hours: number, position: string) => {
    const baseClass = "text-center typography-body-medium font-medium transition-all border border-border hover:border-blue-400 hover:shadow-sm focus:bg-card focus:border-blue-500 focus:ring-2 focus:ring-ring focus:shadow-sm";
    const colors = getHourTypeColorByPayrate(hourType, position);

    if (hours > 0) {
      return `${baseClass} bg-card text-foreground border-blue-300 shadow-sm`;
    } else {
      return `${baseClass} bg-muted/50 text-foreground hover:bg-muted`;
    }
  }, [getHourTypeColorByPayrate]);

  // Column definitions for DataTable
  const columns: ColumnDef<TimesheetTableRow>[] = useMemo(() => {
    const baseColumns: ColumnDef<TimesheetTableRow>[] = [
      {
        id: "stt",
        header: "STT",
        meta: { minWidth: "60px" },
        cell: ({ row }) => (
          <div className="text-center">
            <span className="typography-body-medium text-foreground font-medium">
              {row.index + 1}
            </span>
          </div>
        ),
      },
      {
        id: "fullname",
        header: "Nhân viên",
        accessorKey: "fullname",
        meta: { minWidth: "160px" },
        cell: ({ row }) => {
          const { fullname, employee_code, totalHours, hourStatus } = row.original;
          const isValidPosition = isPositionValid(row.original.position);

          const getHourStatusColor = (status: string) => {
            switch (status) {
              case 'excessive': return 'text-red-600'; // > 16 hours: red
              case 'exceeded': return 'text-yellow-600'; // > 12 hours: yellow
              default: return 'text-muted-foreground';
            }
          };

          const getHourStatusMessage = (status: string) => {
            switch (status) {
              case 'excessive': return 'Nhân viên đã làm việc quá 16 giờ trong ngày';
              case 'exceeded': return 'Nhân viên đã làm việc quá 12 giờ trong ngày';
              default: return '';
            }
          };

          return (
            <div className="flex items-center gap-2">
              <div className="flex flex-col gap-1">
                <span className={cn(
                  "font-semibold typography-body-medium",
                  isValidPosition ? "text-foreground" : "text-gray-400"
                )}>
                  {fullname}
                </span>
                <span className="typography-body-small text-muted-foreground">
                  {employee_code}
                </span>
                {totalHours > 0 && (
                  <span className={cn(
                    "typography-body-small",
                    getHourStatusColor(hourStatus)
                  )}>
                    Tổng: {totalHours.toFixed(1)}h
                  </span>
                )}
                {!isValidPosition && (
                  <span className="typography-body-small text-orange-600">
                    Vị trí không hợp lệ
                  </span>
                )}
              </div>
              {(hourStatus === 'exceeded' || hourStatus === 'excessive') && (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger>
                      <AlertTriangle className={cn(
                        "h-4 w-4",
                        hourStatus === 'excessive' ? "text-red-500" : "text-yellow-500"
                      )} />
                    </TooltipTrigger>
                    <TooltipContent className="max-w-xs">
                      <p>{getHourStatusMessage(hourStatus)}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}
            </div>
          );
        },
      },
      {
        id: "cccd",
        header: "CCCD",
        accessorKey: "cccd",
        meta: { minWidth: "120px" },
        cell: ({ row }) => (
          <span className="font-mono typography-body-medium text-foreground">
            {row.original.cccd}
          </span>
        ),
      },
      {
        id: "position",
        header: "Vị trí",
        accessorKey: "position",
        meta: { minWidth: "140px" },
        cell: ({ row }) => {
          const isValidPosition = isPositionValid(row.original.position);
          return (
            <Badge
              variant="outline"
              className={cn(
                "typography-body-small border",
                isValidPosition
                  ? getPositionBadgeClass(row.original.position)
                  : "bg-orange-50 text-orange-700 border-orange-200"
              )}
            >
              {row.original.position}
            </Badge>
          );
        },
      },
    ];

    // Add dynamic hour type columns
    const hourColumns: ColumnDef<TimesheetTableRow>[] = availableHourTypes.map(hourType => ({
      id: `hour_${hourType}`,
      header: () => (
        <div className="text-center">
          {hourType}
          <br />
          <small className="font-normal text-muted-foreground lowercase">Giờ</small>
        </div>
      ),
      accessorKey: `hour_${hourType}`,
      meta: { minWidth: "100px" },
      cell: ({ row }) => {
        const { employee_id, position, hour_entries, hourStatus } = row.original;
        const hours = hour_entries[hourType] || 0;
        const rate = getRate(position, hourType);
        const isValidPosition = isPositionValid(position);

        return (
          <div className="relative">
            <Input
              id={`hour-input-${employee_id}-${hourType}`}
              type="number"
              min="0"
              max="24"
              step="0.5"
              value={hours || ''}
              disabled={false}
              onChange={(e) => updateHourEntry(employee_id, hourType, parseFloat(e.target.value) || 0)}
              className={cn(
                "h-8",
                getHourTypeInputClass(hourType, hours, position),
                hourStatus === 'excessive' && "border-red-300 focus:border-red-500 focus:ring-red-500/10",
                hourStatus === 'exceeded' && "border-yellow-300 focus:border-yellow-500 focus:ring-yellow-500/10",
                !isValidPosition && "border-orange-200 bg-orange-50"
              )}
              placeholder="0"
            />
          </div>
        );
      },
    }));

    // Add total amount column
    const totalColumn: ColumnDef<TimesheetTableRow> = {
      id: "totalAmount",
      header: () => (
        <div className="text-center">
          Thành tiền
          <br />
          <small className="font-normal text-muted-foreground lowercase">VND</small>
        </div>
      ),
      accessorKey: "totalAmount",
      meta: { minWidth: "120px" },
      cell: ({ row }) => {
        const { totalAmount } = row.original;
        const isValidPosition = isPositionValid(row.original.position);

        return (
          <div className="text-center">
            <span className={cn(
              "typography-body-medium",
              totalAmount > 0 ? (isValidPosition ? "text-green-700" : "text-orange-600") : "text-gray-400"
            )}>
              {totalAmount > 0 ? formatCurrency(totalAmount) : '-'}
            </span>
          </div>
        );
      },
    };

    return [...baseColumns, ...hourColumns, totalColumn];
  }, [availableHourTypes, updateHourEntry, getRate, getHourTypeInputClass, getPositionBadgeClass, isPositionValid]);

  // Loading state
  if (isLoading) {
    return (
      <Card className="p-6">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent mx-auto mb-4" />
          <p className="text-muted-foreground">Đang tải cấu hình lương...</p>
        </div>
      </Card>
    );
  }

  // No payrate config
  if (!payrateConfig) {
    return (
      <div className="space-y-4">
        <Alert className="border-amber-200 bg-amber-50">
          <AlertCircle className="h-4 w-4 text-amber-600" />
          <AlertDescription className="text-amber-800">
            <div className="space-y-3">
              <div>
                <strong>Dự án chưa có cấu hình lương</strong>
                <p className="mt-1">
                  Để nhập bảng công, bạn cần thiết lập cấu hình lương cho dự án này trước.
                  Cấu hình lương bao gồm các vị trí công việc và mức lương theo từng loại giờ làm việc.
                </p>
              </div>
              <div>
                <Button
                  onClick={() => {
                    if (selectedProject?.id) {
                      openProjectDetails(selectedProject.id.toString(), 'payrates');
                    }
                  }}
                  className="bg-amber-600 hover:bg-amber-700 text-white"
                >
                  Thiết lập cấu hình lương
                </Button>
              </div>
            </div>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // No available hour types (payrate config exists but no hour types defined)
  if (availableHourTypes.length === 0) {
    return (
      <div className="space-y-4">
        {/* Date Navigation and Day Type Information */}
        <div className="py-2">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="flex items-center gap-2">
              <Info className="h-4 w-4 text-blue-500" />
              <div>
                <span className="typography-body-medium text-foreground">Loại ngày: </span>
                <Badge variant="outline" className={cn("ml-1", getDayTypeBadgeClass(dayType))}>
                  {VIETNAMESE_PAYRATE_LABELS.dayTypes[dayType]}
                </Badge>
              </div>
            </div>
            <TimesheetDateNavigation
              selectedDate={selectedDate}
              onDateChange={onDateChange}
            />
          </div>
        </div>

        <Alert className="border-orange-200 bg-orange-50">
          <AlertTriangle className="h-4 w-4 text-orange-600" />
          <AlertDescription className="text-orange-800">
            <div className="space-y-3">
              <div>
                <strong>Cấu hình lương chưa đầy đủ</strong>
                <p className="mt-1">
                  Dự án đã có cấu hình lương nhưng chưa có loại giờ làm việc nào được định nghĩa cho ngày {VIETNAMESE_PAYRATE_LABELS.dayTypes[dayType]}.
                  Vui lòng bổ sung thêm loại giờ làm việc (ví dụ: Giờ thường, Giờ tăng ca) để có thể nhập bảng công.
                </p>
              </div>
              <div>
                <Button
                  onClick={() => {
                    if (selectedProject?.id) {
                      openProjectDetails(selectedProject.id.toString(), 'payrates');
                    }
                  }}
                  className="bg-orange-600 hover:bg-orange-700 text-white"
                >
                  Cập nhật cấu hình lương
                </Button>
              </div>
            </div>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Date Navigation and Day Type Information */}
      <div className="py-2">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div className="flex items-center gap-2">
            <Info className="h-4 w-4 text-blue-500" />
            <div>
              <span className="typography-body-medium text-foreground">Loại ngày: </span>
              <Badge variant="outline" className={cn("ml-1", getDayTypeBadgeClass(dayType))}>
                {VIETNAMESE_PAYRATE_LABELS.dayTypes[dayType]}
              </Badge>
            </div>
          </div>
          <TimesheetDateNavigation
            selectedDate={selectedDate}
            onDateChange={onDateChange}
          />
        </div>
      </div>


      <DataTable
        columns={columns}
        data={tableData}
        showPagination={false}
        primaryColumns={["fullname", "cccd"]}
        caption="Bảng chấm công nhân viên"
        className="[&_table]:table-auto"
      />
    </div>
  );
}
