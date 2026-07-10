import type { ReactNode } from "react";
import { Sidebar } from "@/components/layout/Sidebar";
import { TopBar } from "@/components/layout/TopBar";
import type { PresetDTO, ServiceDTO } from "@/types/api";

export function AppShell({
  workspaceName,
  services,
  presets,
  children,
}: {
  workspaceName?: string;
  services: ServiceDTO[];
  presets: PresetDTO[];
  children: ReactNode;
}) {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground">
      <Sidebar workspaceName={workspaceName} />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar services={services} presets={presets} />
        <main className="flex min-h-0 flex-1">{children}</main>
      </div>
    </div>
  );
}
