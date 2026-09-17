import { cn } from "@/lib/utils";
import { METHOD_COLORS, parseEndpoint } from "./utils";

interface Props {
  endpoint: string;
  className?: string;
}

/**
 * Renders METHOD badge + path. Never truncates — path is full text.
 * Fits in a table cell that has overflow-x: auto if needed.
 */
export function EndpointLabel({ endpoint, className }: Props) {
  const parsed = parseEndpoint(endpoint);

  if (!parsed) {
    return (
      <span className={cn("font-mono text-sm whitespace-nowrap", className)}>
        {endpoint}
      </span>
    );
  }

  return (
    <span className={cn("font-mono text-sm whitespace-nowrap", className)}>
      <span
        className={cn(
          "inline-block text-xs font-bold px-1.5 py-0.5 rounded mr-2 align-middle leading-none",
          METHOD_COLORS[parsed.method] ?? "bg-muted text-muted-foreground"
        )}
      >
        {parsed.method}
      </span>
      {parsed.path}
    </span>
  );
}
