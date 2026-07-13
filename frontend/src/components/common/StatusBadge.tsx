import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { ServiceStatus } from "@/types/api";

export const STATUS_LABEL: Record<ServiceStatus, string> = {
  running: "Running",
  starting: "Starting",
  stopping: "Stopping",
  failed: "Failed",
  stopped: "Stopped",
};

export const STATUS_DOT: Record<ServiceStatus, string> = {
  running: "bg-emerald-500",
  starting: "bg-amber-500",
  stopping: "bg-amber-500",
  failed: "bg-red-500",
  stopped: "bg-muted-foreground/50",
};

export function StatusBadge({ status }: { status: ServiceStatus }) {
  return (
    <Badge variant="outline" className="gap-1.5">
      <span
        className={cn(
          "size-1.5 rounded-full",
          STATUS_DOT[status],
          (status === "starting" || status === "stopping") && "animate-pulse",
        )}
      />
      {STATUS_LABEL[status]}
    </Badge>
  );
}
