package cmdb

// Журналирование ручных изменений карточки сервера (models.ServerSprRecords,
// models.ServerSprEventLog) перенесено в пакет loadmon/auditlog
// (auditlog/serversprchangelog.go) - 2026-09-04.
//
// Причина переноса: этот код должен вызываться из controllers/
// (sprsrvupdatebase.go, sprsrvaddserver.go), но cmdb уже имеет обратную
// зависимость от controllers через blank-import `_ "loadmon/routers"` в
// нескольких файлах пакета (initrec.go, updatehwhostinfo.go, updatepower.go,
// updatesprvm.go, volumestorage.go) - routers импортирует controllers.
// Если бы controllers импортировал cmdb напрямую (что и требовалось для
// вызова функций отсюда), получился бы цикл импортов:
// cmdb -> routers -> controllers -> cmdb ("import cycle not allowed",
// воспроизведено при `go build ./metrics_grpc`). Поэтому весь код перенесён
// в auditlog - лёгкий листовой пакет без зависимости от cmdb/routers/
// controllers, который может использовать и cmdb (не потребовалось), и
// controllers (используется).
//
// См. auditlog.ServerSprEventLogRetention, auditlog.ResolveUserDisplayName,
// auditlog.LogServerSprRecordChange, auditlog.LogServerSprRecordCreated,
// auditlog.ClearOldServerSprEventLog.
