import { format } from "date-fns";
import { getXLSX } from '@/lib/xlsx-loader';

export interface XLSExportOptions {
  fileName?: string;
  sheetName?: string;
  headers?: string[];
  dateColumns?: string[];
  numberColumns?: string[];
}

export interface XLSColumnMapping {
  key: string;
  header: string;
  formatter?: (value: unknown) => unknown;
  width?: number;
}

export async function exportToXLS<T extends Record<string, unknown>>(
  data: T[],
  options: XLSExportOptions = {}
): Promise<void> {
  const {
    fileName = `export-${new Date().toISOString().split('T')[0]}.xls`,
    sheetName = 'Sheet1',
    headers,
    dateColumns = [],
    numberColumns = []
  } = options;

  if (!data || data.length === 0) {
    throw new Error('Không có dữ liệu để xuất');
  }

  const processedData = data.map(row => {
    const processedRow: Record<string, unknown> = {};

    Object.entries(row).forEach(([key, value]) => {
      if (dateColumns.includes(key) && value) {
        processedRow[key] = format(new Date(value as string), 'dd/MM/yyyy');
      } else if (numberColumns.includes(key) && value !== null && value !== undefined) {
        processedRow[key] = Number(value);
      } else {
        processedRow[key] = value ?? '';
      }
    });

    return processedRow;
  });

  const XLSX = await getXLSX();
  const ws = XLSX.utils.json_to_sheet(processedData, {
    header: headers
  });

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, sheetName);

  XLSX.writeFile(wb, fileName, { bookType: 'xls' });
}

export async function exportToXLSWithMapping<T extends Record<string, unknown>>(
  data: T[],
  columns: XLSColumnMapping[],
  options: Omit<XLSExportOptions, 'headers'> = {}
): Promise<void> {
  const {
    fileName = `export-${new Date().toISOString().split('T')[0]}.xls`,
    sheetName = 'Sheet1'
  } = options;

  if (!data || data.length === 0) {
    throw new Error('Không có dữ liệu để xuất');
  }

  const headers = columns.map(col => col.header);
  const processedData = data.map(row => {
    const processedRow: Record<string, unknown> = {};

    columns.forEach(column => {
      const value = row[column.key];
      if (column.formatter) {
        processedRow[column.header] = column.formatter(value);
      } else {
        processedRow[column.header] = value ?? '';
      }
    });

    return processedRow;
  });

  const XLSX = await getXLSX();
  const ws = XLSX.utils.json_to_sheet(processedData, {
    header: headers
  });

  if (columns.some(col => col.width)) {
    const colWidths = columns.map(col => ({
      wch: col.width || 15
    }));
    ws['!cols'] = colWidths;
  }

  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, sheetName);

  XLSX.writeFile(wb, fileName, { bookType: 'xls' });
}

export async function downloadDataAsXLS<T extends Record<string, unknown>>(
  data: T[],
  fileName: string,
  columns?: XLSColumnMapping[]
): Promise<void> {
  try {
    if (columns) {
      await exportToXLSWithMapping(data, columns, { fileName });
    } else {
      await exportToXLS(data, { fileName });
    }
  } catch (error) {
    console.error('Lỗi xuất file XLS:', error);
    throw new Error('Không thể xuất file XLS. Vui lòng thử lại.');
  }
}
