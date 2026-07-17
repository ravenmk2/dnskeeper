import { Suspense, useState } from "react";
import {
  Link,
  NavLink,
  Outlet,
  useLocation,
  useNavigate,
} from "react-router-dom";
import {
  Globe,
  LayoutDashboard,
  LogOut,
  Moon,
  Sun,
  UserCircle,
  Users as UsersIcon,
} from "lucide-react";

import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useMe } from "@/features/auth/use-auth";
import { useHealth } from "@/features/system/use-health";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import type { UserType } from "@/types/api";

interface NavItem {
  to: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  adminOnly?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "概览", icon: LayoutDashboard },
  { to: "/dns/zones", label: "DNS 管理", icon: Globe },
  { to: "/admin/users", label: "用户管理", icon: UsersIcon, adminOnly: true },
  { to: "/me", label: "个人中心", icon: UserCircle },
];

function Brand() {
  return (
    <Link
      to="/"
      className="flex items-center gap-2 px-4 h-14 border-b border-sidebar-border"
    >
      <Globe className="size-4 text-primary" aria-hidden />
      <span className="font-mono text-sm font-semibold tracking-tight text-sidebar-foreground">
        Dnskeeper
      </span>
    </Link>
  );
}

function NavItemLink({ item }: { item: NavItem }) {
  return (
    <NavLink
      to={item.to}
      end={item.to === "/"}
      className={({ isActive }) =>
        cn(
          "flex items-center gap-2.5 rounded-md px-3 py-1.5 text-sm transition-colors",
          isActive
            ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
            : "text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground",
        )
      }
    >
      <item.icon className="size-4" aria-hidden />
      {item.label}
    </NavLink>
  );
}

function SidebarNav() {
  const userType = useAuthStore((s) => s.user?.user_type);
  return (
    <nav className="flex flex-1 flex-col gap-1 p-3" aria-label="主导航">
      {NAV_ITEMS.filter(
        (i) => !i.adminOnly || userType === "admin",
      ).map((item) => (
        <NavItemLink key={item.to} item={item} />
      ))}
    </nav>
  );
}

function ThemeToggle() {
  const theme = useThemeStore((s) => s.theme);
  const toggle = useThemeStore((s) => s.toggle);
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={toggle}
      aria-label={theme === "dark" ? "切换到浅色" : "切换到深色"}
    >
      {theme === "dark" ? (
        <Sun className="size-4" />
      ) : (
        <Moon className="size-4" />
      )}
    </Button>
  );
}

function roleLabel(t: UserType | undefined): string | null {
  if (t === "admin") return "管理员";
  if (t === "normal") return "普通用户";
  return null;
}

function UserBlock() {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  if (!user) return null;
  const initials = user.username.slice(0, 2).toUpperCase();
  return (
    <div className="border-t border-sidebar-border p-3">
      <div className="flex items-center gap-2.5 rounded-md px-2 py-1.5">
        <Avatar className="size-7 rounded-md">
          <AvatarFallback className="rounded-md bg-primary/10 text-primary text-xs font-medium">
            {initials}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-sidebar-foreground">
            {user.username}
          </p>
          <p className="truncate text-xs text-muted-foreground">
            {roleLabel(user.user_type)}
          </p>
        </div>
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => {
            logout();
            navigate("/login", { replace: true });
          }}
          aria-label="登出"
        >
          <LogOut className="size-4" />
        </Button>
      </div>
    </div>
  );
}

function HealthDot() {
  const { status } = useHealth();
  const dotClass =
    status === "healthy"
      ? "bg-emerald-500"
      : status === "unhealthy"
        ? "bg-red-500"
        : "bg-muted-foreground/40";
  const label =
    status === "healthy" ? "etcd 健康" : status === "unhealthy" ? "etcd 不可用" : "检查中";
  return (
    <span
      className="inline-flex items-center gap-1.5 text-xs text-muted-foreground"
      title={label}
    >
      <span className={cn("size-2 rounded-full", dotClass)} aria-hidden />
      <span className="hidden sm:inline">{label}</span>
    </span>
  );
}

function sectionLabel(pathname: string): string {
  if (pathname === "/" || pathname === "") return "概览";
  if (pathname.startsWith("/dns")) return "DNS 管理";
  if (pathname.startsWith("/admin/users")) return "用户管理";
  if (pathname.startsWith("/me")) return "个人中心";
  return "";
}

export function AppShell() {
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);
  const label = sectionLabel(location.pathname);

  // 引导校验:刷新后若 user 丢失但有 token,补拉一次
  useMe();

  return (
    <div className="flex min-h-[100dvh] bg-background">
      {/* 桌面侧栏 */}
      <aside className="hidden md:flex w-60 shrink-0 flex-col border-r border-border bg-sidebar">
        <Brand />
        <SidebarNav />
        <div className="flex items-center justify-end px-3 pb-2">
          <ThemeToggle />
        </div>
        <UserBlock />
      </aside>

      {/* 移动端抽屉(简化:覆盖层) */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setMobileOpen(false)}
            aria-hidden
          />
          <aside className="absolute left-0 top-0 h-full w-64 flex flex-col bg-sidebar border-r border-sidebar-border">
            <Brand />
            <SidebarNav />
            <div className="flex items-center justify-end px-3 pb-2">
              <ThemeToggle />
            </div>
            <UserBlock />
          </aside>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between border-b border-border px-4 md:px-6">
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="icon-sm"
              className="md:hidden"
              onClick={() => setMobileOpen((v) => !v)}
              aria-label="切换导航"
            >
              <Globe className="size-4" />
            </Button>
            <span className="text-sm font-medium text-foreground">{label}</span>
          </div>
          <HealthDot />
        </header>
        <main className="flex-1 overflow-auto p-4 md:p-6">
          <Suspense
            fallback={
              <div className="space-y-3">
                <div className="h-6 w-40 rounded-md bg-muted animate-pulse" />
                <div className="h-9 w-full max-w-md rounded-md bg-muted animate-pulse" />
                <div className="h-9 w-full max-w-md rounded-md bg-muted animate-pulse" />
              </div>
            }
          >
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
