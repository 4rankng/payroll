import { useState, useEffect } from 'react';
import { toast } from "@/components/ui/sonner";
import { timesheetService } from '@/services/api/timesheet.service';
import { Timesheet, TimesheetSummary } from '@/types/api/timesheet.types';

export const usePartnerTimesheetData = () => {
  const [timesheetData, setTimesheetData] = useState<Timesheet[]>([]);
  const [summary, setSummary] = useState<TimesheetSummary | null>(null);
  const [selectedMonth, setSelectedMonth] = useState("2025-08");
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchTimesheetData = async () => {
      setIsLoading(true);
      try {
        const startDate = `${selectedMonth}-01`;
        const endDate = `${selectedMonth}-31`;

        // Fetch both summary and timesheet data
        const [summaryResponse, timesheetsResponse] = await Promise.all([
          timesheetService.getSummary({
            fromDate: startDate,
            toDate: endDate
          }),
          timesheetService.getTimesheets({
            fromDate: startDate,
            toDate: endDate,
            page: 1,
            pageSize: 1000,
            sortBy: 'date',
            sortOrder: 'desc'
          })
        ]);

        setSummary(summaryResponse);

        // Handle the timesheet data based on response structure
        if (timesheetsResponse?.data) {
          // If response has a data array directly
          if (Array.isArray(timesheetsResponse.data)) {
            setTimesheetData(timesheetsResponse.data);
          }
          // If response has a nested data structure
          else if ((timesheetsResponse.data as { timesheets?: unknown }).timesheets) {
            setTimesheetData((timesheetsResponse.data as unknown as { timesheets: Timesheet[] }).timesheets);
          }
          // If response data itself is the timesheet array
          else {
            setTimesheetData([]);
          }
        } else {
          setTimesheetData([]);
        }

      } catch (error) {
        console.error('Failed to fetch timesheet data:', error);
        toast({
          title: "Lỗi tải dữ liệu",
          description: "Không thể tải dữ liệu bảng công. Vui lòng thử lại.",
          variant: "destructive"
        });
        setTimesheetData([]);
        setSummary(null);
      } finally {
        setIsLoading(false);
      }
    };

    fetchTimesheetData();
  }, [selectedMonth]);

  const handleUpload = async (file: File, projectId?: number) => {
    try {
      await timesheetService.importTimesheets(file, projectId);
      toast({
        title: "Nhập bảng công thành công",
        description: "Dữ liệu bảng công đã được cập nhật.",
      });

      // Refresh data after upload
      const startDate = `${selectedMonth}-01`;
      const endDate = `${selectedMonth}-31`;

      const [summaryResponse, timesheetsResponse] = await Promise.all([
        timesheetService.getSummary({
          fromDate: startDate,
          toDate: endDate
        }),
        timesheetService.getTimesheets({
          fromDate: startDate,
          toDate: endDate,
          page: 1,
          pageSize: 1000,
          sortBy: 'date',
          sortOrder: 'desc'
        })
      ]);

      setSummary(summaryResponse);

      if (timesheetsResponse?.data) {
        if (Array.isArray(timesheetsResponse.data)) {
          setTimesheetData(timesheetsResponse.data);
        } else if ((timesheetsResponse.data as { timesheets?: unknown }).timesheets) {
          setTimesheetData((timesheetsResponse.data as { timesheets: Timesheet[] }).timesheets);
        } else {
          setTimesheetData([]);
        }
      }
    } catch (error) {
      toast({
        title: "Lỗi nhập bảng công",
        description: "Không thể nhập dữ liệu bảng công. Vui lòng thử lại.",
        variant: "destructive"
      });
    }
  };

  return {
    timesheetData,
    summary,
    selectedMonth,
    isLoading,
    setSelectedMonth,
    handleUpload
  };
};
