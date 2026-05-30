import { CheckCircle2 } from "lucide-react";

export function AllClear({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center gap-2 py-10 text-emerald-600">
      <CheckCircle2 className="h-7 w-7 opacity-50" />
      <span className="text-sm font-medium">{text}</span>
    </div>
  );
}
