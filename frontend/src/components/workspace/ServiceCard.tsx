import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { StatusBadge } from "@/components/common/StatusBadge";
import { cn } from "@/lib/utils";
import type { ServiceDTO } from "@/types/api";

export function ServiceCard({
  service,
  isSelected,
  onSelect,
  onStart,
  onStop,
  onRestart,
  pending,
}: {
  service: ServiceDTO;
  isSelected: boolean;
  onSelect: () => void;
  onStart: () => void;
  onStop: () => void;
  onRestart: () => void;
  pending: boolean;
}) {
  const status = service.state.status;
  const isRunning = status === "running" || status === "starting";

  return (
    <Card
      onClick={onSelect}
      className={cn(
        "cursor-pointer gap-3 transition-colors hover:bg-muted/50",
        isSelected && "ring-2 ring-primary",
      )}
    >
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>{service.name}</CardTitle>
        <StatusBadge status={status} />
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          {service.port ? <span>:{service.port}</span> : null}
          <span>{service.state.git.branch || "—"}</span>
        </div>

        <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
          <Button
            size="sm"
            variant={isRunning ? "outline" : "default"}
            disabled={pending || status === "stopping"}
            onClick={isRunning ? onStop : onStart}
          >
            {isRunning ? "Stop" : "Start"}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={pending || status === "stopped" || status === "stopping"}
            onClick={onRestart}
          >
            Restart
          </Button>
        </div>

        {service.state.lastError ? (
          <p
            className="truncate text-xs text-destructive"
            title={service.state.lastError}
          >
            {service.state.lastError}
          </p>
        ) : null}
      </CardContent>
    </Card>
  );
}
