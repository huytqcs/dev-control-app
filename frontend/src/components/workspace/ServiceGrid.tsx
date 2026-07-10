import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { ServiceCard } from "@/components/workspace/ServiceCard";
import { EmptyState } from "@/components/common/EmptyState";
import { restartService, startService, stopService } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { ServiceDTO } from "@/types/api";

function serviceMutationOptions(
  queryClient: QueryClient,
  fn: (id: string) => Promise<ServiceDTO>,
) {
  return {
    mutationFn: fn,
    onSuccess: (updated: ServiceDTO) => {
      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
        prev?.map((s) => (s.id === updated.id ? updated : s)),
      );
    },
  };
}

export function ServiceGrid({
  services,
  selectedId,
  onSelect,
}: {
  services: ServiceDTO[];
  selectedId: string | undefined;
  onSelect: (id: string) => void;
}) {
  const queryClient = useQueryClient();

  const start = useMutation(serviceMutationOptions(queryClient, startService));
  const stop = useMutation(serviceMutationOptions(queryClient, stopService));
  const restart = useMutation(
    serviceMutationOptions(queryClient, restartService),
  );

  if (services.length === 0) {
    return (
      <EmptyState
        title="No services configured"
        description="Add services to devctl.yaml to see them here."
      />
    );
  }

  return (
    <div className="h-full min-w-0 flex-1 grid grid-cols-1 content-start gap-3 overflow-y-auto p-4 sm:grid-cols-2 xl:grid-cols-3">
      {services.map((service) => {
        const pending =
          (start.isPending && start.variables === service.id) ||
          (stop.isPending && stop.variables === service.id) ||
          (restart.isPending && restart.variables === service.id);

        return (
          <ServiceCard
            key={service.id}
            service={service}
            isSelected={service.id === selectedId}
            onSelect={() => onSelect(service.id)}
            onStart={() => start.mutate(service.id)}
            onStop={() => stop.mutate(service.id)}
            onRestart={() => restart.mutate(service.id)}
            pending={pending}
          />
        );
      })}
    </div>
  );
}
