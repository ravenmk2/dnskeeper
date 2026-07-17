import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="flex min-h-[100dvh] flex-col items-center justify-center gap-3 bg-background p-6 text-center">
      <p className="font-mono text-5xl font-semibold tracking-tight text-foreground">
        404
      </p>
      <p className="text-sm text-muted-foreground">页面不存在</p>
      <Link
        to="/"
        className="text-sm font-medium text-primary hover:underline"
      >
        返回首页
      </Link>
    </div>
  );
}
