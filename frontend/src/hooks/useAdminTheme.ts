import { useCallback, useEffect, useState } from "react";

export type AdminTheme = "light" | "dark";

const STORAGE_KEY = "payroll-admin-theme";

/** Light is the default; there is no system option. Storage failures (private
 *  mode, blocked cookies) degrade to the light default for the session. */
function readStoredTheme(): AdminTheme {
  try {
    return localStorage.getItem(STORAGE_KEY) === "dark" ? "dark" : "light";
  } catch {
    return "light";
  }
}

/**
 * Admin-only appearance toggle (toto.sg dark console). Flipping adds/removes
 * the `dark` class on <html>, which re-points the console tokens through
 * `html.admin-console-toto.dark` in admin-console-toto.css. On unmount the
 * class is always removed, so leaving admin (logout, partner/employee portals,
 * login) never carries `html.dark` into a light-only surface.
 */
export function useAdminTheme(): { theme: AdminTheme; toggle: () => void } {
  // Synchronous initializer: the toggle renders the right icon on first paint,
  // matching the class the index.html pre-paint guard already applied.
  const [theme, setTheme] = useState<AdminTheme>(readStoredTheme);

  useEffect(() => {
    const root = document.documentElement;
    if (theme === "dark") root.classList.add("dark");
    return () => root.classList.remove("dark");
  }, [theme]);

  const toggle = useCallback(() => {
    setTheme((current) => {
      const next: AdminTheme = current === "dark" ? "light" : "dark";
      try {
        localStorage.setItem(STORAGE_KEY, next);
      } catch {
        // Storage unavailable: the flip still applies for this session.
      }
      return next;
    });
  }, []);

  return { theme, toggle };
}
