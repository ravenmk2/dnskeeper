import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { ApiError } from "@/lib/errors";

import {
  createDomain,
  createRecord,
  createZone,
  deleteDomain,
  deleteRecord,
  deleteZone,
  updateRecord,
} from "./api";

function onMutateError(err: unknown) {
  const e = err as ApiError;
  toast.error(e.message || "操作失败");
}

// ── Zone ──────────────────────────────────────────────
export function useCreateZone() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (zone: string) => createZone(zone),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["zones"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
    onError: onMutateError,
  });
}

export function useDeleteZone() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (zone: string) => deleteZone(zone),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["zones"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
      toast.success("Zone 已删除");
    },
    onError: onMutateError,
  });
}

// ── Domain ────────────────────────────────────────────
export function useCreateDomain() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ zone, domain }: { zone: string; domain: string }) =>
      createDomain(zone, domain),
    onSuccess: (_d, { zone }) => {
      qc.invalidateQueries({ queryKey: ["domains", zone] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
    onError: onMutateError,
  });
}

export function useDeleteDomain() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ zone, domain }: { zone: string; domain: string }) =>
      deleteDomain(zone, domain),
    onSuccess: (_d, { zone }) => {
      qc.invalidateQueries({ queryKey: ["domains", zone] });
      qc.invalidateQueries({ queryKey: ["stats"] });
      toast.success("Domain 已删除");
    },
    onError: onMutateError,
  });
}

// ── Record ────────────────────────────────────────────
export function useCreateRecord() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createRecord,
    onSuccess: (_r, { zone, domain }) => {
      qc.invalidateQueries({ queryKey: ["records", zone, domain] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
    onError: onMutateError,
  });
}

export function useUpdateRecord() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: updateRecord,
    onSuccess: (_r, { zone, domain }) => {
      qc.invalidateQueries({ queryKey: ["records", zone, domain] });
    },
    onError: onMutateError,
  });
}

export function useDeleteRecord() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      zone,
      domain,
      id,
    }: {
      zone: string;
      domain: string;
      id: string;
    }) => deleteRecord(zone, domain, id),
    onSuccess: (_d, { zone, domain }) => {
      qc.invalidateQueries({ queryKey: ["records", zone, domain] });
      qc.invalidateQueries({ queryKey: ["stats"] });
      toast.success("Record 已删除");
    },
    onError: onMutateError,
  });
}
