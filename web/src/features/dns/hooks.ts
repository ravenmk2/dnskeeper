import { useQuery } from "@tanstack/react-query";

import { listDomains, listRecords, listZones } from "./api";

export const zoneKeys = {
  all: ["zones"] as const,
};

export function useZones() {
  return useQuery({ queryKey: zoneKeys.all, queryFn: listZones });
}

export function useDomains(zone: string | undefined) {
  return useQuery({
    queryKey: ["domains", zone],
    queryFn: () => listDomains(zone!),
    enabled: !!zone,
  });
}

export function useRecords(
  zone: string | undefined,
  domain: string | undefined,
) {
  return useQuery({
    queryKey: ["records", zone, domain],
    queryFn: () => listRecords(zone!, domain!),
    enabled: !!zone && !!domain,
  });
}

export interface DnsStats {
  zoneCount: number;
  domainCount: number;
  recordCount: number;
  zones: Awaited<ReturnType<typeof listZones>>;
}

/**
 * 客户端聚合统计:并行拉取所有 zone + 每个 zone 的 domains,
 * 汇总 zone/domain/record 计数。数据量小,无上限保护(按既定方案)。
 */
export function useStats() {
  return useQuery({
    queryKey: ["stats"],
    queryFn: async (): Promise<DnsStats> => {
      const zones = await listZones();
      const domainsPerZone = await Promise.all(
        zones.map((z) => listDomains(z.zone).catch(() => [])),
      );
      const allDomains = domainsPerZone.flat();
      const recordCount = allDomains.reduce(
        (sum, d) => sum + d.record_count,
        0,
      );
      return {
        zoneCount: zones.length,
        domainCount: allDomains.length,
        recordCount,
        zones,
      };
    },
  });
}
