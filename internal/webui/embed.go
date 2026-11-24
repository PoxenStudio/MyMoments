package webui

import (
	"embed"
	"io/fs"
)

// FS 保存前端构建产物；未构建时提供占位页。
//
//go:embed all:static
var embedded embed.FS

// FS 返回以站点根为前缀的静态资源。
func FS() (fs.FS, error) {
	return fs.Sub(embedded, "static")
}
