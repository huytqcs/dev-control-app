import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { PresetBar } from "@/components/workspace/PresetBar";
import { stopAll } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { PresetDTO, ServiceDTO } from "@/types/api";

export function TopBar({
  services,
  presets,
}: {
  services: ServiceDTO[];
  presets: PresetDTO[];
}) {
  const queryClient = useQueryClient();
  const running = services.filter((s) => s.state.status === "running").length;
  const failed = services.filter((s) => s.state.status === "failed").length;

  // Deliberate action, not an implicit stop-on-exit — services are meant to
  // survive a devctl backend restart (ReconcileOrphans, SPEC.md §26.1), so
  // this is the explicit "I'm done for now" equivalent instead (GAMMA_PLAN.md
  // T-080 decision).
  const stopAllMut = useMutation({
    mutationFn: stopAll,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: servicesQueryKey }),
  });

  return (
    <header className="flex h-14 shrink-0 items-center gap-4 border-b px-4">
      <h1 className="text-sm font-semibold">devctl</h1>
      <div className="flex items-center gap-3 text-sm text-muted-foreground">
        <span>
          Running <span className="font-medium text-foreground">{running}</span>
        </span>
        {failed > 0 ? (
          <span className="text-destructive">
            Failed <span className="font-medium">{failed}</span>
          </span>
        ) : null}
        <span>
          Total <span className="font-medium text-foreground">{services.length}</span>
        </span>
      </div>
      <div className="ml-auto flex items-center gap-2">
        <PresetBar presets={presets} services={services} />
        <Button
          size="sm"
          variant="outline"
          disabled={stopAllMut.isPending || running === 0}
          onClick={() => stopAllMut.mutate()}
        >
          Stop All
        </Button>
      </div>
      {stopAllMut.data?.errors.length ? (
        <span className="text-xs text-destructive">
          {stopAllMut.data.errors.join("; ")}
        </span>
      ) : null}
    </header>
  );
}
