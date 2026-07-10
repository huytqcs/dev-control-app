import { useQuery } from "@tanstack/react-query";
import { getWorkspace } from "@/lib/api";

export function useWorkspaceQuery() {
  return useQuery({ queryKey: ["workspace"], queryFn: getWorkspace });
}
