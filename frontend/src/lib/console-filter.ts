/*
  Development-only console filters to reduce noise from external browser extensions
  that emit: "Unchecked runtime.lastError: Could not establish connection. Receiving end does not exist."
  This does not alter application behavior; it only suppresses that specific console message.
*/

type ConsoleMethod = (...args: unknown[]) => void;

const RUNTIME_LAST_ERROR_PREFIX =
  "Unchecked runtime.lastError: Could not establish connection. Receiving end does not exist";

const SUPPRESSED_WARNINGS: string[] = [];

export function installConsoleFilters(): void {
  if (typeof window === "undefined") return;

  const originalError: ConsoleMethod = console.error.bind(console);

  console.error = (...args: unknown[]) => {
    try {
      const first = args[0];
      if (typeof first === "string") {
        if (first.startsWith(RUNTIME_LAST_ERROR_PREFIX)) return;
        if (SUPPRESSED_WARNINGS.some((w) => first.startsWith(w))) return;
      }
    } catch {
      // Fall through to original error if any unexpected shape
    }
    originalError(...args);
  };
}

