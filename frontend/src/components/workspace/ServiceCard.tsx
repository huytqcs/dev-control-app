import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { StatusBadge } from "@/components/common/StatusBadge";
import { HealthBadge } from "@/components/health/HealthBadge";
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
  const [errorExpanded, setErrorExpanded] = useState(false);
  const status = service.state.status;
  const isRunning = status === "running" || status === "starting";

  return (
    <Card
      role="button"
      tabIndex={0}
      onClick={onSelect}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onSelect();
        }
      }}
      className={cn(
        "cursor-pointer gap-2.5 transition-colors outline-none hover:bg-muted/50 focus-visible:ring-2 focus-visible:ring-ring",
        isSelected && "ring-2 ring-primary",
      )}
    >
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>{service.name}</CardTitle>
        <div className="flex items-center gap-1.5">
          <HealthBadge status={service.state.health.status} serviceStatus={status} />
          <StatusBadge status={status} />
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          {service.port ? <span>:{service.port}</span> : null}
          {service.port ? <span className="text-border">·</span> : null}
          <span className="truncate rounded-sm bg-muted px-1.5 py-0.5 font-mono">
            {service.state.git.branch || "—"}
          </span>
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
          <div onClick={(e) => e.stopPropagation()}>
            <p
              className={cn(
                "font-mono text-xs text-destructive",
                !errorExpanded && "truncate",
              )}
            >
              {service.state.lastError}
            </p>
            <button
              type="button"
              onClick={() => setErrorExpanded((v) => !v)}
              className="text-xs text-muted-foreground underline hover:text-foreground"
            >
              {errorExpanded ? "Show less" : "Show more"}
            </button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
