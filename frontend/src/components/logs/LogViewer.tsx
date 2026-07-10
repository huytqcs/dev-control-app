import { useEffect, useMemo, useRef, useState } from "react";
import { Copy, Search, Trash2 } from "lucide-react";
import { useServiceLogs } from "@/hooks/useServiceLogs";
import { LoadingState } from "@/components/common/LoadingState";
import { EmptyState } from "@/components/common/EmptyState";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { LogEntry } from "@/types/api";
import {
  formatLogLine,
  isErrorLine,
  NEAR_BOTTOM_THRESHOLD_PX,
  type SeverityFilter,
} from "./LogViewer.helpers";

export function LogViewer({ serviceId }: { serviceId: string }) {
  const { logs, isLoading, error } = useServiceLogs(serviceId);
  const containerRef = useRef<HTMLDivElement>(null);

  const [searchQuery, setSearchQuery] = useState("");
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>("all");
  // Session-local "clear view": remembers the id of the last entry that was
  // visible when the user cleared, so later renders can hide everything up
  // to (and including) that point without touching the backend ring buffer.
  const [clearedThroughId, setClearedThroughId] = useState<string | null>(null);
  const [autoFollow, setAutoFollow] = useState(true);
  const [pendingCount, setPendingCount] = useState(0);
  const [justCopied, setJustCopied] = useState(false);

  // Reset all local view state when switching services so filters/clear/
  // follow state from one service don't leak into another.
  useEffect(() => {
    setSearchQuery("");
    setSeverityFilter("all");
    setClearedThroughId(null);
    setAutoFollow(true);
    setPendingCount(0);
  }, [serviceId]);

  const baseLogs = useMemo(() => {
    if (!clearedThroughId) return logs;
    const idx = logs.findIndex((entry) => entry.id === clearedThroughId);
    // If the marker itself fell out of the (backend-trimmed) buffer, every
    // remaining entry is newer than it, so nothing needs to be hidden.
    return idx === -1 ? logs : logs.slice(idx + 1);
  }, [logs, clearedThroughId]);

  const severityFiltered = useMemo(() => {
    if (severityFilter === "all") return baseLogs;
    return baseLogs.filter((entry) => entry.source === severityFilter);
  }, [baseLogs, severityFilter]);

  const visibleLogs = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) return severityFiltered;
    return severityFiltered.filter((entry) => entry.line.toLowerCase().includes(query));
  }, [severityFiltered, searchQuery]);

  const prevVisibleLenRef = useRef(0);

  useEffect(() => {
    const el = containerRef.current;
    const prevLen = prevVisibleLenRef.current;
    const delta = visibleLogs.length - prevLen;
    prevVisibleLenRef.current = visibleLogs.length;

    if (!el) return;

    if (autoFollow) {
      el.scrollTop = el.scrollHeight;
      if (pendingCount !== 0) setPendingCount(0);
    } else if (delta > 0) {
      setPendingCount((count) => count + delta);
    }
    // pendingCount intentionally excluded: it's read, not a trigger.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visibleLogs.length, autoFollow]);

  function handleScroll() {
    const el = containerRef.current;
    if (!el) return;
    const nearBottom =
      el.scrollTop + el.clientHeight >= el.scrollHeight - NEAR_BOTTOM_THRESHOLD_PX;
    setAutoFollow(nearBottom);
    if (nearBottom) setPendingCount(0);
  }

  function handleResumeFollow() {
    setAutoFollow(true);
    setPendingCount(0);
    const el = containerRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }

  function handleClearView() {
    if (logs.length === 0) return;
    setClearedThroughId(logs[logs.length - 1].id);
    setAutoFollow(true);
    setPendingCount(0);
  }

  async function handleCopy() {
    const text = visibleLogs.map((entry) => formatLogLine(entry)).join("\n");
    try {
      await navigator.clipboard.writeText(text);
      setJustCopied(true);
      window.setTimeout(() => setJustCopied(false), 1500);
    } catch {
      // Clipboard access can fail (permissions, insecure context, etc.) —
      // there's nothing else useful to do client-side, so fail silently.
    }
  }

  if (isLoading) return <LoadingState label="Loading logs…" />;

  if (error) {
    return <EmptyState title="Couldn't load logs" description={error.message} />;
  }

  if (logs.length === 0) {
    return (
      <EmptyState
        title="No logs yet"
        description="Start the service to see output here."
      />
    );
  }

  return (
    <div className="flex h-full w-full flex-1 flex-col">
      <div className="flex flex-wrap items-center gap-2 border-b border-border bg-background px-3 py-2">
        <div className="relative min-w-[160px] flex-1">
          <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder="Search logs…"
            className="h-7 w-full rounded-md border border-border bg-background pl-7 pr-2 text-xs outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50"
          />
        </div>

        <div className="flex items-center gap-0.5 rounded-md border border-border p-0.5">
          {(["all", "stdout", "stderr"] as const).map((option) => (
            <button
              key={option}
              type="button"
              aria-pressed={severityFilter === option}
              onClick={() => setSeverityFilter(option)}
              className={cn(
                "rounded-[calc(var(--radius-md)-2px)] px-2 py-1 text-xs capitalize transition-colors",
                severityFilter === option
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {option}
            </button>
          ))}
        </div>

        <Button type="button" variant="outline" size="xs" onClick={handleCopy}>
          <Copy />
          {justCopied ? "Copied!" : "Copy"}
        </Button>

        <Button type="button" variant="outline" size="xs" onClick={handleClearView}>
          <Trash2 />
          Clear view
        </Button>
      </div>

      <div className="relative min-h-0 flex-1">
        <div
          ref={containerRef}
          onScroll={handleScroll}
          className="h-full w-full flex-1 overflow-y-auto bg-background p-3 font-mono text-xs leading-relaxed"
        >
          {visibleLogs.length === 0 ? (
            <p className="text-muted-foreground">No lines match the current filters.</p>
          ) : (
            visibleLogs.map((entry) => <LogLine key={entry.id} entry={entry} />)
          )}
        </div>

        {!autoFollow && pendingCount > 0 && (
          <button
            type="button"
            onClick={handleResumeFollow}
            className="absolute bottom-3 right-3 rounded-full bg-primary px-3 py-1 text-xs font-medium text-primary-foreground shadow-lg transition-opacity hover:opacity-90"
          >
            {pendingCount} new line{pendingCount === 1 ? "" : "s"} ↓
          </button>
        )}
      </div>
    </div>
  );
}

function LogLine({ entry }: { entry: LogEntry }) {
  const errorish = isErrorLine(entry.line);

  return (
    <div
      className={cn(
        "whitespace-pre-wrap break-all",
        entry.source === "stderr" && "text-red-500",
        errorish && "border-l-2 border-red-500 bg-red-500/10 pl-2 -ml-2",
      )}
    >
      <span className="mr-2 text-muted-foreground">
        {new Date(entry.time).toLocaleTimeString()}
      </span>
      {entry.line}
    </div>
  );
}
