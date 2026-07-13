import type { ReactNode } from "react";
import { Sidebar } from "@/components/layout/Sidebar";
import { TopBar } from "@/components/layout/TopBar";
import type { PresetDTO, ServiceDTO } from "@/types/api";

export function AppShell({
  workspaceName,
  services,
  presets,
  selectedId,
  onSelect,
  children,
}: {
  workspaceName?: string;
  services: ServiceDTO[];
  presets: PresetDTO[];
  selectedId?: string;
  onSelect: (id: string) => void;
  children: ReactNode;
}) {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground">
      <div className="hidden lg:flex">
        <Sidebar
          workspaceName={workspaceName}
          services={services}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar services={services} presets={presets} />
        {/* Below lg, the sidebar (which duplicates this list) is hidden and
            the grid/details pair stacks vertically instead of squeezing into
            a fixed-width 3-pane row that has no room to shrink further. */}
        <main className="flex min-h-0 flex-1 flex-col lg:flex-row">{children}</main>
      </div>
    </div>
  );
}
