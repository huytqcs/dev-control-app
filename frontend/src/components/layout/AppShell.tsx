import type { ReactNode } from "react";
import { Sidebar } from "@/components/layout/Sidebar";
import { TopBar } from "@/components/layout/TopBar";
import type { ServiceDTO } from "@/types/api";

export function AppShell({
  workspaceName,
  services,
  children,
}: {
  workspaceName?: string;
  services: ServiceDTO[];
  children: ReactNode;
}) {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground">
      <Sidebar workspaceName={workspaceName} />
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar services={services} />
        <main className="flex min-h-0 flex-1">{children}</main>
      </div>
    </div>
  );
}
