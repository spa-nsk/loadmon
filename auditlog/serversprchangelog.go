// Package auditlog содержит журналирование ручных изменений карточки
// сервера (models.ServerSprRecords) с автором и датой
// (models.ServerSprEventLog, см. models/models_add.go).
//
// Вынесено в отдельный ЛИСТОВОЙ пакет (2026-09-04), а не в cmdb, где
// изначально и было написано: cmdb уже имеет обратную зависимость от
// controllers через `_ "loadmon/routers"` (blank-import в нескольких
// файлах cmdb, например cmdb/initrec.go) -> routers импортирует
// controllers -> если controllers импортирует cmdb напрямую (что и
// потребовалось бы, чтобы контроллеры /sprsrv/updatebase и
// /sprsrv/addserver могли вызывать эти функции), получается цикл импортов
// (`import cycle not allowed`, воспроизведено при `go build`). auditlog не
// импортирует ни cmdb, ни routers, ни controllers - только models и orm,
// поэтому его может использовать и cmdb (для симметрии, если понадобится),
// и controllers.
package auditlog

import (
	"fmt"
	models "loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ServerSprEventLogRetention - срок хранения журнала изменений карточки
// сервера (models.ServerSprEventLog) - 180 суток. Это журнал РУЧНЫХ правок
// с указанием автора (аудит "кто и когда менял карточку"), а не журнал
// каждой попытки автоматической синхронизации - событий заведомо мало
// (правит человек через форму), поэтому храним намного дольше, чем
// cmdb.HardwareEventLogRetention (30 суток) или
// cmdb.StorageEventLogRetention (3 суток).
const ServerSprEventLogRetention = time.Hour * 24 * 180

// ResolveUserDisplayName возвращает читаемое имя пользователя
// (models.AuthUser: "Фамилия Имя", с запасными вариантами Email/Id) по его
// Id - используется и для записи "снимка" имени автора в
// ServerSprEventLog.UserName (см. LogServerSprRecordChange/
// LogServerSprRecordCreated), и контроллером карточки сервера
// (controllers/sprsrvupdatebase.go) для отображения "Автор последних
// изменений". Id=0 (системное/неавторизованное изменение) - возвращает
// пустую строку.
func ResolveUserDisplayName(o orm.Ormer, userId int64) string {
	if userId <= 0 {
		return ""
	}
	var u models.AuthUser
	u.Id = int(userId)
	if err := o.Read(&u); err != nil {
		return fmt.Sprintf("Id=%d", userId)
	}
	name := strings.TrimSpace(u.Last + " " + u.First)
	if name != "" {
		return name
	}
	if u.Email != "" {
		return u.Email
	}
	return fmt.Sprintf("Id=%d", userId)
}

// serverSprFieldsToTrack - список отслеживаемых для diff'а полей карточки
// сервера (models.ServerSprRecords) и их читаемые названия. Не ВСЕ поля
// структуры - только те, что реально редактируются через форму
// /sprsrv/updatebase (см. controllers/sprsrvupdatebase.go) и представляют
// интерес для истории изменений; служебные поля (Id, OldId, VMSprId/
// HWSprId/ZabbixHostId, метрики CPU/Mem/Disk, Time_Mfy/Id_User сами по себе)
// не отслеживаются.
var serverSprFieldsToTrack = []struct {
	name string
	get  func(r *models.ServerSprRecords) string
}{
	{"Name", func(r *models.ServerSprRecords) string { return r.Name }},
	{"NameFQDN", func(r *models.ServerSprRecords) string { return r.NameFQDN }},
	{"IPAddress", func(r *models.ServerSprRecords) string { return r.IPAddress }},
	{"SerialNumber", func(r *models.ServerSprRecords) string { return r.SerialNumber }},
	{"Vcenter", func(r *models.ServerSprRecords) string { return r.Vcenter }},
	{"Cluster", func(r *models.ServerSprRecords) string { return r.Cluster }},
	{"ITSystemKey", func(r *models.ServerSprRecords) string { return r.ITSystemKey }},
	{"OS", func(r *models.ServerSprRecords) string { return r.OS }},
	{"Owner", func(r *models.ServerSprRecords) string { return r.Owner }},
	{"BusinessOwner", func(r *models.ServerSprRecords) string { return r.BusinessOwner }},
	{"ITOwner", func(r *models.ServerSprRecords) string { return r.ITOwner }},
	{"ProductOwner", func(r *models.ServerSprRecords) string { return r.ProductOwner }},
	{"Support", func(r *models.ServerSprRecords) string { return r.Support }},
	{"SupportTeam", func(r *models.ServerSprRecords) string { return r.SupportTeam }},
	{"Role", func(r *models.ServerSprRecords) string { return r.Role }},
	{"CriticalLevel", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.CriticalLevel) }},
	{"TypeWork", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.TypeWork) }},
	{"Active", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.Active) }},
	{"Monitoring", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.Monitoring) }},
	{"Backuped", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.Backuped) }},
	{"DPCOffice", func(r *models.ServerSprRecords) string { return strconv.Itoa(r.DPCOffice) }},
	{"Details", func(r *models.ServerSprRecords) string { return r.Details }},
	{"Anotation", func(r *models.ServerSprRecords) string { return r.Anotation }},
}

