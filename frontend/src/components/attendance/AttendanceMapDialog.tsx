import { Dialog, DialogContent } from "@/components/ui/dialog";
import { AttendanceLocationMap } from "@/components/admin-dashboard/AttendanceLocationMap";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";

/**
 * Full-screen Leaflet map of a single attendance's check-in/out GPS + geofence.
 * Reuses the proven AttendanceLocationMap wrapper and the same full-bleed
 * Dialog className used by HealthDrilldownSheet's AttendanceMapDialog so Leaflet
 * reflows correctly (the FitBounds helper calls invalidateSize after the open
 * animation). Open state is driven by `row !== null`.
 */
export function AttendanceMapDialog({
  row,
  onClose,
}: {
  row: AdminAttendanceResponse | null;
  onClose: () => void;
}) {
  return (
    <Dialog
      open={row !== null}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent
        hideCloseButton
        className="inset-0 translate-x-0 translate-y-0 max-w-none max-h-none rounded-none border-0 p-0 gap-0"
      >
        {row ? (
          <AttendanceLocationMap row={row} onClose={onClose} />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
