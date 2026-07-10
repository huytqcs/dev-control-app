import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { LogViewer } from "@/components/logs/LogViewer";
import { EmptyState } from "@/components/common/EmptyState";
import { StatusBadge } from "@/components/common/StatusBadge";
import type { ServiceDTO } from "@/types/api";

export function ServiceDetailsPanel({
  service,
}: {
  service: ServiceDTO | undefined;
}) {
  if (!service) {
    return (
      <div className="flex h-full min-w-0 flex-col border-l">
        <EmptyState
          title="No service selected"
          description="Pick a service from the grid to see its logs and details."
        />
      </div>
    );
  }

  return (
    <div className="flex h-full min-w-0 flex-col border-l">
      <div className="flex items-center justify-between border-b p-3">
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold">{service.name}</div>
          <div className="truncate text-xs text-muted-foreground">
            {service.path}
          </div>
        </div>
        <StatusBadge status={service.state.status} />
      </div>

      <Tabs defaultValue="logs" className="flex min-h-0 flex-1 flex-col">
        <TabsList className="mx-3 mt-2 self-start">
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="info">Info</TabsTrigger>
        </TabsList>

        <TabsContent value="logs" className="flex min-h-0 flex-1 flex-col">
          <LogViewer serviceId={service.id} />
        </TabsContent>

        <TabsContent value="info" className="min-h-0 flex-1 overflow-y-auto p-4 text-sm">
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
            <dt className="text-muted-foreground">Type</dt>
            <dd>{service.type}</dd>
            <dt className="text-muted-foreground">Path</dt>
            <dd className="break-all">{service.path}</dd>
            <dt className="text-muted-foreground">Port</dt>
            <dd>{service.port || "—"}</dd>
            <dt className="text-muted-foreground">Depends on</dt>
            <dd>
              {service.dependsOn.length ? service.dependsOn.join(", ") : "—"}
            </dd>
            <dt className="text-muted-foreground">PID</dt>
            <dd>{service.state.pid ?? "—"}</dd>
            <dt className="text-muted-foreground">Last exit code</dt>
            <dd>{service.state.lastExitCode ?? "—"}</dd>
          </dl>
        </TabsContent>
      </Tabs>
    </div>
  );
}
