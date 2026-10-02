package models

import "github.com/beego/beego/v2/client/orm"

// Значения поля ServerSprRecords.Stat.
//
// ServerSprStatDestroy (666) - ТЕХНИЧЕСКАЯ пометка "запись помечена для
// уничтожения" (лишний дубль, найденный дедупликацией при синхронизации с
// HP OneView/vSphere - см. checkAndCorrect* в cmdb). Это техническая
// альтернатива ServerSprStatMarkedForDelete (2): правило владельца проекта
// (2026-10-02) - "из state=666 нельзя переводить в другие, это должно
// пропускаться во всех запросах". Поэтому:
//   - все выборки server_spr_records идут через ServerSprQS (или
//     ExcludeDestroyedCond для запросов с SetCond) - такие записи не
//     попадают ни в синхронизации, ни в отчёты, ни в формы;
//   - запись, прочитанная по ключу (Id/Name/HWSprId...), проверяется
//     IsServerSprDestroyed и считается "не найденной" - её поля, включая
//     Stat, не перезаписываются никаким кодом.
// Ставить 666 могут только функции дедупликации; снимать - никто.
const (
	ServerSprStatHistory         = 0   // историческая запись / выведена из учёта
	ServerSprStatUsed            = 1   // используется
	ServerSprStatMarkedForDelete = 2   // помечено для удаления (вручную)
	ServerSprStatDestroy         = 666 // техническая пометка "к уничтожению", см. выше
)

// QueryTabler - то, у чего есть QueryTable (orm.Ormer и orm.TxOrmer).
type QueryTabler interface {
	QueryTable(ptrStructOrTableName interface{}) orm.QuerySeter
}

// ServerSprQS - выборка server_spr_records БЕЗ записей, помеченных к
// уничтожению (Stat == ServerSprStatDestroy). Использовать вместо
// o.QueryTable("server_spr_records") во всех запросах.
//
// ⚠️ SetCond заменяет все ранее заданные условия (beego querySet.SetCond),
// поэтому для запросов с SetCond условие нужно собирать через
// ExcludeDestroyedCond, а не полагаться на этот фильтр.
func ServerSprQS(o QueryTabler) orm.QuerySeter {
	return o.QueryTable("server_spr_records").Exclude("Stat", ServerSprStatDestroy)
}

// ExcludeDestroyedCond оборачивает условие так, чтобы записи с
// Stat == ServerSprStatDestroy были исключены: (cond) AND NOT stat = 666.
// Оборачивание в подгруппу нужно, чтобы условие с OR (a OR b) не
// превратилось в "a OR (b AND NOT stat=666)".
func ExcludeDestroyedCond(cond *orm.Condition) *orm.Condition {
	if cond == nil || cond.IsEmpty() {
		return orm.NewCondition().AndNot("Stat", ServerSprStatDestroy)
	}
	return orm.NewCondition().AndCond(cond).AndNot("Stat", ServerSprStatDestroy)
}

// IsServerSprDestroyed - запись помечена к уничтожению; после чтения по
// ключу такую запись нужно считать отсутствующей и не изменять.
func IsServerSprDestroyed(s *ServerSprRecords) bool {
	return s != nil && s.Stat == ServerSprStatDestroy
}