// LogServerSprRecordChange сравнивает состояние карточки сервера ДО
// (before) и ПОСЛЕ (after) правки через веб-форму и пишет в
// server_spr_event_log одну строку на каждое реально изменившееся
// отслеживаемое поле (см. serverSprFieldsToTrack). Вызывается из
// controllers/sprsrvupdatebase.go непосредственно перед o.Update(&record) -
// after.Id_User на этот момент уже содержит Id редактировавшего
// пользователя (см. вызов в контроллере).
func LogServerSprRecordChange(o orm.Ormer, before, after *models.ServerSprRecords) {
	if before == nil || after == nil || after.Id <= 0 {
		return
	}
	userName := ResolveUserDisplayName(o, after.Id_User)
	now := time.Now().Local()
	for _, f := range serverSprFieldsToTrack {
		oldVal, newVal := f.get(before), f.get(after)
		if oldVal == newVal {
			continue
		}
		entry := &models.ServerSprEventLog{
			Time: now, EventType: models.ServerSprEventFieldChanged,
			ServerId: after.Id, Key: after.Name,
			FieldName: f.name, OldValue: oldVal, NewValue: newVal,
			UserId: after.Id_User, UserName: userName,
			Message: fmt.Sprintf("%s: поле %s изменилось с %q на %q", after.Name, f.name, oldVal, newVal),
		}
		if _, err := o.Insert(entry); err != nil {
			fmt.Println("Ошибка записи журнала изменений карточки сервера:", entry.Message, err)
		}
	}
}

// LogServerSprRecordCreated пишет в server_spr_event_log факт создания новой
// карточки сервера вручную (controllers/sprsrvaddserver.go, после
// успешного o.Insert - record.Id уже присвоен).
func LogServerSprRecordCreated(o orm.Ormer, rec *models.ServerSprRecords) {
	if rec == nil || rec.Id <= 0 {
		return
	}
	userName := ResolveUserDisplayName(o, rec.CreatorUser)
	entry := &models.ServerSprEventLog{
		Time: time.Now().Local(), EventType: models.ServerSprEventCreated,
		ServerId: rec.Id, Key: rec.Name,
		UserId: rec.CreatorUser, UserName: userName,
		Message: fmt.Sprintf("Карточка сервера \"%s\" создана вручную", rec.Name),
	}
	if _, err := o.Insert(entry); err != nil {
		fmt.Println("Ошибка записи журнала создания карточки сервера:", entry.Message, err)
	}
}

// ClearOldServerSprEventLog удаляет из таблицы server_spr_event_log записи
// старше ServerSprEventLogRetention (180 суток) - тот же приём порционного
// удаления, что и cmdb.ClearOldHardwareEventLog/cmdb.ClearOldStorageEventLog.
func ClearOldServerSprEventLog() {
	cutoff := time.Now().Local().Add(-ServerSprEventLogRetention)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw("DELETE FROM server_spr_event_log WHERE id IN (SELECT id FROM server_spr_event_log WHERE time <=? LIMIT 1000)", cutoff).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы server_spr_event_log", err)
			break
		}
		num, _ := res.RowsAffected()
		count += num
		if num == 0 {
			break
		}
	}
	fmt.Println("Удалено :", count, "записей старее", ServerSprEventLogRetention.String(), "из таблицы server_spr_event_log")
}
