// Package webui 嵌入前端静态资源。
package webui

import "embed"

// Assets 嵌入式前端资源。
//
// 目录结构：frontend/dist 由 Wails/前端构建产物填充。
// 此处通过 go:embed 直接将 dist 目录下所有非隐藏文件嵌入二进制。
//
//go:embed all:frontend_dist
var Assets embed.FS
