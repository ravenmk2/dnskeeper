import { z } from "zod";

/**
 * 密码强度规则(与后端一致):长度 6-24,至少包含
 * 大写/小写/数字/特殊字符 中的 2 类。
 */
export const passwordSchema = z
  .string()
  .min(6, "长度 6-24")
  .max(24, "长度 6-24")
  .refine((v) => {
    const classes = [
      /[a-z]/,
      /[A-Z]/,
      /[0-9]/,
      /[^a-zA-Z0-9]/,
    ].filter((r) => r.test(v)).length;
    return classes >= 2;
  }, "需包含大写/小写/数字/特殊字符中的 2 类");

/** 可选密码(编辑用户时留空表示不变) */
export const optionalPasswordSchema = z
  .string()
  .max(24, "长度 6-24")
  .refine((v) => {
    if (v === "") return true;
    const classes = [
      /[a-z]/,
      /[A-Z]/,
      /[0-9]/,
      /[^a-zA-Z0-9]/,
    ].filter((r) => r.test(v)).length;
    return classes >= 2 && v.length >= 6;
  }, "留空表示不变;否则需 6-24 位且含 2 类字符");
