import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { realtimeClient } from "@/lib/ws";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { ServiceDTO } from "@/types/api";

// Applies service.updated events onto the services query cache, so the
// dashboard reflects state changes without polling (T-043).
export function useRealtimeEvents() {
  const queryClient = useQueryClient();

  useEffect(() => {
    realtimeClient.connect();

    const unsubscribe = realtimeClient.subscribe((event) => {
      if (event.type !== "service.updated") return;
      const payload = event.payload;

      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) => {
        if (!prev) return prev;
        return prev.map((svc) =>
          svc.id === event.serviceId
            ? {
                ...svc,
                state: {
                  status: payload.status,
                  pid: payload.pid,
                  startedAt: payload.startedAt,
                  lastError: payload.lastError,
                  lastExitCode: payload.lastExitCode,
                  git: payload.git,
                  health: payload.health,
                },
              }
            : svc,
        );
      });
    });

    return unsubscribe;
  }, [queryClient]);
}
