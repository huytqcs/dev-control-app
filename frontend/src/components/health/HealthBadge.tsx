import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { HealthStatus } from "@/types/api";

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

export function HealthBadge({ status }: { status: HealthStatus }) {
  if (status === "unknown") return null;

  return (
    <Badge variant="outline" className="gap-1.5">
      <span className={cn("size-1.5 rounded-full", DOT[status])} />
      {LABEL[status]}
    </Badge>
  );
}
