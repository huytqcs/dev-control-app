import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const MAX_VISIBLE_MATCHES = 8;

export function BranchCheckoutForm({
  branches,
  currentBranch,
  branchesLoading,
  onCheckout,
  pending,
}: {
  branches: string[];
  currentBranch: string;
  branchesLoading: boolean;
  onCheckout: (branch: string) => void;
  pending: boolean;
}) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    const pool = q ? branches.filter((b) => b.toLowerCase().includes(q)) : branches;
    return pool.slice(0, MAX_VISIBLE_MATCHES);
  }, [branches, query]);

  function submit(branch: string) {
    const trimmed = branch.trim();
    if (!trimmed || trimmed === currentBranch) return;
    onCheckout(trimmed);
    setOpen(false);
  }

  return (
    <div className="relative">
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          submit(query);
        }}
      >
        <input
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => setTimeout(() => setOpen(false), 150)}
          placeholder={branchesLoading ? "loading branches…" : "search branches…"}
          disabled={pending}
          className="h-8 min-w-0 flex-1 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
        <Button type="submit" size="sm" variant="outline" disabled={pending || !query.trim()}>
          Checkout
        </Button>
      </form>

      {open && matches.length > 0 ? (
        <ul className="absolute z-10 mt-1 max-h-48 w-full overflow-y-auto rounded-md border bg-popover text-sm shadow-md">
          {matches.map((b) => (
            <li key={b}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  setQuery(b);
                  submit(b);
                }}
                disabled={b === currentBranch}
                className={cn(
                  "block w-full truncate px-2 py-1 text-left hover:bg-muted disabled:cursor-default disabled:text-muted-foreground disabled:hover:bg-transparent",
                )}
              >
                {b}
                {b === currentBranch ? " (current)" : ""}
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
