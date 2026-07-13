import { useEffect, useMemo, useState } from "react";
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

  // Keyboard-navigable highlight through the dropdown (branch rows, then the
  // "create" row last if present) — previously Enter always acted on
  // matches[0] with no way to arrow down to another result.
  const totalRows = matches.length + (canCreate ? 1 : 0);
  const [highlightedIndex, setHighlightedIndex] = useState(0);

  useEffect(() => {
    setHighlightedIndex(0);
  }, [trimmedQuery, matches.length, canCreate]);

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
          if (highlightedIndex < matches.length) {
            const b = matches[highlightedIndex];
            if (b) submit(b);
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
          onKeyDown={(e) => {
            if (e.key === "ArrowDown") {
              e.preventDefault();
              setOpen(true);
              setHighlightedIndex((i) => Math.min(i + 1, Math.max(totalRows - 1, 0)));
            } else if (e.key === "ArrowUp") {
              e.preventDefault();
              setHighlightedIndex((i) => Math.max(i - 1, 0));
            } else if (e.key === "Escape") {
              setOpen(false);
            }
          }}
          placeholder={branchesLoading ? "loading branches…" : "search branches…"}
          disabled={pending}
          className="h-8 w-full min-w-0 rounded-md border bg-background px-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </form>

      {open && (matches.length > 0 || canCreate) ? (
        <ul className="absolute z-10 mt-1 max-h-48 w-full overflow-y-auto rounded-md border bg-popover text-sm shadow-md">
          {matches.map((b, i) => (
            <li key={b}>
              <button
                type="button"
                onMouseDown={(e) => e.preventDefault()}
                onMouseEnter={() => setHighlightedIndex(i)}
                onClick={() => {
                  setQuery(b);
                  submit(b);
                }}
                disabled={b === currentBranch}
                className={cn(
                  "block w-full truncate px-2 py-1 text-left hover:bg-muted disabled:cursor-default disabled:text-muted-foreground disabled:hover:bg-transparent",
                  i === highlightedIndex && "bg-muted",
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
                onMouseEnter={() => setHighlightedIndex(matches.length)}
                onClick={submitCreate}
                className={cn(
                  "block w-full truncate px-2 py-1 text-left text-primary hover:bg-muted",
                  highlightedIndex === matches.length && "bg-muted",
                )}
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
