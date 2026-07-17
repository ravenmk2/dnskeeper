import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

export interface Crumb {
  label: string;
  to?: string;
}

/**
 * 简洁面包屑。最后一项为当前页(加粗、不可点);中间项可点。
 */
export function Breadcrumb({ items }: { items: Crumb[] }) {
  return (
    <nav
      aria-label="面包屑"
      className="mb-4 flex flex-wrap items-center gap-1 text-sm text-muted-foreground"
    >
      {items.map((it, i) => {
        const last = i === items.length - 1;
        return (
          <span key={i} className="flex items-center gap-1">
            {i > 0 && (
              <ChevronRight className="size-3.5 shrink-0" aria-hidden />
            )}
            {it.to && !last ? (
              <Link
                to={it.to}
                className="hover:text-foreground"
              >
                {it.label}
              </Link>
            ) : (
              <span
                className={cn(
                  "font-mono",
                  last && "font-medium text-foreground",
                )}
              >
                {it.label}
              </span>
            )}
          </span>
        );
      })}
    </nav>
  );
}
