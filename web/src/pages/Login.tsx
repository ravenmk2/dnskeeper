import { useState } from "react";
import { useForm } from "react-hook-form";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Eye, EyeOff, Globe, Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useLogin } from "@/features/auth/use-auth";
import { ApiError } from "@/lib/errors";
import { useAuthStore } from "@/stores/auth";

const schema = z.object({
  username: z.string().min(1, "请输入用户名"),
  password: z.string().min(1, "请输入密码"),
});

type Values = z.infer<typeof schema>;

export default function Login() {
  const token = useAuthStore((s) => s.token);
  const location = useLocation();
  const navigate = useNavigate();

  const from =
    (location.state as { from?: string } | null)?.from ?? "/";

  const [showPw, setShowPw] = useState(false);
  const loginMut = useLogin();

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { username: "", password: "" },
  });
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = form;

  if (token) return <Navigate to={from} replace />;

  const onSubmit = (vals: Values) => {
    loginMut.mutate(vals, {
      onSuccess: (me) => {
        toast.success(`欢迎,${me.username}`);
        navigate(from, { replace: true });
      },
      onError: (err: unknown) => {
        const e = err as ApiError;
        if (e.code === "INVALID_CREDENTIALS") {
          setError("password", { message: "用户名或密码错误" });
          return;
        }
        toast.error(e.message || "登录失败");
      },
    });
  };

  return (
    <div className="flex min-h-[100dvh] items-center justify-center bg-background p-4">
      <Card size="sm" className="w-full max-w-sm">
        <CardHeader>
          <div className="flex items-center gap-2">
            <Globe className="size-4 text-primary" aria-hidden />
            <CardTitle className="font-mono text-base font-semibold tracking-tight">
              Dnskeeper
            </CardTitle>
          </div>
          <CardDescription className="text-xs">
            CoreDNS etcd 插件 DNS 管理控制台
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={handleSubmit(onSubmit)}
          >
            {loginMut.isError && (loginMut.error as ApiError)?.code !== "INVALID_CREDENTIALS" && (
              <Alert variant="destructive">
                <AlertDescription>
                  {(loginMut.error as ApiError)?.message}
                </AlertDescription>
              </Alert>
            )}
            <Field>
              <FieldLabel htmlFor="username">用户名</FieldLabel>
              <Input
                id="username"
                autoComplete="username"
                aria-invalid={!!errors.username}
                {...register("username")}
              />
              <FieldError errors={[errors.username]} />
            </Field>
            <Field>
              <FieldLabel htmlFor="password">密码</FieldLabel>
              <div className="relative">
                <Input
                  id="password"
                  type={showPw ? "text" : "password"}
                  autoComplete="current-password"
                  className="pr-8"
                  aria-invalid={!!errors.password}
                  {...register("password")}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="absolute right-1 top-1/2 -translate-y-1/2"
                  onClick={() => setShowPw((v) => !v)}
                  aria-label={showPw ? "隐藏密码" : "显示密码"}
                  tabIndex={-1}
                >
                  {showPw ? (
                    <EyeOff className="size-3.5" />
                  ) : (
                    <Eye className="size-3.5" />
                  )}
                </Button>
              </div>
              <FieldError errors={[errors.password]} />
            </Field>
            <Button
              type="submit"
              size="lg"
              className="mt-1 w-full"
              disabled={loginMut.isPending}
            >
              {loginMut.isPending && (
                <Loader2 className="size-4 animate-spin" />
              )}
              登录
            </Button>
          </form>
          <p className="mt-4 text-center text-xs text-muted-foreground">
            内置管理员 <span className="font-mono">admin</span> /{" "}
            <span className="font-mono">admin123</span>(首次登录后请尽快改密)
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
