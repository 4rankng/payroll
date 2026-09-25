import React from 'react';
import {
  parseMetadata,
  formatAuditValue,
  formatFieldName,
  type AuditFieldChange,
  type MetadataShape,
} from './utils';
import { cn } from '@/lib/utils';

interface MetadataRendererProps {
  metadata: string | null | undefined;
  action?: string;
  entityType?: string;
  className?: string;
}

function ChangedFieldsDiff({ fields }: { fields: Record<string, AuditFieldChange> }) {
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
              <th className="text-left px-3 py-2 text-emerald-700 font-medium w-1/3">Sau</th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([key, change]) => (
              <tr key={key} className="border-b border-border last:border-0 odd:bg-muted/20">
                <td className="px-3 py-2 text-foreground font-medium">{formatFieldName(key)}</td>
                <td className="px-3 py-2 text-red-600 line-through opacity-70">
                  {formatAuditValue(key, change.before)}
                </td>
                <td className="px-3 py-2 text-emerald-700 font-medium">
                  {formatAuditValue(key, change.after)}
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
}: {
  data: Record<string, unknown>;
  title?: string;
}) {
  const entries = Object.entries(data);
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
              {formatAuditValue(key, value)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function getDetailsTitle(action?: string): string {
  switch (action) {
    case 'CREATE':
    case 'BULK_CREATE':
      return 'Giá trị được tạo';
    case 'DELETE':
      return 'Giá trị đã xóa';
    case 'UPDATE':
      return 'Giá trị được ghi nhận';
    default:
      return 'Chi tiết';
  }
}

function renderShape(shape: MetadataShape, action?: string): React.ReactNode {
  switch (shape.kind) {
    case 'empty':
      return (
        <p className="text-xs text-muted-foreground italic">Không có thông tin chi tiết</p>
      );
    case 'changed_fields':
      return (
        <div className="space-y-4">
          <ChangedFieldsDiff fields={shape.fields} />
          {Object.keys(shape.context).length > 0 && (
            <KeyValueList data={shape.context} title="Thông tin liên quan" />
          )}
        </div>
      );
    case 'details':
      return <KeyValueList data={shape.data} title={getDetailsTitle(action)} />;
    case 'invalid':
      return (
        <p className="text-xs text-destructive">
          Dữ liệu chi tiết không hợp lệ
        </p>
      );
  }
}

export function MetadataRenderer({ metadata, action, className }: MetadataRendererProps) {
  const shape = parseMetadata(metadata);
  return (
    <div className={cn('w-full', className)}>
      {renderShape(shape, action)}
    </div>
  );
}
