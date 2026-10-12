// Package api хранит спецификацию REST API /api/v1 (plan.md E7.3) — единственный файл, встроенный
// в бинарник cmdb-web, чтобы GET /api/v1/openapi.yaml отдавал её без зависимости от рабочей
// директории процесса в проде.
package api

import _ "embed"

//go:embed openapi.yaml
var OpenAPIYAML string
