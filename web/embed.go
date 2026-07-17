// Package web 内嵌前端构建产物(web/dist),供后端同源提供 SPA。
// 编译依赖:web/dist 须存在(至少含 .gitkeep 占位;完整前端需在 web/ 执行 npm run build)。
package web

import "embed"

// Assets 嵌入整个 web/dist。运行时由 app 层取 "dist" 子树作为静态根。
//
//go:embed all:dist
var Assets embed.FS
