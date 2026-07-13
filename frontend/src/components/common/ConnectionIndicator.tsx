import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useConnectionStatus } from "@/hooks/useConnectionStatus";

// A dropped WS used to be silently invisible — data just stopped updating
// with no signal anything was wrong (see useRealtimeEvents' onReconnect
// resync comment). This surfaces that state instead of hiding it.
export function ConnectionIndicator() {
  const status = useConnectionStatus();

  if (status === "connected") return null;

  return (
    <Badge variant="outline" className="gap-1.5 text-amber-600 dark:text-amber-500">
      <span className={cn("size-1.5 rounded-full bg-amber-500 animate-pulse")} />
      Reconnecting…
    </Badge>
  );
}
