import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { AlertCircle, Layers, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Breadcrumb } from "@/components/breadcrumb";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { PageHeader } from "@/components/page-header";
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
import { useDomains } from "@/features/dns/hooks";
import { useCreateDomain, useDeleteDomain } from "@/features/dns/mutations";
import { ApiError } from "@/lib/errors";
import { formatDateTime } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import type { Domain } from "@/types/api";

const LABEL_RE =
  /^([a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/i;

const domainSchema = z.object({
  domain: z
    .string()
    .min(1, "不能为空")
    .refine(
      (v) => v === "@" || LABEL_RE.test(v),
      "无效域名,如 www / www.beta / @(Zone 根)",
    ),
});
type DomainValues = z.infer<typeof domainSchema>;

function CreateDomainDialog({
  zone,
  open,
  onOpenChange,
}: {
  zone: string;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const createMut = useCreateDomain();
  const form = useForm<DomainValues>({
    resolver: zodResolver(domainSchema),
    defaultValues: { domain: "" },
  });
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = form;

  useEffect(() => {
    if (open) reset({ domain: "" });
  }, [open, reset]);

  const onSubmit = (vals: DomainValues) => {
    createMut.mutate(
      { zone, domain: vals.domain },
      {
        onSuccess: () => {
          toast.success("Domain 已创建");
          onOpenChange(false);
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>新建 Domain</DialogTitle>
          <DialogDescription>
            在 Zone{" "}
            <span className="font-mono text-foreground">{zone}</span> 下创建子域名。
            用 <span className="font-mono">@</span> 表示 Zone 根。
          </DialogDescription>
        </DialogHeader>
        <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
          <Field>
            <FieldLabel htmlFor="domain">子域名</FieldLabel>
            <Input
              id="domain"
              placeholder="www"
              autoComplete="off"
              aria-invalid={!!errors.domain}
              {...register("domain")}
            />
            <FieldError errors={[errors.domain]} />
          </Field>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={createMut.isPending}>
              创建
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
      <Layers className="size-8 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-sm font-medium text-foreground">还没有 Domain</p>
        <p className="mt-1 text-xs text-muted-foreground">
          在此 Zone 下创建第一个 Domain
        </p>
      </div>
      <Button onClick={onCreate} size="sm">
        <Plus className="size-4" />
        新建 Domain
      </Button>
    </div>
  );
}

export default function Domains() {
  const { zone = "" } = useParams<{ zone: string }>();
  const { data: domains, isPending, isError, error } = useDomains(zone);
  const isAdmin = useAuthStore((s) => s.user?.user_type === "admin");
  const [createOpen, setCreateOpen] = useState(false);
  const [delDomain, setDelDomain] = useState<Domain | null>(null);
  const deleteMut = useDeleteDomain();

  return (
    <div>
      <Breadcrumb
        items={[
          { label: "DNS 管理", to: "/dns/zones" },
          { label: zone },
        ]}
      />
      <PageHeader
        title={zone}
        description="Domain 列表"
        actions={
          isAdmin && (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="size-4" />
              新建 Domain
            </Button>
          )
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
      ) : domains && domains.length > 0 ? (
        <div className="overflow-hidden rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead className="w-32 text-right">Record 数</TableHead>
                <TableHead className="w-44">创建时间</TableHead>
                {isAdmin && (
                  <TableHead className="w-20 text-right">操作</TableHead>
                )}
              </TableRow>
            </TableHeader>
            <TableBody>
              {domains.map((d) => (
                <TableRow key={d.domain}>
                  <TableCell>
                    <Link
                      to={`/dns/zones/${encodeURIComponent(zone)}/domains/${encodeURIComponent(d.domain)}/records`}
                      className="font-mono text-sm text-primary hover:underline"
                    >
                      {d.name}
                    </Link>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {d.record_count}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatDateTime(d.created_at)}
                  </TableCell>
                  {isAdmin && (
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`删除 ${d.name}`}
                        onClick={() => setDelDomain(d)}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <EmptyState onCreate={() => setCreateOpen(true)} />
      )}

      <CreateDomainDialog
        zone={zone}
        open={createOpen}
        onOpenChange={setCreateOpen}
      />
      <ConfirmDialog
        open={!!delDomain}
        onOpenChange={(o) => !o && setDelDomain(null)}
        title="删除 Domain"
        description={
          <>
            将删除 Domain{" "}
            <span className="font-mono text-foreground">
              {delDomain?.name}
            </span>{" "}
            及其下所有 Record,此操作不可撤销。
          </>
        }
        confirmText="确认删除"
        destructive
        pending={deleteMut.isPending}
        onConfirm={() => {
          if (!delDomain) return;
          deleteMut.mutate(
            { zone, domain: delDomain.domain },
            { onSuccess: () => setDelDomain(null) },
          );
        }}
      />
    </div>
  );
}
