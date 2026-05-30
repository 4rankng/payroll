import React from 'react';
import { parseMetadata, formatVND, formatFieldName, formatValue, type MetadataShape } from './utils';
import { cn } from '@/lib/utils';

interface MetadataRendererProps {
  metadata: string | null | undefined;
  className?: string;
}

function ChangedFieldsDiff({ fields }: { fields: Record<string, { old: unknown; new: unknown }> }) {
  const entries = Object.entries(fields);
  if (entries.length === 0) return null;

  return (
    <div className="space-y-2">
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        Thay đổi
      </p>
      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-border bg-muted/50">
              <th className="text-left px-3 py-2 text-muted-foreground font-medium w-1/3">Trường</th>
              <th className="text-left px-3 py-2 text-red-600 font-medium w-1/3">Trước</th>
              <th className="text-left px-3 py-2 text-emerald-600 font-medium w-1/3">Sau</th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([key, change]) => (
              <tr key={key} className="border-b border-border last:border-0 odd:bg-muted/20">
                <td className="px-3 py-2 text-foreground font-medium">{formatFieldName(key)}</td>
                <td className="px-3 py-2 text-red-600 line-through opacity-70">
                  {formatValue(change?.old)}
                </td>
                <td className="px-3 py-2 text-emerald-700 font-medium">
                  {formatValue(change?.new)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function KeyValueList({
  data,
  title,
  amountKeys = [],
}: {
  data: Record<string, unknown>;
  title?: string;
  amountKeys?: string[];
}) {
  const entries = Object.entries(data).filter(([, v]) => v !== null && v !== undefined && v !== '');
  if (entries.length === 0) return null;

  return (
    <div className="space-y-2">
      {title && (
        <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          {title}
        </p>
      )}
      <div className="rounded-lg border border-border overflow-hidden">
        {entries.map(([key, value]) => (
          <div
            key={key}
            className="flex items-start gap-3 px-3 py-2 border-b border-border last:border-0 odd:bg-muted/20"
          >
            <span className="text-xs text-muted-foreground font-medium w-2/5 shrink-0 pt-0.5">
              {formatFieldName(key)}
            </span>
            <span className="text-xs text-foreground break-all">
              {amountKeys.includes(key) ? formatVND(value) : formatValue(value)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function renderShape(shape: MetadataShape): React.ReactNode {
  switch (shape.kind) {
    case 'empty':
      return (
        <p className="text-xs text-muted-foreground italic">Không có thông tin chi tiết</p>
      );
    case 'changed_fields':
      return <ChangedFieldsDiff fields={shape.fields} />;
    case 'auth':
      return <KeyValueList data={shape.data} title="Chi tiết xác thực" />;
    case 'bulk':
      return <KeyValueList data={shape.data} title="Thao tác hàng loạt" amountKeys={['total_amount']} />;
    case 'financial':
      return <KeyValueList data={shape.data} title="Thông tin tài chính" amountKeys={['amount', 'total_amount', 'settlement_amount']} />;
    case 'relationship':
      return <KeyValueList data={shape.data} title="Quan hệ đối tượng" />;
    case 'identity':
      return <KeyValueList data={shape.data} title="Thông tin đối tượng" />;
    case 'raw':
      return (
        <div className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Chi tiết</p>
          <pre className="text-xs text-foreground bg-muted rounded-lg border border-border p-3 overflow-x-auto whitespace-pre-wrap break-all">
            {JSON.stringify(shape.data, null, 2)}
          </pre>
        </div>
      );
  }
}

export function MetadataRenderer({ metadata, className }: MetadataRendererProps) {
  const shape = parseMetadata(metadata);
  return (
    <div className={cn('w-full', className)}>
      {renderShape(shape)}
    </div>
  );
}
