import { cn } from "@/lib/utils";
import type { RecordType } from "@/types/api";

const STYLES: Record<RecordType, string> = {
  A: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  AAAA: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
  SRV: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  TXT: "bg-slate-500/10 text-slate-600 dark:text-slate-300",
};

export function RecordTypeBadge({ type }: { type: RecordType }) {
  return (
    <span
      className={cn(
        "inline-flex h-5 items-center rounded-md px-1.5 font-mono text-xs font-medium",
        STYLES[type],
      )}
    >
      {type}
    </span>
  );
}
