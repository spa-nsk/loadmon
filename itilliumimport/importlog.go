package itilliumimport

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// Журнал последнего SyncGetCU: буфер в памяти → InsertMulti; таблица
// itilium_import_log перед прогоном очищается (resetImportLog).

type importLogState struct {
	mu      sync.Mutex
	enabled bool
	cuClass int
	runTime time.Time
	buf     []models.ItiliumImportLog
}

var importLog = &importLogState{}

const importLogInsertChunk = 500

// resetImportLog очищает itilium_import_log и включает буфер для текущего прогона.
func resetImportLog() error {
	o := orm.NewOrm()
	if _, err := o.Raw("DELETE FROM itilium_import_log").Exec(); err != nil {
		// таблицы ещё может не быть до RunSyncdb — не блокируем sync
		log.Printf("itilliumimport.resetImportLog: DELETE: %v", err)
	}
	importLog.mu.Lock()
	defer importLog.mu.Unlock()
	importLog.enabled = true
	importLog.cuClass = 0
	importLog.runTime = time.Now().Local()
	importLog.buf = importLog.buf[:0]
	if importLog.buf == nil {
		importLog.buf = make([]models.ItiliumImportLog, 0, 1024)
	}
	return nil
}

// setImportLogCUClass задаёт cuClass для последующих recordImportOne.
func setImportLogCUClass(cuClass int) {
	importLog.mu.Lock()
	defer importLog.mu.Unlock()
	importLog.cuClass = cuClass
}

// appendImportLogRow добавляет произвольную строку (ошибка getCU и т.п.).
func appendImportLogRow(cuClass, result int, key, message string) {
	importLog.mu.Lock()
	defer importLog.mu.Unlock()
	if !importLog.enabled {
		return
	}
	importLog.buf = append(importLog.buf, models.ItiliumImportLog{
		Time:    importLog.runTime,
		CUClass: cuClass,
		Result:  result,
		Key:     truncateRunes(key, 256),
		Message: truncateRunes(message, 1024),
	})
}

// recordImportOne пишет исход одной КЕ в буфер журнала (если sync активен).
func recordImportOne(one cuImportOne) {
	importLog.mu.Lock()
	defer importLog.mu.Unlock()
	if !importLog.enabled {
		return
	}
	result := models.ItiliumImportResultUpdated
	switch {
	case one.err:
		result = models.ItiliumImportResultError
	case one.skipped:
		result = models.ItiliumImportResultSkipped
	case one.inserted:
		result = models.ItiliumImportResultInserted
	}
	msg := strings.TrimSpace(one.message)
	if msg == "" {
		msg = defaultImportResultMessage(result)
	}
	var notes []string
	if one.inactive {
		notes = append(notes, "inactive")
	}
	if one.noLoc {
		notes = append(notes, "noLoc")
	}
	if one.unknownSt {
		notes = append(notes, "unknownStatus")
	}
	if len(notes) > 0 {
		if msg != "" {
			msg = msg + "; " + strings.Join(notes, ",")
		} else {
			msg = strings.Join(notes, ",")
		}
	}
	key := strings.TrimSpace(one.key)
	if key == "" {
		key = "?"
	}
	importLog.buf = append(importLog.buf, models.ItiliumImportLog{
		Time:    importLog.runTime,
		CUClass: importLog.cuClass,
		Result:  result,
		Key:     truncateRunes(key, 256),
		Message: truncateRunes(msg, 1024),
	})
}

func defaultImportResultMessage(result int) string {
	switch result {
	case models.ItiliumImportResultInserted:
		return "добавлен"
	case models.ItiliumImportResultUpdated:
		return "обновлён"
	case models.ItiliumImportResultSkipped:
		return "пропуск"
	case models.ItiliumImportResultError:
		return "ошибка"
	default:
		return fmt.Sprintf("result=%d", result)
	}
}

// flushImportLog пишет буфер в БД и отключает журналирование.
func flushImportLog() {
	importLog.mu.Lock()
	buf := importLog.buf
	enabled := importLog.enabled
	importLog.enabled = false
	importLog.buf = nil
	importLog.mu.Unlock()
	if !enabled || len(buf) == 0 {
		return
	}
	o := orm.NewOrm()
	for i := 0; i < len(buf); i += importLogInsertChunk {
		end := i + importLogInsertChunk
		if end > len(buf) {
			end = len(buf)
		}
		chunk := buf[i:end]
		if _, err := o.InsertMulti(len(chunk), chunk); err != nil {
			log.Printf("itilliumimport.flushImportLog: InsertMulti[%d:%d]: %v", i, end, err)
		}
	}
	log.Printf("itilliumimport.flushImportLog: записано строк=%d", len(buf))
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
