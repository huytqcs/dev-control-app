import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { BranchCheckoutForm } from "@/components/git/BranchCheckoutForm";
import { gitCheckout, gitFetch, gitListBranches, gitPull, gitPush } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { GitStateDTO, ServiceDTO } from "@/types/api";

function branchesQueryKey(serviceId: string) {
  return ["git-branches", serviceId] as const;
}

export function GitPanel({ service }: { service: ServiceDTO }) {
  const queryClient = useQueryClient();

  const branchesQuery = useQuery({
    queryKey: branchesQueryKey(service.id),
    queryFn: () => gitListBranches(service.id),
  });

  function applyGitState(git: GitStateDTO) {
    queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
      prev?.map((s) => (s.id === service.id ? { ...s, state: { ...s.state, git } } : s)),
    );
  }

  // Fetch/pull can surface branches that didn't exist locally before —
  // refresh the branch list alongside git state instead of leaving it stale
  // until the panel happens to remount.
  function refreshBranches() {
    queryClient.invalidateQueries({ queryKey: branchesQueryKey(service.id) });
  }

  const fetchMut = useMutation({
    mutationFn: () => gitFetch(service.id),
    onSuccess: (git) => {
      applyGitState(git);
      refreshBranches();
    },
  });
  const pullMut = useMutation({
    mutationFn: () => gitPull(service.id),
    onSuccess: (git) => {
      applyGitState(git);
      refreshBranches();
    },
  });
  const pushMut = useMutation({ mutationFn: () => gitPush(service.id), onSuccess: applyGitState });
  const checkoutMut = useMutation({
    mutationFn: (branch: string) => gitCheckout(service.id, branch),
    onSuccess: applyGitState,
  });

  const pending =
    fetchMut.isPending || pullMut.isPending || pushMut.isPending || checkoutMut.isPending;
  const error = fetchMut.error ?? pullMut.error ?? pushMut.error ?? checkoutMut.error;
  const git = service.state.git;

  return (
    <div className="flex flex-col gap-4 p-4 text-sm">
      <div className="flex items-center justify-between">
        <div>
          <div className="font-medium">{git.branch || "—"}</div>
          <div className="text-xs text-muted-foreground">
            {git.dirty ? "Uncommitted changes" : "Clean"}
            {(git.ahead > 0 || git.behind > 0) && (
              <span>
                {" · "}
                {git.ahead > 0 ? `↑${git.ahead}` : null}
                {git.ahead > 0 && git.behind > 0 ? " " : null}
                {git.behind > 0 ? `↓${git.behind}` : null}
              </span>
            )}
          </div>
        </div>
        {git.dirty ? (
          <span className="size-2 shrink-0 rounded-full bg-amber-500" title="Dirty" />
        ) : null}
      </div>

      <div className="flex gap-2">
        <Button size="sm" variant="outline" disabled={pending} onClick={() => fetchMut.mutate()}>
          Fetch
        </Button>
        <Button size="sm" variant="outline" disabled={pending} onClick={() => pullMut.mutate()}>
          Pull
        </Button>
        <Button size="sm" variant="outline" disabled={pending} onClick={() => pushMut.mutate()}>
          Push
        </Button>
      </div>

      <BranchCheckoutForm
        branches={branchesQuery.data ?? []}
        branchesLoading={branchesQuery.isLoading}
        currentBranch={git.branch}
        pending={checkoutMut.isPending}
        onCheckout={(branch) => checkoutMut.mutate(branch)}
      />

      {error ? (
        <p className="text-xs text-destructive">{(error as Error).message}</p>
      ) : branchesQuery.isError ? (
        <p className="text-xs text-destructive">
          {(branchesQuery.error as Error).message}
        </p>
      ) : null}
    </div>
  );
}
