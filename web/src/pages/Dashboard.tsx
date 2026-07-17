import { AlertCircle, Globe, Hash, Layers, Server } from "lucide-react";

import { PageHeader } from "@/components/page-header";
import {
  Card,
  CardContent,
  CardHeader,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useStats } from "@/features/dns/hooks";
import { useHealth } from "@/features/system/use-health";
import { ApiError } from "@/lib/errors";
import { cn } from "@/lib/utils";

function StatCard({
  label,
  value,
  icon: Icon,
  pending,
}: {
  label: string;
  value: number;
  icon: React.ComponentType<{ className?: string }>;
  pending: boolean;
}) {
  return (
    <Card size="sm">
      <CardHeader>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Icon className="size-4" aria-hidden />
          <span className="text-xs font-medium">{label}</span>
        </div>
      </CardHeader>
      <CardContent>
        {pending ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <p className="font-mono text-3xl font-semibold tracking-tight text-foreground">
            {value}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function HealthCard() {
  const { status, query } = useHealth();
  const dotClass =
    status === "healthy"
      ? "bg-emerald-500"
      : status === "unhealthy"
        ? "bg-red-500"
        : "bg-muted-foreground/40";
  const label =
    status === "healthy"
      ? "etcd 健康"
      : status === "unhealthy"
        ? "etcd 不可用"
        : "检查中";
  const err = query.isError ? (query.error as ApiError) : null;

  return (
    <Card size="sm">
      <CardHeader>
        <div className="flex items-center gap-2 text-muted-foreground">
          <Server className="size-4" aria-hidden />
          <span className="text-xs font-medium">etcd 健康</span>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2">
          <span
            className={cn("size-2.5 rounded-full", dotClass)}
            aria-hidden
          />
          <span className="text-sm font-medium text-foreground">{label}</span>
        </div>
        {err && (
          <p className="mt-1.5 text-xs text-muted-foreground">{err.message}</p>
        )}
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  const stats = useStats();
  const pending = stats.isPending;

  return (
    <div>
      <PageHeader
        title="概览"
        description="CoreDNS etcd 管理控制台总览"
      />
      {stats.isError ? (
        <Card size="sm">
          <CardContent>
            <div className="flex items-center gap-2 text-destructive">
              <AlertCircle className="size-4" aria-hidden />
              <span className="text-sm">
                {(stats.error as ApiError)?.message ?? "加载统计失败"}
              </span>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <StatCard
            label="Zone"
            value={stats.data?.zoneCount ?? 0}
            icon={Globe}
            pending={pending}
          />
          <StatCard
            label="Domain"
            value={stats.data?.domainCount ?? 0}
            icon={Layers}
            pending={pending}
          />
          <StatCard
            label="Record"
            value={stats.data?.recordCount ?? 0}
            icon={Hash}
            pending={pending}
          />
          <HealthCard />
        </div>
      )}
    </div>
  );
}
