import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { runAction } from "@/lib/api";
import { realtimeClient } from "@/lib/ws";
import type { ActionSummaryDTO, LogEntry, ServiceDTO } from "@/types/api";

export function ActionsPanel({ service }: { service: ServiceDTO }) {
  const [running, setRunning] = useState<Record<string, boolean>>({});
  const [outputs, setOutputs] = useState<Record<string, LogEntry[]>>({});
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});

  // actionId -> runId currently being streamed, so incoming events can be
  // filtered to the run that's actually in flight for that action.
  const activeRunsRef = useRef<Record<string, string>>({});

  const markDone = (actionId: string) => {
    setRunning((prev) => ({ ...prev, [actionId]: false }));
    delete activeRunsRef.current[actionId];
  };

  useEffect(() => {
    realtimeClient.connect();
    const unsubscribe = realtimeClient.subscribe((event) => {
      if (event.serviceId !== service.id) return;

      if (event.type === "action.output") {
        const { actionId, runId, entry } = event.payload;
        if (activeRunsRef.current[actionId] !== runId) return;
        setOutputs((prev) => ({
          ...prev,
          [actionId]: [...(prev[actionId] ?? []), entry],
        }));
        return;
      }

      if (event.type === "action.completed") {
        const { actionId, runId, success, error } = event.payload;
        if (activeRunsRef.current[actionId] !== runId) return;
        if (!success) {
          setErrors((prev) => ({ ...prev, [actionId]: error || "Action failed" }));
        }
        markDone(actionId);
      }
    });

    return unsubscribe;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [service.id]);

  const handleRun = (action: ActionSummaryDTO) => {
    setRunning((prev) => ({ ...prev, [action.id]: true }));
    setOutputs((prev) => ({ ...prev, [action.id]: [] }));
    setErrors((prev) => ({ ...prev, [action.id]: undefined }));

    runAction(service.id, action.id)
      .then(({ runId }) => {
        activeRunsRef.current[action.id] = runId;
      })
      .catch((err) => {
        setErrors((prev) => ({
          ...prev,
          [action.id]: err instanceof Error ? err.message : String(err),
        }));
        markDone(action.id);
      });
  };

  if (service.actions.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No actions configured for this service.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {service.actions.map((action) => {
        const isRunning = running[action.id] ?? false;
        const output = outputs[action.id] ?? [];
        const error = errors[action.id];

        return (
          <div key={action.id} className="flex flex-col gap-2 rounded-md border p-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">{action.name}</span>
              <Button
                size="sm"
                variant="default"
                disabled={isRunning}
                onClick={() => handleRun(action)}
              >
                {isRunning ? "Running…" : "Run"}
              </Button>
            </div>

            {error ? (
              <p className="text-xs text-red-500">{error}</p>
            ) : null}

            {output.length > 0 ? (
              <div className="max-h-48 w-full overflow-y-auto rounded-sm bg-background p-2 font-mono text-xs leading-relaxed">
                {output.map((entry) => (
                  <div
                    key={entry.id}
                    className={cn(
                      "whitespace-pre-wrap break-all",
                      entry.source === "stderr" && "text-red-500",
                    )}
                  >
                    <span className="mr-2 text-muted-foreground">
                      {new Date(entry.time).toLocaleTimeString()}
                    </span>
                    {entry.line}
                  </div>
                ))}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
