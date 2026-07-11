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
        "fixed z-50 top-4",
        "flex items-center justify-center",
        "h-9 w-2.5 rounded-r-md border border-l-0 border-white/15",
        "bg-[#263d33] text-white/70 shadow-sm",
        "before:absolute before:-inset-y-1 before:-left-4 before:-right-[17px] before:content-['']",
        "hover:bg-[#315042] hover:text-white",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-400 focus-visible:ring-offset-2",
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
          "relative z-10 h-3.5 w-3.5 shrink-0 transition-transform duration-200",
          open && "rotate-180"
        )}
        aria-hidden="true"
      />
    </button>
  );
};
