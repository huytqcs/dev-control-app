import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { startPreset, stopPreset } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { PresetDTO, ServiceDTO } from "@/types/api";

// Preset start/stop starts/stops each service in dependency order on the
// backend (T-051); the frontend just fires the request and lets
// service.updated WS events reflect per-service progress as it happens
// (ARCHITECTURE.md §6.3) — no separate preset-level state to track here.
//
// T-079: while a preset's start/stop mutation is pending we derive live
// "starting N/M" / "stopping N/M" progress from the `services` prop, which
// useRealtimeEvents keeps current via service.updated WS events — no
// polling needed here. Per-service failures from the mutation result are
// shown scoped to that preset's row (compact badge + expandable list)
// instead of one generic error string for the whole bar.
export function PresetBar({
  presets,
  services,
}: {
  presets: PresetDTO[];
  services: ServiceDTO[];
}) {
  const queryClient = useQueryClient();
  const [expandedPresetId, setExpandedPresetId] = useState<string | null>(null);

  const start = useMutation({
    mutationFn: startPreset,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: servicesQueryKey }),
  });
  const stop = useMutation({
    mutationFn: stopPreset,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: servicesQueryKey }),
  });

  if (presets.length === 0) return null;

  const servicesById = new Map(services.map((svc) => [svc.id, svc]));

  return (
    <div className="flex items-center gap-2">
      {presets.map((preset) => {
        const starting = start.isPending && start.variables === preset.id;
        const stopping = stop.isPending && stop.variables === preset.id;
        const pending = starting || stopping;

        const total = preset.services.length;
        const settled = preset.services.reduce((count, serviceId) => {
          const status = servicesById.get(serviceId)?.state.status;
          if (starting && status === "running") return count + 1;
          if (stopping && status === "stopped") return count + 1;
          return count;
        }, 0);

        // Only the most recently fired mutation's result/variables are
        // available from useMutation, so scope errors to this preset by
        // checking `variables` matches — a stale result from a previous
        // preset is dropped as soon as any other preset's mutation fires.
        const errors: string[] =
          (start.variables === preset.id ? start.data?.errors : undefined) ??
          (stop.variables === preset.id ? stop.data?.errors : undefined) ??
          [];
        const showErrors = !pending && errors.length > 0;
        const expanded = expandedPresetId === preset.id;

        return (
          <div
            key={preset.id}
            className="relative flex items-center gap-1 rounded-md border p-1"
          >
            <span className="px-1.5 text-xs font-medium text-muted-foreground">
              {preset.name}
            </span>
            <Button
              size="sm"
              variant="outline"
              disabled={pending}
              onClick={() => start.mutate(preset.id)}
            >
              Start
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={pending}
              onClick={() => stop.mutate(preset.id)}
            >
              Stop
            </Button>
            {pending ? (
              <span className="px-1 text-xs tabular-nums text-muted-foreground">
                {starting ? "starting" : "stopping"} {settled}/{total}
              </span>
            ) : null}
            {showErrors ? (
              <button
                type="button"
                className="rounded bg-destructive/10 px-1.5 py-0.5 text-xs font-medium text-destructive"
                title={errors.join("; ")}
                onClick={() => setExpandedPresetId(expanded ? null : preset.id)}
              >
                {errors.length} failed
              </button>
            ) : null}
            {showErrors && expanded ? (
              <ul className="absolute right-0 top-full z-10 mt-1 max-w-xs list-disc space-y-0.5 rounded-md border bg-popover p-2 pl-5 text-xs text-destructive shadow-md">
                {errors.map((message, i) => (
                  <li key={i}>{message}</li>
                ))}
              </ul>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
