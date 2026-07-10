import type { ReactNode } from "react";

export function EmptyState({
  title,
  description,
}: {
  title: string;
  description?: ReactNode;
}) {
  return (
    <div className="flex h-full w-full flex-1 flex-col items-center justify-center gap-1 p-8 text-center text-muted-foreground">
      <p className="text-sm font-medium text-foreground">{title}</p>
      {description ? <p className="text-sm">{description}</p> : null}
    </div>
  );
}
