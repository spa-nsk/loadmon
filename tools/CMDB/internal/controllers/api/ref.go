package api

import (
	"context"

	"cmdb/internal/services/refdata"
)

// RefController — /api/v1/ref: справочники, нужные интеграциям, чтобы заполнить КЕ корректными
// кодами (площадка, окружение, компания, тип компонента, тип связи, класс) без парсинга UI.
type RefController struct {
	BaseController
}

var refLists = map[string]func(context.Context) ([]refdata.Option, error){
	"locations":       refdata.Locations,
	"environments":    refdata.Environments,
	"companies":       refdata.Companies,
	"component-types": refdata.ComponentTypes,
	"rel-types":       refdata.RelTypes,
	"classes":         refdata.Classes,
}

// List — GET /api/v1/ref/{name}: одна из справочных таблиц (см. refLists) как [{"value","label"}].
func (c *RefController) List() {
	name := c.Ctx.Input.Param(":name")
	fn, ok := refLists[name]
	if !ok {
		c.badRequest("неизвестный справочник " + name)
	}
	opts, err := fn(c.ctx())
	c.must(err)
	c.writeJSON(200, map[string]any{"items": opts})
}
