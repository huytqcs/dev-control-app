import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { startWorker, stopWorker } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { ServiceDTO, WorkerSummaryDTO } from "@/types/api";

const DOT: Record<string, string> = {
  running: "bg-emerald-500",
  starting: "bg-amber-500 animate-pulse",
  failed: "bg-red-500",
  stopped: "bg-muted-foreground/50",
};

export function WorkerControls({ service }: { service: ServiceDTO }) {
  const queryClient = useQueryClient();

  const start = useMutation({
    mutationFn: (workerId: string) => startWorker(service.id, workerId),
    onSuccess: (updated) =>
      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
        prev?.map((s) => (s.id === updated.id ? updated : s)),
      ),
  });
  const stop = useMutation({
    mutationFn: (workerId: string) => stopWorker(service.id, workerId),
    onSuccess: (updated) =>
      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
        prev?.map((s) => (s.id === updated.id ? updated : s)),
      ),
  });

  if (service.workers.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No workers configured for this service.</p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {service.workers.map((worker: WorkerSummaryDTO) => {
        const isRunning = worker.status === "running" || worker.status === "starting";
        const pending =
          (start.isPending && start.variables === worker.id) ||
          (stop.isPending && stop.variables === worker.id);

        return (
          <div
            key={worker.id}
            className="flex items-center justify-between rounded-md border p-2"
          >
            <div className="flex items-center gap-2">
              <span
                className={cn("size-1.5 rounded-full", DOT[worker.status] ?? DOT.stopped)}
              />
              <span className="text-sm font-medium">{worker.name}</span>
              <Badge variant="outline">{worker.status}</Badge>
            </div>
            <Button
              size="sm"
              variant={isRunning ? "outline" : "default"}
              disabled={pending}
              onClick={() =>
                isRunning ? stop.mutate(worker.id) : start.mutate(worker.id)
              }
            >
              {isRunning ? "Stop" : "Start"}
            </Button>
          </div>
        );
      })}
    </div>
  );
}
