package ui

import (
	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/services/refadmin"
)

func init() {
	_ = web.AddFuncMap("stageClass", stageClass)
	_ = web.AddFuncMap("namingClass", namingClass)
	_ = web.AddFuncMap("indent", func(depth int) float64 { return float64(depth) * 1.25 })
	_ = web.AddFuncMap("isActiveStage", func(stage string) bool { return stage != "Retired" && stage != "Disposed" })
	// fieldType — refadmin.FieldType как обычная строка, чтобы сравнивать в шаблоне через eq.
	_ = web.AddFuncMap("fieldType", func(f refadmin.Field) string { return string(f.Type) })
}

// stageClass — класс бейджа Bootstrap для стадии жизненного цикла.
func stageClass(stage string) string {
	switch stage {
	case "Operational":
		return "text-bg-success"
	case "Commissioning", "Maintenance", "Decommissioning":
		return "text-bg-warning"
	case "Retired", "Disposed":
		return "text-bg-dark"
	}
	return "text-bg-secondary"
}

// namingClass — класс бейджа для статуса соответствия имени профилю.
func namingClass(status string) string {
	switch status {
	case "compliant":
		return "text-bg-success"
	case "legacy":
		return "text-bg-info"
	case "noncompliant":
		return "text-bg-danger"
	}
	return "text-bg-secondary"
}

func init() {
	_ = web.AddFuncMap("add", func(a, b int) int { return a + b })
	_ = web.AddFuncMap("sub", func(a, b int) int { return a - b })
}
