package api

import "cmdb/internal/services/baseline"

// BaselinesController — /api/v1/baselines: список и карточка именованных срезов (E6.2). Создание,
// заморозка и утверждение срезов через API не выставлены — это редкие плановые операции с
// подтверждением (roles: accounting_admin), интеграциям нужно только читать.
type BaselinesController struct {
	BaseController
}

// List — GET /api/v1/baselines.
func (c *BaselinesController) List() {
	rows, err := baseline.List(c.ctx())
	c.must(err)
	c.writeJSON(200, map[string]any{"baselines": rows})
}

// Get — GET /api/v1/baselines/{id}: срез вместе со счётом элементов по сущностям.
func (c *BaselinesController) Get() {
	id := c.id()
	b, err := baseline.Get(c.ctx(), id)
	c.must(err)
	counts, err := baseline.Count(c.ctx(), id)
	c.must(err)
	c.writeJSON(200, map[string]any{"baseline": b, "counts": counts})
}
