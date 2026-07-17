import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useParams } from "react-router-dom";
import { z } from "zod";
import { AlertCircle, FileText, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Breadcrumb } from "@/components/breadcrumb";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PageHeader } from "@/components/page-header";
import { RecordTypeBadge } from "@/components/record-type-badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useRecords } from "@/features/dns/hooks";
import {
  useCreateRecord,
  useDeleteRecord,
  useUpdateRecord,
} from "@/features/dns/mutations";
import { ApiError } from "@/lib/errors";
import type { Record as DnsRecord, RecordType } from "@/types/api";

const IPV4_RE =
  /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
const IPV6_RE =
  /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:))$/;
const FQDN_RE =
  /^([a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.?$/i;

const numOrUndef = (v: string) =>
  v === "" ? undefined : Number.isNaN(Number(v)) ? undefined : Number(v);

const recordSchema = z
  .object({
    type: z.enum(["A", "AAAA", "SRV", "TXT"]),
    value: z.string().min(1, "不能为空"),
    ttl: z
      .number({ message: "必须为数字" })
      .int("必须为整数")
      .min(1, "范围 1-86400")
      .max(86400, "范围 1-86400"),
    priority: z
      .number()
      .int()
      .min(0, "0-65535")
      .max(65535, "0-65535")
      .optional(),
    port: z
      .number()
      .int()
      .min(0, "0-65535")
      .max(65535, "0-65535")
      .optional(),
    weight: z
      .number()
      .int()
      .min(0, "0-65535")
      .max(65535, "0-65535")
      .optional(),
  })
  .superRefine((d, ctx) => {
    if (d.type === "A" && !IPV4_RE.test(d.value))
      ctx.addIssue({ path: ["value"], code: "custom", message: "无效 IPv4 地址" });
    if (d.type === "AAAA" && !IPV6_RE.test(d.value))
      ctx.addIssue({ path: ["value"], code: "custom", message: "无效 IPv6 地址" });
    if (d.type === "SRV") {
      if (!FQDN_RE.test(d.value))
        ctx.addIssue({
          path: ["value"],
          code: "custom",
          message: "无效目标主机名(FQDN)",
        });
      if (d.priority === undefined)
        ctx.addIssue({ path: ["priority"], code: "custom", message: "SRV 必填" });
      if (d.port === undefined)
        ctx.addIssue({ path: ["port"], code: "custom", message: "SRV 必填" });
    }
    if (d.type === "TXT" && d.value.length > 255)
      ctx.addIssue({
        path: ["value"],
        code: "custom",
        message: "TXT 最长 255 字节",
      });
  });

type RecordValues = z.infer<typeof recordSchema>;

const TYPES: RecordType[] = ["A", "AAAA", "SRV", "TXT"];

function valuePlaceholder(type: RecordType): string {
  switch (type) {
    case "A":
      return "1.2.3.4";
    case "AAAA":
      return "2001:db8::1";
    case "SRV":
      return "target.example.com";
    case "TXT":
      return "v=spf1 ...";
  }
}

function RecordFormDialog({
  zone,
  domain,
  record,
  open,
  onOpenChange,
}: {
  zone: string;
  domain: string;
  record: DnsRecord | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const isEdit = !!record;
  const createMut = useCreateRecord();
  const updateMut = useUpdateRecord();

  const form = useForm<RecordValues>({
    resolver: zodResolver(recordSchema),
    defaultValues: { type: "A", value: "", ttl: 300, priority: undefined, port: undefined, weight: undefined },
  });
  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = form;

  const type = watch("type");

  useEffect(() => {
    if (!open) return;
    if (record) {
      reset({
        type: record.type,
        value: record.value,
        ttl: record.ttl,
        priority: record.priority,
        port: record.port,
        weight: record.weight,
      });
    } else {
      reset({
        type: "A",
        value: "",
        ttl: 300,
        priority: undefined,
        port: undefined,
        weight: undefined,
      });
    }
  }, [open, record, reset]);

  const onSubmit = (vals: RecordValues) => {
    if (isEdit && record) {
      updateMut.mutate(
        {
          zone,
          domain,
          id: record.id,
          value: vals.value,
          ttl: vals.ttl,
          priority: vals.type === "SRV" ? vals.priority : undefined,
          port: vals.type === "SRV" ? vals.port : undefined,
          weight: vals.type === "SRV" ? vals.weight : undefined,
        },
        {
          onSuccess: () => {
            toast.success("Record 已更新");
            onOpenChange(false);
          },
        },
      );
    } else {
      createMut.mutate(
        {
          zone,
          domain,
          type: vals.type,
          value: vals.value,
          ttl: vals.ttl,
          priority: vals.type === "SRV" ? vals.priority : undefined,
          port: vals.type === "SRV" ? vals.port : undefined,
          weight: vals.type === "SRV" ? vals.weight : undefined,
        },
        {
          onSuccess: () => {
            toast.success("Record 已创建");
            onOpenChange(false);
          },
          onError: (err: unknown) => {
            const e = err as ApiError;
            if (e.code === "VALIDATION_ERROR") {
              const fe = e.fieldErrors();
              for (const [field, msg] of Object.entries(fe)) {
                if (field in vals)
                  form.setError(field as keyof RecordValues, { message: msg });
              }
            }
          },
        },
      );
    }
  };

  const pending = createMut.isPending || updateMut.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "编辑 Record" : "新建 Record"}</DialogTitle>
          <DialogDescription>
            <span className="font-mono text-foreground">
              {domain === "@" ? zone : `${domain}.${zone}`}
            </span>{" "}
            下的 DNS 记录
            {isEdit && (
              <>
                ,ID{" "}
                <span className="font-mono text-foreground">{record?.id}</span>
              </>
            )}
          </DialogDescription>
        </DialogHeader>
        <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <Field orientation="horizontal" className="items-center">
              <FieldLabel className="w-16 shrink-0">类型</FieldLabel>
              <select
                className="h-8 w-full rounded-lg border border-input bg-transparent px-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30"
                aria-label="Record 类型"
                disabled={isEdit}
                value={type}
                onChange={(e) =>
                  setValue("type", e.target.value as RecordType, {
                    shouldValidate: false,
                  })
                }
              >
                {TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>

            <Field>
              <FieldLabel htmlFor="value">值</FieldLabel>
              <Input
                id="value"
                placeholder={valuePlaceholder(type)}
                autoComplete="off"
                aria-invalid={!!errors.value}
                {...register("value")}
              />
              <FieldError errors={[errors.value]} />
            </Field>

            {type === "SRV" && (
              <Field orientation="horizontal">
                <FieldLabel htmlFor="priority" className="w-16 shrink-0">
                  优先级
                </FieldLabel>
                <Input
                  id="priority"
                  type="number"
                  min={0}
                  max={65535}
                  placeholder="10"
                  className="w-24"
                  aria-invalid={!!errors.priority}
                  {...register("priority", { setValueAs: numOrUndef })}
                />
                <FieldError errors={[errors.priority]} />
              </Field>
            )}
            {type === "SRV" && (
              <Field orientation="horizontal">
                <FieldLabel htmlFor="port" className="w-16 shrink-0">
                  端口
                </FieldLabel>
                <Input
                  id="port"
                  type="number"
                  min={0}
                  max={65535}
                  placeholder="5080"
                  className="w-24"
                  aria-invalid={!!errors.port}
                  {...register("port", { setValueAs: numOrUndef })}
                />
                <FieldError errors={[errors.port]} />
              </Field>
            )}
            {type === "SRV" && (
              <Field orientation="horizontal">
                <FieldLabel htmlFor="weight" className="w-16 shrink-0">
                  权重
                </FieldLabel>
                <Input
                  id="weight"
                  type="number"
                  min={0}
                  max={65535}
                  placeholder="0"
                  className="w-24"
                  aria-invalid={!!errors.weight}
                  {...register("weight", { setValueAs: numOrUndef })}
                />
                <FieldError errors={[errors.weight]} />
              </Field>
            )}

            <Field orientation="horizontal">
              <FieldLabel htmlFor="ttl" className="w-16 shrink-0">
                TTL
              </FieldLabel>
              <Input
                id="ttl"
                type="number"
                min={1}
                max={86400}
                placeholder="300"
                className="w-28"
                aria-invalid={!!errors.ttl}
                {...register("ttl", { setValueAs: numOrUndef })}
              />
              <FieldError errors={[errors.ttl]} />
              <span className="self-center text-xs text-muted-foreground">
                秒(1-86400)
              </span>
            </Field>
          </FieldGroup>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={pending}>
              {isEdit ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border py-16 text-center">
      <FileText className="size-8 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-sm font-medium text-foreground">还没有 Record</p>
        <p className="mt-1 text-xs text-muted-foreground">
          创建第一条 DNS 记录(A / AAAA / SRV / TXT)
        </p>
      </div>
      <Button onClick={onCreate} size="sm">
        <Plus className="size-4" />
        新建 Record
      </Button>
    </div>
  );
}

export default function Records() {
  const { zone = "", domain = "" } = useParams<{
    zone: string;
    domain: string;
  }>();
  const { data: records, isPending, isError, error } = useRecords(zone, domain);
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<DnsRecord | null>(null);
  const [delRec, setDelRec] = useState<DnsRecord | null>(null);
  const deleteMut = useDeleteRecord();

  const fqdn = domain === "@" ? zone : `${domain}.${zone}`;

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };
  const openEdit = (r: DnsRecord) => {
    setEditing(r);
    setFormOpen(true);
  };

  return (
    <div>
      <Breadcrumb
        items={[
          { label: "DNS 管理", to: "/dns/zones" },
          { label: zone, to: `/dns/zones/${encodeURIComponent(zone)}/domains` },
          { label: domain },
        ]}
      />
      <PageHeader
        title={fqdn}
        description="Record 列表"
        actions={
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            新建 Record
          </Button>
        }
      />

      {isError ? (
        <div className="flex items-center gap-2 rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="size-4" aria-hidden />
          {(error as ApiError)?.message ?? "加载失败"}
        </div>
      ) : isPending ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-9 rounded-md bg-muted animate-pulse" />
          ))}
        </div>
      ) : records && records.length > 0 ? (
        <div className="overflow-hidden rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">ID</TableHead>
                <TableHead className="w-20">类型</TableHead>
                <TableHead>值</TableHead>
                <TableHead className="w-24 text-right">TTL</TableHead>
                <TableHead className="w-44">SRV(pri/port/wt)</TableHead>
                <TableHead className="w-28 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {r.id}
                  </TableCell>
                  <TableCell>
                    <RecordTypeBadge type={r.type} />
                  </TableCell>
                  <TableCell className="font-mono text-sm">{r.value}</TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {r.ttl}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {r.type === "SRV"
                      ? `${r.priority ?? "-"}/${r.port ?? "-"}/${r.weight ?? 0}`
                      : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`编辑 ${r.id}`}
                        onClick={() => openEdit(r)}
                      >
                        <Pencil className="size-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`删除 ${r.id}`}
                        onClick={() => setDelRec(r)}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyState onCreate={openCreate} />
      )}

      <RecordFormDialog
        zone={zone}
        domain={domain}
        record={editing}
        open={formOpen}
        onOpenChange={setFormOpen}
      />
      <ConfirmDialog
        open={!!delRec}
        onOpenChange={(o) => !o && setDelRec(null)}
        title="删除 Record"
        description={
          <>
            将删除 Record{" "}
            <span className="font-mono text-foreground">{delRec?.id}</span>(
            {delRec?.type}),此操作不可撤销。
          </>
        }
        confirmText="确认删除"
        destructive
        pending={deleteMut.isPending}
        onConfirm={() => {
          if (!delRec) return;
          deleteMut.mutate(
            { zone, domain, id: delRec.id },
            { onSuccess: () => setDelRec(null) },
          );
        }}
      />
    </div>
  );
}
