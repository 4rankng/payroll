import { ChevronRight } from "lucide-react";
import { useSidebar } from "@/components/ui/sidebar";
import { cn } from "@/lib/utils";

/**
 * Fixed-position toggle tab attached to the sidebar's right edge.
 */
export const SidebarToggle = ({ className }: { className?: string }) => {
  const { open, toggleSidebar, isMobile } = useSidebar();

  if (isMobile) return null;

  return (
    <button
      type="button"
      onClick={toggleSidebar}
      aria-label={open ? "Thu gọn sidebar" : "Mở rộng sidebar"}
      aria-expanded={open}
      className={cn(
        "admin-toggle-tab fixed z-50 top-5",
        "flex items-center justify-center",
        "h-10 w-6 -translate-x-px rounded-r-xl border border-l-0 border-white/15 bg-[#263d33] text-white/75 shadow-sm",
        "before:absolute before:-inset-y-1 before:-left-3 before:-right-3 before:content-['']",
        "hover:bg-[#315042] hover:text-white",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--secondary))] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))]",
        "transition-[left,background-color,color] duration-200 ease-in-out",
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
          "relative z-10 h-4 w-4 shrink-0 transition-transform duration-200",
          open && "rotate-180"
        )}
        aria-hidden="true"
      />
    </button>
  );
};
