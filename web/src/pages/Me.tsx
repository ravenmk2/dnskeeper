import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Shield } from "lucide-react";
import { toast } from "sonner";

import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { useChangePassword } from "@/features/auth/use-auth";
import { ApiError } from "@/lib/errors";
import { passwordSchema } from "@/lib/password-schema";
import { cn, formatDateTime } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";

const schema = z
  .object({
    old_password: z.string().min(1, "请输入旧密码"),
    new_password: passwordSchema,
    confirm: z.string().min(1, "请确认新密码"),
  })
  .refine((d) => d.new_password === d.confirm, {
    path: ["confirm"],
    message: "两次输入不一致",
  })
  .refine((d) => d.old_password !== d.new_password, {
    path: ["new_password"],
    message: "新密码不能与旧密码相同",
  });
type Values = z.infer<typeof schema>;

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between py-1.5 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-foreground">{children}</span>
    </div>
  );
}

export default function Me() {
  const user = useAuthStore((s) => s.user);
  const changeMut = useChangePassword();
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { old_password: "", new_password: "", confirm: "" },
  });
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors },
  } = form;

  const onSubmit = (vals: Values) => {
    changeMut.mutate(
      { old_password: vals.old_password, new_password: vals.new_password },
      {
        onSuccess: () => {
          toast.success("密码已修改");
          reset({ old_password: "", new_password: "", confirm: "" });
        },
        onError: (err: unknown) => {
          const e = err as ApiError;
          if (e.code === "WRONG_PASSWORD")
            setError("old_password", { message: "旧密码错误" });
          else if (e.code === "SAME_PASSWORD")
            setError("new_password", { message: "新密码不能与旧密码相同" });
          else if (e.code === "WEAK_PASSWORD")
            setError("new_password", {
              message: "密码强度不足(6-24 位,含 2 类字符)",
            });
        },
      },
    );
  };

  return (
    <div>
      <PageHeader title="个人中心" description="当前账号信息与密码修改" />

      <div className="grid gap-6 md:grid-cols-2">
        <Card size="sm">
          <CardHeader>
            <CardTitle className="text-sm font-medium">账号信息</CardTitle>
          </CardHeader>
          <CardContent>
            {user ? (
              <div className="divide-y divide-border">
                <InfoRow label="用户名">
                  <span className="font-mono">{user.username}</span>
                </InfoRow>
                <InfoRow label="角色">
                  <span
                    className={cn(
                      "inline-flex items-center gap-1",
                      user.user_type === "admin" && "text-primary",
                    )}
                  >
                    {user.user_type === "admin" && (
                      <Shield className="size-3" aria-hidden />
                    )}
                    {user.user_type === "admin" ? "管理员" : "普通用户"}
                  </span>
                </InfoRow>
                <InfoRow label="内置账号">{user.builtin ? "是" : "否"}</InfoRow>
                <InfoRow label="创建时间">
                  <span className="font-mono text-xs">
                    {formatDateTime(user.created_at)}
                  </span>
                </InfoRow>
                <InfoRow label="更新时间">
                  <span className="font-mono text-xs">
                    {formatDateTime(user.updated_at)}
                  </span>
                </InfoRow>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">未加载用户信息</p>
            )}
          </CardContent>
        </Card>

        <Card size="sm">
          <CardHeader>
            <CardTitle className="text-sm font-medium">修改密码</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className={cn("flex flex-col gap-4")}
              onSubmit={handleSubmit(onSubmit)}
            >
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="old_password">旧密码</FieldLabel>
                  <Input
                    id="old_password"
                    type="password"
                    autoComplete="current-password"
                    aria-invalid={!!errors.old_password}
                    {...register("old_password")}
                  />
                  <FieldError errors={[errors.old_password]} />
                </Field>
                <Field>
                  <FieldLabel htmlFor="new_password">新密码</FieldLabel>
                  <Input
                    id="new_password"
                    type="password"
                    autoComplete="new-password"
                    aria-invalid={!!errors.new_password}
                    {...register("new_password")}
                  />
                  <FieldError errors={[errors.new_password]} />
                  <p className="text-xs text-muted-foreground">
                    6-24 位,含大小写/数字/特殊字符中的 2 类
                  </p>
                </Field>
                <Field>
                  <FieldLabel htmlFor="confirm">确认新密码</FieldLabel>
                  <Input
                    id="confirm"
                    type="password"
                    autoComplete="new-password"
                    aria-invalid={!!errors.confirm}
                    {...register("confirm")}
                  />
                  <FieldError errors={[errors.confirm]} />
                </Field>
              </FieldGroup>
              <Button type="submit" disabled={changeMut.isPending}>
                修改密码
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
