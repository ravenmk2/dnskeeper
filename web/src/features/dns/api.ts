import { apiCall } from "@/lib/api";
import type {
  CreateRecordRequest,
  Domain,
  Record as DnsRecord,
  UpdateRecordRequest,
  Zone,
} from "@/types/api";

// ── Zone ──────────────────────────────────────────────
export function listZones(): Promise<Zone[]> {
  return apiCall<Zone[]>("dns/zone/list");
}
export function createZone(zone: string): Promise<Zone> {
  return apiCall<Zone>("dns/zone/create", { zone });
}
export function updateZone(zone: string): Promise<Zone> {
  return apiCall<Zone>("dns/zone/update", { zone });
}
export function deleteZone(zone: string): Promise<void> {
  return apiCall<void>("dns/zone/delete", { zone });
}

// ── Domain ────────────────────────────────────────────
export function listDomains(zone: string): Promise<Domain[]> {
  return apiCall<Domain[]>("dns/domain/list", { zone });
}
export function createDomain(zone: string, domain: string): Promise<Domain> {
  return apiCall<Domain>("dns/domain/create", { zone, domain });
}
export function updateDomain(
  zone: string,
  domain: string,
): Promise<Domain> {
  return apiCall<Domain>("dns/domain/update", { zone, domain });
}
export function deleteDomain(zone: string, domain: string): Promise<void> {
  return apiCall<void>("dns/domain/delete", { zone, domain });
}

// ── Record ────────────────────────────────────────────
export function listRecords(
  zone: string,
  domain: string,
): Promise<DnsRecord[]> {
  return apiCall<DnsRecord[]>("dns/record/list", { zone, domain });
}
export function createRecord(req: CreateRecordRequest): Promise<DnsRecord> {
  return apiCall<DnsRecord>("dns/record/create", req);
}
export function updateRecord(req: UpdateRecordRequest): Promise<DnsRecord> {
  return apiCall<DnsRecord>("dns/record/update", req);
}
export function deleteRecord(
  zone: string,
  domain: string,
  id: string,
): Promise<void> {
  return apiCall<void>("dns/record/delete", { zone, domain, id });
}
