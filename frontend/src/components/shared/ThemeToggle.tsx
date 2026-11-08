import { Moon01, Sun } from "@untitledui/icons";
import { useAdminTheme } from "@/hooks/useAdminTheme";
import { cn } from "@/lib/utils";

/**
 * Admin-only light/dark switch (toto.sg dark console). Shows the mode it
 * switches to, so the icon reads as the action rather than the state. Icon-only
 * so it renders in both the expanded and collapsed sidebar rail. Colors ride
 * the console tokens (border/card/muted/ring), not the static UU palette, so
 * the button re-skins with the dark token block.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const { theme, toggle } = useAdminTheme();

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label="Chuyển giao diện sáng/tối"
      aria-pressed={theme === "dark"}
      title="Chuyển giao diện sáng/tối"
      className={cn(
        "grid h-9 w-9 shrink-0 place-items-center rounded-md border border-border bg-card text-muted-foreground",
        "transition-colors duration-100 ease-linear hover:bg-muted hover:text-foreground",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        className,
      )}
    >
      {theme === "dark" ? (
        <Sun aria-hidden="true" className="size-5" />
      ) : (
        <Moon01 aria-hidden="true" className="size-5" />
      )}
    </button>
  );
}
