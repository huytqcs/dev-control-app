import { PresetBar } from "@/components/workspace/PresetBar";
import type { PresetDTO, ServiceDTO } from "@/types/api";

export function TopBar({
  services,
  presets,
}: {
  services: ServiceDTO[];
  presets: PresetDTO[];
}) {
  const running = services.filter((s) => s.state.status === "running").length;
  const failed = services.filter((s) => s.state.status === "failed").length;

  return (
    <header className="flex h-14 shrink-0 items-center gap-4 border-b px-4">
      <h1 className="text-sm font-semibold">devctl</h1>
      <div className="flex items-center gap-3 text-sm text-muted-foreground">
        <span>
          Running <span className="font-medium text-foreground">{running}</span>
        </span>
        {failed > 0 ? (
          <span className="text-destructive">
            Failed <span className="font-medium">{failed}</span>
          </span>
        ) : null}
        <span>
          Total <span className="font-medium text-foreground">{services.length}</span>
        </span>
      </div>
      <div className="ml-auto">
        <PresetBar presets={presets} />
      </div>
    </header>
  );
}
