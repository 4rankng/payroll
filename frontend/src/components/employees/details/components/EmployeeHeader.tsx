import { useState, useEffect } from "react";
import { UserAvatar } from "@/components/ui/user-avatar";
import { userService } from "@/services/api/user.service";
import { authManager } from "@/lib/auth";
import type { EmployeeDetailsProps } from "../types";

interface EmployeeHeaderProps extends EmployeeDetailsProps {
  showName?: boolean;
}

export function EmployeeHeader({ employee, showName = true }: EmployeeHeaderProps) {
  const [creatorName, setCreatorName] = useState<string>('');
  const userRole = authManager.getUserRole();
  const isAdmin = userRole === 'admin';

  // Get creator name from API (only for admin users)
  useEffect(() => {
    const fetchCreatorName = async () => {
      if (!isAdmin || !employee?.created_by) {
        return;
      }

      try {
        const user = await userService.getUserById(employee.created_by);
        setCreatorName(user.fullname || user.username);
      } catch {
        setCreatorName(`ID #${employee.created_by}`);
      }
    };

    fetchCreatorName();
  }, [employee?.created_by, isAdmin]);

  return (
    <div className="flex items-center gap-3 w-full">
      <UserAvatar
        size="md"
        name={employee.fullname}
        username={employee.username}
        cccd={employee.cccd}
        className="border-2 border-muted flex-shrink-0"
      />

      <div className="flex-1 min-w-0">
        {showName && (
          <h1 className="text-sm font-semibold leading-tight truncate mb-0.5">
            {employee.fullname}
          </h1>
        )}
        {isAdmin && employee.created_by && (
          <div className="text-xs text-muted-foreground truncate">
            Người tạo: {creatorName || `ID #${employee.created_by}`}
          </div>
        )}
      </div>
    </div>
  );
}
