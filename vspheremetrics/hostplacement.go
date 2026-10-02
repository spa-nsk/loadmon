package vspheremetrics

import (
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/net/context"
)

// Заглушки размещения хоста, когда vCenter не отдал датацентр или
// родителя (кластер/ComputeResource). Хост при этом НЕ теряется
// (2026-10-02): он попадает в пул и в справочники, а в ServerSprRecords
// Datacenter/Cluster пишутся пустыми (PlacementValue). Заглушки нужны,
// чтобы ключи пула (vcenter/датацентр/кластер/имя) оставались
// уникальными и читаемыми в логах.
const (
	NoDatacenter = "(датацентр не определён)"
	NoCluster    = "(кластер не определён)"
)

// PlacementValue - значение датацентра/кластера для записи в справочники:
// заглушки NoDatacenter/NoCluster превращаются в пустую строку.
func PlacementValue(s string) string {
	if s == NoDatacenter || s == NoCluster {
		return ""
	}
	return s
}

// hostDatacenter определяет датацентр хоста по полному списку его предков
// (mo.Ancestors - один запрос к PropertyCollector), без ручного подъёма по
// папкам: работает при любой вложенности папок, для хостов в кластерах и
// вне кластеров. Возвращает имя и ссылку на объект Datacenter (нужна для
// чтения тегов среды).
func hostDatacenter(ctx context.Context, c *govmomi.Client, host types.ManagedObjectReference) (string, *types.ManagedObjectReference, error) {
	ents, err := mo.Ancestors(ctx, c.Client, c.ServiceContent.PropertyCollector, host)
	if err != nil {
		return "", nil, fmt.Errorf("чтение предков хоста: %w", err)
	}
	for i := len(ents) - 1; i >= 0; i-- {
		if ents[i].Self.Type == "Datacenter" {
			ref := ents[i].Self
			return ents[i].Name, &ref, nil
		}
	}
	return "", nil, fmt.Errorf("среди предков хоста нет объекта Datacenter")
}
