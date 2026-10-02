package vspheremetrics

import (
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

// Отметки успешной полной синхронизации пула в памяти с vCenter.
//
// PruneStaleObjects в конце каждого опроса vCenter строит "живой" набор
// хостов/ВМ/кластеров прямым запросом к vCenter и удаляет из пула всё
// отсутствующее. Если он отработал без ошибок, пул (HostMapList/VMMapList/
// HostBySNMapList) по этому vCenter достоверен: объекта, которого в пуле
// нет, в vCenter действительно нет. Если опрос или сверка упали, пул мог
// остаться устаревшим - тогда "нет в пуле" ещё не значит "нет в VMware".
//
// Используется cmdb (единое правило Stat/PowerState/Active, 2026-10-02):
// запись помечается "не найдена" только если vCenter, в котором её искали,
// недавно синхронизирован успешно (иначе недоступность vCenter вывела бы
// из эксплуатации весь парк).
var vcenterSyncedAt = cmap.New[time.Time]()

// MarkVCenterSynced фиксирует время успешной полной сверки пула с vCenter.
func MarkVCenterSynced(vcenter string) {
	vcenterSyncedAt.Set(vcenter, time.Now())
}

// VCenterSyncedSince - vCenter успешно синхронизирован не раньше чем maxAge назад.
func VCenterSyncedSince(vcenter string, maxAge time.Duration) bool {
	t, ok := vcenterSyncedAt.Get(vcenter)
	return ok && time.Since(t) <= maxAge
}
