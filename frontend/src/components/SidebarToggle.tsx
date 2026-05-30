import { ChevronRight } from "lucide-react";
import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";

/**
 * Fixed-position toggle tab that sticks to the sidebar's right edge.
 * Pill-shaped tab with a chevron that rotates based on sidebar state.
 */
export const SidebarToggle = ({ className }: { className?: string }) => {
  const { open, toggleSidebar, isMobile } = useSidebar();

  if (isMobile) return null;

  return (
    <button
      type="button"
      onClick={toggleSidebar}
      aria-label={open ? "Thu gọn sidebar" : "Mở rộng sidebar"}
      className={cn(
        "fixed z-50 top-4",
        "flex items-center justify-center",
        "w-2.5 h-9",
        "bg-neutral-800 border border-neutral-700 rounded-r-md shadow-sm",
        "text-neutral-300",
        "hover:bg-accent hover:shadow-sm hover:text-foreground",
        "transition-all duration-300 ease-in-out",
        className
      )}
      style={{
        left: open
          ? "var(--sidebar-width)"
          : "var(--sidebar-width-icon)",
      }}
    >
      <ChevronRight
        className={cn(
          "w-3.5 h-3.5 transition-transform duration-200",
          open && "rotate-180"
        )}
      />
    </button>
  );
};
