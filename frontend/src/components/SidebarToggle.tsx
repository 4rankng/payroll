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
        "admin-toggle-tab ct-btn ct-btn-ghost fixed z-50 top-4 min-h-0 p-0",
        "flex items-center justify-center",
        "h-7 w-4 -translate-x-px border border-white/15 bg-sidebar-accent text-white/75 shadow-sm",
        "before:absolute before:-inset-y-2 before:-left-3.5 before:-right-3.5 before:content-['']",
        "hover:bg-sidebar-accent/80 hover:text-white",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--secondary))] focus-visible:ring-offset-2 focus-visible:ring-offset-[hsl(var(--background))]",
        "transition-[left,background-color,color] duration-200 ease-in-out",
        className
      )}
      style={{
        borderTopLeftRadius: 0,
        borderBottomLeftRadius: 0,
        borderTopRightRadius: "0.75rem",
        borderBottomRightRadius: "0.75rem",
        borderLeftWidth: 0,
        left: open
          ? "var(--sidebar-width)"
          : "var(--sidebar-width-icon)",
      }}
    >
      <ChevronRight
        className="relative z-10 h-3 w-3 shrink-0"
        aria-hidden="true"
      />
    </button>
  );
};
