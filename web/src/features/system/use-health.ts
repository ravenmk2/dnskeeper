import { useQuery } from "@tanstack/react-query";

import { checkHealth } from "./api";

export type HealthStatus = "healthy" | "unhealthy" | "checking";

export function useHealth() {
  const query = useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      await checkHealth();
      return true;
    },
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
    retry: false,
    staleTime: 0,
  });

  let status: HealthStatus = "checking";
  if (query.isSuccess) status = "healthy";
  else if (query.isError) status = "unhealthy";

  return { status, data: query.data ?? null, query };
}
