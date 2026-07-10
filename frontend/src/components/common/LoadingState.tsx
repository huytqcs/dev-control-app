export function LoadingState({ label = "Loading…" }: { label?: string }) {
  return (
    <div className="flex h-full w-full flex-1 items-center justify-center p-8 text-sm text-muted-foreground">
      {label}
    </div>
  );
}
