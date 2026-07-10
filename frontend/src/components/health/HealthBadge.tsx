import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { HealthStatus, ServiceStatus } from "@/types/api";

const LABEL: Record<HealthStatus, string> = {
  healthy: "Healthy",
  unhealthy: "Unhealthy",
  unknown: "No health check",
};

const DOT: Record<HealthStatus, string> = {
  healthy: "bg-emerald-500",
  unhealthy: "bg-red-500",
  unknown: "bg-muted-foreground/40",
};

export function HealthBadge({
  status,
  serviceStatus,
}: {
  status: HealthStatus;
  // Health only ever means anything while the process is actually running —
  // the backend only resets it to "unknown" once watchExit sees the process
  // exit, which can lag the UI's own status by seconds (SIGTERM'd processes
  // can take up to DefaultStopTimeout to die) or, if a WS event was ever
  // dropped, might not resync until the next background poll. Gate on
  // serviceStatus directly instead of relying on the health field alone —
  // hide whenever the service isn't "running", not just once it's fully
  // "stopped", so the stale badge can't reappear during "stopping" either.
  serviceStatus: ServiceStatus;
}) {
  if (status === "unknown" || serviceStatus !== "running") return null;

  return (
    <Badge variant="outline" className="gap-1.5">
      <span className={cn("size-1.5 rounded-full", DOT[status])} />
      {LABEL[status]}
    </Badge>
  );
}
