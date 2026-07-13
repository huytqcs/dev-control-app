import { useEffect, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LogViewer } from "@/components/logs/LogViewer";
import { EmptyState } from "@/components/common/EmptyState";
import { StatusBadge } from "@/components/common/StatusBadge";
import { HealthBadge } from "@/components/health/HealthBadge";
import { GitPanel } from "@/components/git/GitPanel";
import { WorkerControls } from "@/components/workspace/WorkerControls";
import { ActionsPanel } from "@/components/actions/ActionsPanel";
import { forceKillService, openBrowser, openRepo, openTerminal } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import { cn } from "@/lib/utils";
import type { ServiceDTO } from "@/types/api";

export function ServiceDetailsPanel({
  service,
}: {
  service: ServiceDTO | undefined;
}) {
  const queryClient = useQueryClient();
  const forceKill = useMutation({
    mutationFn: () => forceKillService(service!.id),
    onSuccess: (updated) =>
      queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
        prev?.map((s) => (s.id === updated.id ? updated : s)),
      ),
  });
  const openBrowserMut = useMutation({ mutationFn: () => openBrowser(service!.id) });
  const openRepoMut = useMutation({ mutationFn: () => openRepo(service!.id) });
  const openTerminalMut = useMutation({ mutationFn: () => openTerminal(service!.id) });
  const [errorExpanded, setErrorExpanded] = useState(false);
  // Reset on service switch — otherwise an expanded error stays expanded
  // when selecting into a different, unrelated service's error.
  useEffect(() => setErrorExpanded(false), [service?.id]);

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
        <div className="flex items-center gap-1.5">
          {service.openUrls.length > 0 ? (
            <Button
              size="sm"
              variant="ghost"
              disabled={openBrowserMut.isPending}
              onClick={() => openBrowserMut.mutate()}
              title={`Open ${service.openUrls[0]}`}
            >
              Open
            </Button>
          ) : null}
          <Button
            size="sm"
            variant="ghost"
            disabled={openRepoMut.isPending}
            onClick={() => openRepoMut.mutate()}
            title="Open repo folder in Finder"
          >
            Finder
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={openTerminalMut.isPending}
            onClick={() => openTerminalMut.mutate()}
            title="Open Terminal at repo path"
          >
            Terminal
          </Button>
          <HealthBadge status={service.state.health.status} serviceStatus={service.state.status} />
          <StatusBadge status={service.state.status} />
        </div>
      </div>

      <Tabs defaultValue="logs" className="flex min-h-0 flex-1 flex-col">
        <TabsList className="mx-3 mt-2 self-start">
          <TabsTrigger value="logs">Logs</TabsTrigger>
          <TabsTrigger value="git">Git</TabsTrigger>
          <TabsTrigger value="workers">Workers</TabsTrigger>
          <TabsTrigger value="actions">Actions</TabsTrigger>
          <TabsTrigger value="info">Info</TabsTrigger>
        </TabsList>

        <TabsContent value="logs" className="flex min-h-0 flex-1 flex-col">
          <LogViewer serviceId={service.id} />
        </TabsContent>

        <TabsContent value="git" className="min-h-0 flex-1 overflow-y-auto">
          <GitPanel service={service} />
        </TabsContent>

        <TabsContent value="workers" className="min-h-0 flex-1 overflow-y-auto p-4">
          <WorkerControls service={service} />
        </TabsContent>

        <TabsContent value="actions" className="min-h-0 flex-1 overflow-y-auto p-4">
          <ActionsPanel key={service.id} service={service} />
        </TabsContent>

        <TabsContent value="info" className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 text-sm">
          <div className="rounded-md border p-3">
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
              <dt className="text-muted-foreground">Type</dt>
              <dd>{service.type}</dd>
              <dt className="text-muted-foreground">Path</dt>
              <dd className="break-all">{service.path}</dd>
              <dt className="text-muted-foreground">Port</dt>
              <dd>{service.port || "—"}</dd>
              <dt className="text-muted-foreground">PID</dt>
              <dd>{service.state.pid ?? "—"}</dd>
              <dt className="text-muted-foreground">Last exit code</dt>
              <dd>{service.state.lastExitCode ?? "—"}</dd>
            </dl>
          </div>

          <div>
            <div className="mb-1.5 text-xs font-medium text-muted-foreground">
              Depends on
            </div>
            {service.dependsOn.length ? (
              <div className="flex flex-wrap gap-1.5">
                {service.dependsOn.map((dep) => (
                  <Badge key={dep} variant="outline">
                    {dep}
                  </Badge>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground">None</p>
            )}
          </div>

          {service.state.lastError ? (
            <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3">
              <div className="mb-1 text-xs font-medium text-destructive">
                Last error
              </div>
              <p
                className={cn(
                  "font-mono text-xs whitespace-pre-wrap text-destructive",
                  !errorExpanded && "line-clamp-3",
                )}
              >
                {service.state.lastError}
              </p>
              <button
                type="button"
                onClick={() => setErrorExpanded((v) => !v)}
                className="mt-1 text-xs text-muted-foreground underline hover:text-foreground"
              >
                {errorExpanded ? "Show less" : "Show more"}
              </button>
            </div>
          ) : null}

          <div className="mt-auto border-t pt-4">
            <Button
              size="sm"
              variant="destructive"
              disabled={forceKill.isPending}
              onClick={() => forceKill.mutate()}
            >
              Force kill
            </Button>
            <p className="mt-2 text-xs text-muted-foreground">
              Kills whatever is listening on port {service.port || "—"},
              regardless of whether devctl started it. Use this if a service
              survived a devctl restart and shows the wrong state.
            </p>
            {forceKill.isError ? (
              <p className="mt-1 text-xs text-destructive">
                {(forceKill.error as Error).message}
              </p>
            ) : null}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
