import { useQuery } from "@tanstack/react-query";
import { getServices } from "@/lib/api";

export const servicesQueryKey = ["services"] as const;

export function useServicesQuery() {
  return useQuery({
    queryKey: servicesQueryKey,
    queryFn: getServices,
    // service.updated events keep this fresh; a slow background refetch is
    // just a safety net against a missed/dropped WS event.
    refetchInterval: 30_000,
  });
}
