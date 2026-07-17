import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { AlertCircle, Globe, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

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
import { useCreateZone, useDeleteZone } from "@/features/dns/mutations";
import { useZones as useZonesQuery } from "@/features/dns/hooks";
import { ApiError } from "@/lib/errors";
import { formatDateTime } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import type { Zone } from "@/types/api";

const ZONE_RE =
  /^([a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/i;

const zoneSchema = z.object({
  zone: z
    .string()
    .min(3, "至少 2 级标签,如 example.com")
    .regex(ZONE_RE, "无效域名,如 example.com"),
});
type ZoneValues = z.infer<typeof zoneSchema>;

function CreateZoneDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const createMut = useCreateZone();
  const form = useForm<ZoneValues>({
    resolver: zodResolver(zoneSchema),
    defaultValues: { zone: "" },
  });
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = form;

  useEffect(() => {
    if (open) reset({ zone: "" });
  }, [open, reset]);

  const onSubmit = (vals: ZoneValues) => {
    createMut.mutate(vals.zone, {
      onSuccess: () => {
        toast.success("Zone 已创建");
        onOpenChange(false);
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>新建 Zone</DialogTitle>
          <DialogDescription>
            Zone 是顶级域名容器,至少 2 级标签,如 example.com
          </DialogDescription>
        </DialogHeader>
        <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
          <Field>
            <FieldLabel htmlFor="zone">域名</FieldLabel>
            <Input
              id="zone"
              placeholder="example.com"
              autoComplete="off"
              aria-invalid={!!errors.zone}
              {...register("zone")}
            />
            <FieldError errors={[errors.zone]} />
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

function DeleteZoneDialog({
  zone,
  onOpenChange,
}: {
  zone: Zone | null;
  onOpenChange: (v: boolean) => void;
}) {
  const deleteMut = useDeleteZone();
  return (
    <Dialog open={!!zone} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>删除 Zone</DialogTitle>
          <DialogDescription>
            将删除 Zone{" "}
            <span className="font-mono text-foreground">{zone?.zone}</span>{" "}
            及其下所有 Domain 与 Record,此操作不可撤销。
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={deleteMut.isPending}
            onClick={() => {
              if (!zone) return;
              deleteMut.mutate(zone.zone, { onSuccess: () => onOpenChange(false) });
            }}
          >
            确认删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border py-16 text-center">
      <Globe className="size-8 text-muted-foreground" aria-hidden />
      <div>
        <p className="text-sm font-medium text-foreground">还没有 Zone</p>
        <p className="mt-1 text-xs text-muted-foreground">
          创建第一个 Zone 以开始管理 DNS 记录
        </p>
      </div>
      <Button onClick={onCreate} size="sm">
        <Plus className="size-4" />
        新建 Zone
      </Button>
    </div>
  );
}

export default function Zones() {
  const { data: zones, isPending, isError, error } = useZonesQuery();
  const isAdmin = useAuthStore((s) => s.user?.user_type === "admin");
  const [createOpen, setCreateOpen] = useState(false);
  const [delZone, setDelZone] = useState<Zone | null>(null);

  return (
    <div>
      <PageHeader
        title="DNS 管理"
        description="Zone 列表"
        actions={
          isAdmin && (
            <Button onClick={() => setCreateOpen(true)}>
              <Plus className="size-4" />
              新建 Zone
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
            <div
              key={i}
              className="h-9 rounded-md bg-muted animate-pulse"
            />
          ))}
        </div>
      ) : zones && zones.length > 0 ? (
        <div className="overflow-hidden rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Zone</TableHead>
                <TableHead className="w-32 text-right">Domain 数</TableHead>
                <TableHead className="w-44">创建时间</TableHead>
                {isAdmin && <TableHead className="w-20 text-right">操作</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {zones.map((z) => (
                <TableRow key={z.zone}>
                  <TableCell>
                    <Link
                      to={`/dns/zones/${encodeURIComponent(z.zone)}/domains`}
                      className="font-mono text-sm text-primary hover:underline"
                    >
                      {z.zone}
                    </Link>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {z.domain_count}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatDateTime(z.created_at)}
                  </TableCell>
                  {isAdmin && (
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`删除 ${z.zone}`}
                        onClick={() => setDelZone(z)}
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

      <CreateZoneDialog open={createOpen} onOpenChange={setCreateOpen} />
      <DeleteZoneDialog zone={delZone} onOpenChange={(o) => !o && setDelZone(null)} />
    </div>
  );
}
