import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { AlertCircle, Pencil, Plus, Shield, Trash2 } from "lucide-react";
import { toast } from "sonner";

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
import { useUsers } from "@/features/users/hooks";
import {
  useCreateUser,
  useDeleteUser,
  useUpdateUser,
} from "@/features/users/mutations";
import { ApiError } from "@/lib/errors";
import { optionalPasswordSchema, passwordSchema } from "@/lib/password-schema";
import { cn, formatDateTime } from "@/lib/utils";
import type { User, UserType } from "@/types/api";

function RoleBadge({ type }: { type: UserType }) {
  return (
    <span
      className={cn(
        "inline-flex h-5 items-center gap-1 rounded-md px-1.5 text-xs font-medium",
        type === "admin"
          ? "bg-primary/10 text-primary"
          : "bg-secondary text-secondary-foreground",
      )}
    >
      {type === "admin" && <Shield className="size-3" aria-hidden />}
      {type === "admin" ? "管理员" : "普通用户"}
    </span>
  );
}

const usernameSchema = z
  .string()
  .min(2, "至少 2 字符")
  .max(32, "至多 32 字符")
  .regex(/^[a-zA-Z0-9_.-]+$/, "仅字母/数字/._-");

const createUserSchema = z.object({
  username: usernameSchema,
  password: passwordSchema,
  userType: z.enum(["admin", "normal"]),
});
type CreateUserValues = z.infer<typeof createUserSchema>;

const updateUserSchema = z.object({
  password: optionalPasswordSchema,
  userType: z.enum(["admin", "normal"]),
});
type UpdateUserValues = z.infer<typeof updateUserSchema>;

const selectClass =
  "h-8 w-full rounded-lg border border-input bg-transparent px-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:opacity-50 dark:bg-input/30";

function CreateUserDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const createMut = useCreateUser();
  const form = useForm<CreateUserValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: { username: "", password: "", userType: "normal" },
  });
  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = form;

  useEffect(() => {
    if (open)
      reset({ username: "", password: "", userType: "normal" });
  }, [open, reset]);

  const onSubmit = (vals: CreateUserValues) => {
    createMut.mutate(
      { username: vals.username, password: vals.password, user_type: vals.userType },
      {
        onSuccess: () => {
          toast.success("用户已创建");
          onOpenChange(false);
        },
        onError: (err: unknown) => {
          const e = err as ApiError;
          if (e.code === "VALIDATION_ERROR" || e.code === "USER_EXISTS") {
            const fe = e.fieldErrors();
            if (fe.username)
              form.setError("username", { message: fe.username });
            else toast.error(e.message);
          }
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>新建用户</DialogTitle>
          <DialogDescription>创建新的管理控制台账号</DialogDescription>
        </DialogHeader>
        <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="new-username">用户名</FieldLabel>
              <Input
                id="new-username"
                placeholder="alice"
                autoComplete="off"
                aria-invalid={!!errors.username}
                {...register("username")}
              />
              <FieldError errors={[errors.username]} />
            </Field>
            <Field>
              <FieldLabel htmlFor="new-password">密码</FieldLabel>
              <Input
                id="new-password"
                type="password"
                autoComplete="new-password"
                aria-invalid={!!errors.password}
                {...register("password")}
              />
              <FieldError errors={[errors.password]} />
              <p className="text-xs text-muted-foreground">
                6-24 位,含大小写/数字/特殊字符中的 2 类
              </p>
            </Field>
            <Field orientation="horizontal" className="items-center">
              <FieldLabel className="w-16 shrink-0">角色</FieldLabel>
              <select
                className={selectClass}
                aria-label="角色"
                value={watch("userType")}
                onChange={(e) =>
                  setValue("userType", e.target.value as UserType)
                }
              >
                <option value="normal">普通用户</option>
                <option value="admin">管理员</option>
              </select>
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
            <Button type="submit" disabled={createMut.isPending}>
              创建
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function UpdateUserDialog({
  user,
  open,
  onOpenChange,
}: {
  user: User | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const updateMut = useUpdateUser();
  const form = useForm<UpdateUserValues>({
    resolver: zodResolver(updateUserSchema),
    defaultValues: { password: "", userType: "normal" },
  });
  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = form;

  const demoteLocked = user?.builtin && user?.user_type === "admin";

  useEffect(() => {
    if (open && user)
      reset({ password: "", userType: user.user_type });
  }, [open, user, reset]);

  if (!user) return null;

  const onSubmit = (vals: UpdateUserValues) => {
    updateMut.mutate(
      {
        id: user.id,
        password: vals.password === "" ? undefined : vals.password,
        user_type: vals.userType,
      },
      {
        onSuccess: () => {
          toast.success("用户已更新");
          onOpenChange(false);
        },
        onError: (err: unknown) => {
          const e = err as ApiError;
          if (e.code === "VALIDATION_ERROR") {
            const fe = e.fieldErrors();
            if (fe.password)
              form.setError("password", { message: fe.password });
            else toast.error(e.message);
          }
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>编辑用户</DialogTitle>
          <DialogDescription>
            用户{" "}
            <span className="font-mono text-foreground">{user.username}</span>
            {user.builtin && " · 内置用户不可降级"}
          </DialogDescription>
        </DialogHeader>
        <form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <Field>
              <FieldLabel>用户名</FieldLabel>
              <Input value={user.username} disabled />
            </Field>
            <Field>
              <FieldLabel htmlFor="upd-password">新密码</FieldLabel>
              <Input
                id="upd-password"
                type="password"
                autoComplete="new-password"
                placeholder="留空则不变"
                aria-invalid={!!errors.password}
                {...register("password")}
              />
              <FieldError errors={[errors.password]} />
            </Field>
            <Field orientation="horizontal" className="items-center">
              <FieldLabel className="w-16 shrink-0">角色</FieldLabel>
              <select
                className={selectClass}
                aria-label="角色"
                disabled={demoteLocked}
                value={watch("userType")}
                onChange={(e) =>
                  setValue("userType", e.target.value as UserType)
                }
              >
                <option value="normal">普通用户</option>
                <option value="admin">管理员</option>
              </select>
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
            <Button type="submit" disabled={updateMut.isPending}>
              保存
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default function Users() {
  const { data: users, isPending, isError, error } = useUsers();
  const [createOpen, setCreateOpen] = useState(false);
  const [editUser, setEditUser] = useState<User | null>(null);
  const [delUser, setDelUser] = useState<User | null>(null);
  const deleteMut = useDeleteUser();

  return (
    <div>
      <PageHeader
        title="用户管理"
        description="管理控制台账号与角色"
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="size-4" />
            新建用户
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
      ) : (
        <div className="overflow-hidden rounded-xl border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户名</TableHead>
                <TableHead className="w-32">角色</TableHead>
                <TableHead className="w-24">内置</TableHead>
                <TableHead className="w-44">创建时间</TableHead>
                <TableHead className="w-28 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users?.map((u) => (
                <TableRow key={u.id}>
                  <TableCell className="font-mono text-sm">
                    {u.username}
                  </TableCell>
                  <TableCell>
                    <RoleBadge type={u.user_type} />
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {u.builtin ? "是" : "—"}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatDateTime(u.created_at)}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`编辑 ${u.username}`}
                        onClick={() => setEditUser(u)}
                      >
                        <Pencil className="size-3.5" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`删除 ${u.username}`}
                        disabled={u.builtin}
                        title={u.builtin ? "内置用户不可删除" : undefined}
                        onClick={() => setDelUser(u)}
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
      )}

      <CreateUserDialog open={createOpen} onOpenChange={setCreateOpen} />
      <UpdateUserDialog
        user={editUser}
        open={!!editUser}
        onOpenChange={(o) => !o && setEditUser(null)}
      />
      <ConfirmDialog
        open={!!delUser}
        onOpenChange={(o) => !o && setDelUser(null)}
        title="删除用户"
        description={
          <>
            将删除用户{" "}
            <span className="font-mono text-foreground">
              {delUser?.username}
            </span>
            ,此操作不可撤销。
          </>
        }
        confirmText="确认删除"
        destructive
        pending={deleteMut.isPending}
        onConfirm={() => {
          if (!delUser) return;
          deleteMut.mutate(delUser.id, {
            onSuccess: () => setDelUser(null),
          });
        }}
      />
    </div>
  );
}
