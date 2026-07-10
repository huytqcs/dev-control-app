import { useEffect, useState } from "react";
import { getServiceLogs } from "@/lib/api";
import { realtimeClient } from "@/lib/ws";
import type { LogEntry } from "@/types/api";

const MAX_BUFFERED_LINES = 5000;

// Logs are kept out of TanStack Query's cache and appended locally instead —
// they're a high-frequency stream, not a request/response snapshot
// (SPEC.md §22.2: keep log updates localized to log viewer state).
export function useServiceLogs(serviceId: string | undefined) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!serviceId) {
      setLogs([]);
      return;
    }

    let cancelled = false;
    setIsLoading(true);
    setError(null);

    getServiceLogs(serviceId)
      .then((entries) => {
        if (!cancelled) setLogs(entries);
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });

    realtimeClient.connect();
    const unsubscribe = realtimeClient.subscribe((event) => {
      if (event.type !== "log.appended" || event.serviceId !== serviceId) return;
      setLogs((prev) => {
        const next = [...prev, event.payload.entry];
        return next.length > MAX_BUFFERED_LINES
          ? next.slice(next.length - MAX_BUFFERED_LINES)
          : next;
      });
    });

    return () => {
      cancelled = true;
      unsubscribe();
    };
  }, [serviceId]);

  return { logs, isLoading, error };
}
