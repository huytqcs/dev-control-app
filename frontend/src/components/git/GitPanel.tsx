import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { BranchCheckoutForm } from "@/components/git/BranchCheckoutForm";
import { gitCheckout, gitFetch, gitPull, gitPush } from "@/lib/api";
import { servicesQueryKey } from "@/hooks/useServicesQuery";
import type { GitStateDTO, ServiceDTO } from "@/types/api";

export function GitPanel({ service }: { service: ServiceDTO }) {
  const queryClient = useQueryClient();

  function applyGitState(git: GitStateDTO) {
    queryClient.setQueryData<ServiceDTO[]>(servicesQueryKey, (prev) =>
      prev?.map((s) => (s.id === service.id ? { ...s, state: { ...s.state, git } } : s)),
    );
  }

  const fetchMut = useMutation({ mutationFn: () => gitFetch(service.id), onSuccess: applyGitState });
  const pullMut = useMutation({ mutationFn: () => gitPull(service.id), onSuccess: applyGitState });
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
        pending={checkoutMut.isPending}
        onCheckout={(branch) => checkoutMut.mutate(branch)}
      />

      {error ? (
        <p className="text-xs text-destructive">{(error as Error).message}</p>
      ) : null}
    </div>
  );
}
