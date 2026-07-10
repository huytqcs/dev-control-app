import { useState } from "react";
import { useWorkspaceQuery } from "@/hooks/useWorkspaceQuery";
import { useServicesQuery } from "@/hooks/useServicesQuery";
import { useRealtimeEvents } from "@/hooks/useRealtimeEvents";
import { AppShell } from "@/components/layout/AppShell";
import { ServiceGrid } from "@/components/workspace/ServiceGrid";
import { ServiceDetailsPanel } from "@/components/workspace/ServiceDetailsPanel";
import { LoadingState } from "@/components/common/LoadingState";
import { EmptyState } from "@/components/common/EmptyState";

export function DashboardPage() {
  useRealtimeEvents();

  const workspaceQuery = useWorkspaceQuery();
  const servicesQuery = useServicesQuery();
  const [selectedId, setSelectedId] = useState<string>();

  const services = servicesQuery.data ?? [];
  const selectedService = services.find((s) => s.id === selectedId);

  return (
    <AppShell workspaceName={workspaceQuery.data?.name} services={services}>
      {servicesQuery.isLoading ? (
        <LoadingState label="Loading services…" />
      ) : servicesQuery.isError ? (
        <EmptyState
          title="Couldn't load services"
          description={(servicesQuery.error as Error).message}
        />
      ) : (
        <>
          <ServiceGrid
            services={services}
            selectedId={selectedId}
            onSelect={setSelectedId}
          />
          <div className="h-full w-[420px] shrink-0">
            <ServiceDetailsPanel service={selectedService} />
          </div>
        </>
      )}
    </AppShell>
  );
}
