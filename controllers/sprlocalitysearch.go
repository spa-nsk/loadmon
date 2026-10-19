package controllers

import (
	"loadmon/models"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// localityHit - элемент ответа поиска населённых пунктов для подсказки в формах.
type localityHit struct {
	Oktmo string `json:"oktmo"`
	Label string `json:"label"` // "г Новосибирск - город Новосибирск, Новосибирской области"
}

// SprLocalitySearch отдаёт JSON-список (до 20) населённых пунктов, название
// которых начинается с / содержит параметр q (от 2 символов): сначала города
// ("г") с совпадением по началу названия, затем остальные. Используется
// подсказкой в карточке площадки/ЦОД (datacentersiteedit.tpl).
//
// Маршрут: /sprsrv/localitysearch?q=...
func (this *MainController) SprLocalitySearch() {
	hits := []localityHit{}
	q := strings.TrimSpace(this.GetString("q"))
	if _, ok := this.GetSession("loadmon").(models.LoadFilter); ok && len([]rune(q)) >= 2 {
		const limit = 20
		o := orm.NewOrm()
		seen := map[string]bool{}
		add := func(qs orm.QuerySeter) {
			var list []models.Locality
			if _, err := qs.Limit(limit).All(&list); err != nil {
				return
			}
			for _, l := range list {
				if len(hits) >= limit {
					return
				}
				if seen[l.Oktmo] {
					continue
				}
				seen[l.Oktmo] = true
				label := l.FullName
				if l.Municipality != "" {
					label += " — " + l.Municipality
				}
				label += ", " + l.RegionName
				hits = append(hits, localityHit{Oktmo: l.Oktmo, Label: label})
			}
		}
		base := func() orm.QuerySeter { return o.QueryTable("cmdb_locality") }
		add(base().Filter("Name__istartswith", q).Filter("TypeShort", "г").OrderBy("Name", "Oktmo"))
		add(base().Filter("Name__istartswith", q).OrderBy("Name", "Oktmo"))
		if len(hits) < limit {
			add(base().Filter("Name__icontains", q).OrderBy("Name", "Oktmo"))
		}
	}
	this.Data["json"] = hits
	this.ServeJSON()
}
