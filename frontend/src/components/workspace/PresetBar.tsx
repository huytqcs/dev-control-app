import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { startPreset, stopPreset } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { PresetDTO } from "@/types/api";

// Preset start/stop starts/stops each service in dependency order on the
// backend (T-051); the frontend just fires the request and lets
// service.updated WS events reflect per-service progress as it happens
// (ARCHITECTURE.md §6.3) — no separate preset-level state to track here.
export function PresetBar({ presets }: { presets: PresetDTO[] }) {
  const queryClient = useQueryClient();

  const start = useMutation({
    mutationFn: startPreset,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: servicesQueryKey }),
  });
  const stop = useMutation({
    mutationFn: stopPreset,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: servicesQueryKey }),
  });

  if (presets.length === 0) return null;

  const error =
    (start.data?.errors.length ? start.data.errors.join("; ") : null) ??
    (stop.data?.errors.length ? stop.data.errors.join("; ") : null);

  return (
    <div className="flex items-center gap-2">
      {presets.map((preset) => {
        const pending =
          (start.isPending && start.variables === preset.id) ||
          (stop.isPending && stop.variables === preset.id);

        return (
          <div key={preset.id} className="flex items-center gap-1 rounded-md border p-1">
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
          </div>
        );
      })}
      {error ? <span className="text-xs text-destructive">{error}</span> : null}
    </div>
  );
}
