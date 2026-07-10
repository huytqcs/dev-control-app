export function Sidebar({ workspaceName }: { workspaceName?: string }) {
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
    </aside>
  );
}
