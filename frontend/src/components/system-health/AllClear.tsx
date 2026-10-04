// W13c: lucide → verified @untitledui/icons.
import { CheckCircle } from "@untitledui/icons";

export function AllClear({ text }: { text: string }) {
  return (
    <div className="flex flex-col items-center gap-2 py-10 text-fg-success-primary">
      <CheckCircle className="h-7 w-7 opacity-50" />
      <span className="text-sm font-medium">{text}</span>
    </div>
  );
}
