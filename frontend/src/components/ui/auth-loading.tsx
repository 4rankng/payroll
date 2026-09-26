import { Loader2 } from "lucide-react";

export function AuthLoadingScreen() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-card">
      <div className="flex flex-col items-center gap-4">
        <div className="relative">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
        <div className="text-center space-y-2">
          <h3 className="typography-title-medium font-semibold">
            Đang xác thực...
          </h3>
          <p className="typography-body-medium text-muted-foreground">
            Vui lòng chờ trong giây lát
          </p>
        </div>
      </div>
    </div>
  );
}