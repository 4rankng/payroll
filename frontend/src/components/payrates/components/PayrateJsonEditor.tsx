import { useState, useEffect } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Textarea } from '@/components/ui/textarea';
import type { PayrateStructure } from '../types';

interface PayrateJsonEditorProps {
  rates: PayrateStructure;
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
}

export function PayrateJsonEditor({ rates, onChange, readOnly = false }: PayrateJsonEditorProps) {
  const [jsonText, setJsonText] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setJsonText(JSON.stringify(rates, null, 2));
    setError(null);
  }, [rates]);

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const text = e.target.value;
    setJsonText(text);
    
    if (readOnly) return;

    try {
      const parsed = JSON.parse(text);
      if (typeof parsed !== 'object' || parsed === null) {
        throw new Error("Cấu hình phải là một đối tượng JSON hợp lệ");
      }
      setError(null);
      onChange(parsed as PayrateStructure);
    } catch (err: unknown) {
      setError((err as Error).message || "JSON không hợp lệ");
    }
  };

  return (
    <Card className="shadow-sm">
      <CardContent className="p-0">
        <Textarea
          aria-label="Cấu hình mức lương dạng JSON"
          aria-invalid={!!error}
          value={jsonText}
          onChange={handleChange}
          readOnly={readOnly}
          className={`font-mono text-xs w-full h-[600px] rounded-xl border-0 p-4 focus-visible:ring-0 resize-y ${error ? 'bg-red-50 text-red-900' : 'bg-muted/10'}`}
          placeholder="Nhập cấu hình mức lương dạng JSON..."
          spellCheck={false}
        />
        {error && (
          <div role="alert" className="p-3 text-xs text-red-700 bg-red-50 font-medium">
            Lỗi: {error}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
