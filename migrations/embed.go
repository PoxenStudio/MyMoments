package migrations

import "embed"

// FS 保存按版本号排序的向前迁移脚本。
//
//go:embed *.sql
var FS embed.FS
