import { useMemo, useState } from "react";
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
  onCheckout: (branch: string, createFrom?: string) => void;
  pending: boolean;
}) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);

  const trimmedQuery = query.trim();
  const matches = useMemo(() => {
    const q = trimmedQuery.toLowerCase();
    if (!q) return branches.slice(0, MAX_VISIBLE_MATCHES);

    // Rank exact match, then prefix match, ahead of "contains" matches —
    // otherwise an alphabetically-early branch that merely contains the
    // query (e.g. "chore/production-deploy") fills the MAX_VISIBLE_MATCHES
    // slots and pushes an exact match like "production" out of the list
    // entirely, even though it passed the filter.
    const rank = (b: string) => {
      const lower = b.toLowerCase();
      if (lower === q) return 0;
      if (lower.startsWith(q)) return 1;
      return 2;
    };

    return branches
      .filter((b) => b.toLowerCase().includes(q))
      .sort((a, b) => rank(a) - rank(b) || a.localeCompare(b))
      .slice(0, MAX_VISIBLE_MATCHES);
  }, [branches, trimmedQuery]);

  // No exact match for the typed name — offer to branch it off HEAD instead
  // of just dead-ending the search (Plan.md §C "create branch from main").
  const canCreate =
    !branchesLoading &&
    trimmedQuery !== "" &&
    !branches.some((b) => b.toLowerCase() === trimmedQuery.toLowerCase());

  function submit(branch: string) {
    const trimmed = branch.trim();
    if (!trimmed || trimmed === currentBranch) return;
    onCheckout(trimmed);
    setOpen(false);
  }

  function submitCreate() {
    if (!trimmedQuery || !canCreate) return;
    onCheckout(trimmedQuery, currentBranch);
    setOpen(false);
  }

  return (
    <div className="relative">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          // Enter acts on the top dropdown row — same thing clicking it
          // does — instead of a separate "Checkout" button re-submitting
          // whatever's typed as if it were its own path.
          if (matches.length > 0) {
            submit(matches[0]);
          } else if (canCreate) {
            submitCreate();
          }
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
          className="h-8 w-full min-w-0 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </form>

      {open && (matches.length > 0 || canCreate) ? (
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
          {canCreate ? (
            <li className={matches.length > 0 ? "border-t" : undefined}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onClick={submitCreate}
                className="block w-full truncate px-2 py-1 text-left text-primary hover:bg-muted"
              >
                Create &ldquo;{trimmedQuery}&rdquo; from {currentBranch || "current branch"}
              </button>
            </li>
          ) : null}
        </ul>
      ) : null}
    </div>
  );
}
