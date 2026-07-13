import { STATUS_DOT, STATUS_LABEL } from "@/components/common/StatusBadge";
import { cn } from "@/lib/utils";
import type { ServiceDTO } from "@/types/api";

export function Sidebar({
  workspaceName,
  services,
  selectedId,
  onSelect,
}: {
  workspaceName?: string;
  services: ServiceDTO[];
  selectedId?: string;
  onSelect: (id: string) => void;
}) {
  return (
    <aside className="flex w-56 shrink-0 flex-col border-r bg-muted/30 p-4">
      <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        Workspace
      </div>
      <div className="mt-1 text-sm font-semibold">
        {workspaceName ?? "—"}
      </div>

      <div className="mt-6 text-xs font-medium tracking-wide text-muted-foreground uppercase">
        Services
      </div>

      <nav className="-mx-1 mt-1 flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto">
        {services.length === 0 ? (
          <p className="mt-2 px-1 text-xs text-muted-foreground">
            No services configured.
          </p>
        ) : (
          services.map((service) => {
            const isSelected = service.id === selectedId;
            const status = service.state.status;

            return (
              <button
                key={service.id}
                type="button"
                onClick={() => onSelect(service.id)}
                className={cn(
                  "flex items-center gap-2 rounded-r-md border-l-2 border-transparent px-2 py-1.5 text-left text-sm transition-colors hover:bg-muted/50",
                  isSelected
                    ? "border-l-primary bg-primary/10 font-medium text-foreground"
                    : "text-muted-foreground",
                )}
              >
                <span
                  title={STATUS_LABEL[status]}
                  className={cn(
                    "size-1.5 shrink-0 rounded-full",
                    STATUS_DOT[status],
                    (status === "starting" || status === "stopping") && "animate-pulse",
                  )}
                />
                <span className="truncate">{service.name}</span>
              </button>
            );
          })
        )}
      </nav>
    </aside>
  );
}
