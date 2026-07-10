import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { realtimeClient } from "@/lib/ws";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { ServiceDTO } from "@/types/api";

// Applies service/health/git/worker.updated events onto the services query
// cache, so the dashboard reflects state changes without polling (T-043,
// T-054, T-058, T-062).
export function useRealtimeEvents() {
  const queryClient = useQueryClient();

  useEffect(() => {
    realtimeClient.connect();

    // Anything broadcast while disconnected (a dropped WS, a server-side
    // buffer overflow during a burst of events, a backgrounded tab) is
    // gone for good — there's no missed-event replay. Force a full resync
    // the moment the socket comes back instead of waiting on the slow
    // background poll to eventually notice.
    const unsubscribeReconnect = realtimeClient.onReconnect(() => {
      queryClient.invalidateQueries({ queryKey: servicesQueryKey });
    });

    const unsubscribe = realtimeClient.subscribe((event) => {
      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) => {
        if (!prev) return prev;

        switch (event.type) {
          case "service.updated": {
            const payload = event.payload;
            return prev.map((svc) =>
              svc.id === event.serviceId
                ? {
                    ...svc,
                    state: {
                      ...svc.state,
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
          }
          case "health.updated":
            return prev.map((svc) =>
              svc.id === event.serviceId
                ? { ...svc, state: { ...svc.state, health: event.payload.health } }
                : svc,
            );
          case "git.updated":
            return prev.map((svc) =>
              svc.id === event.serviceId
                ? { ...svc, state: { ...svc.state, git: event.payload.git } }
                : svc,
            );
          case "worker.updated":
            return prev.map((svc) =>
              svc.id === event.serviceId
                ? {
                    ...svc,
                    workers: svc.workers.map((w) =>
                      w.id === event.payload.worker.id ? event.payload.worker : w,
                    ),
                  }
                : svc,
            );
          default:
            return prev;
        }
      });
    });

    return () => {
      unsubscribe();
      unsubscribeReconnect();
    };
  }, [queryClient]);
}
