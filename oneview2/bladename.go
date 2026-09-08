package oneview

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// bladeNameBayRe matches HPE OneView's naming convention for blade
// server-hardware resources: "<имя корзины>, bay <N>" (например
// "c7000-9, bay 15"). Пробелы вокруг запятой/после "bay" необязательны, регистр
// слова "bay" не важен - на разных версиях API это может отличаться.
var bladeNameBayRe = regexp.MustCompile(`(?i)^\s*(.+?)\s*,\s*bay\s*(\d+)\s*$`)

// ParseBladeNameBay разбирает имя ServerHardware-ресурса (Server.Identity.Name)
// на нормализованное имя корзины и номер отсека (bay), например
// "c7000-9, bay 15" -> ("c7000-09", 15, true).
//
// Смысл: у HPE OneView сами Name блейд-серверов УЖЕ содержат имя корзины и
// номер отсека - это приходит в ТОМ ЖЕ ответе /rest/server-hardware, что
// список серверов (Server.Identity.Name), без дополнительного запроса.
// Раньше единственным источником этой связи был Server.EnclosureInfo,
// заполняемый ОТДЕЛЬНЫМ вызовом GetEnclosure(hw.LocationURI) на каждый
// сервер (см. oneview2/export.go:exportHardware) - если этот вызов не
// отрабатывал (таймаут, протухший URI и т.п.), EnclosureInfo оставался
// пустым и связь терялась. Парсинг имени работает даже тогда.
//
// Для rack-серверов Name не матчится под этот шаблон - функция возвращает
// ok=false, что ожидаемо: у них нет корзины (см. FormFactor=1 ветку в
// cmdb/initrec.go, которая этот случай не трогает).
//
// Нормализация имени корзины ("c7000-9" -> "c7000-09", БЕЗ смены регистра) -
// та же схема %s-%02d, что уже используется в
// cmdb/syncbladesystemcomponents.go:bladeSystemNameFromEnclosure - специально
// без принудительного изменения регистра, чтобы не создать НОВОЕ расхождение
// с этими уже существующими источниками (запись в BladeSystems.Name может
// прийти из Итиллиум-импорта в любом регистре, который используется как
// эталон - см. itilliumimport/importblades.go).
func ParseBladeNameBay(name string) (chassis string, bay int, ok bool) {
	m := bladeNameBayRe.FindStringSubmatch(name)
	if m == nil {
		return "", 0, false
	}
	bayNum, err := strconv.Atoi(m[2])
	if err != nil || bayNum <= 0 {
		return "", 0, false
	}
	chassis = NormalizeEnclosureName(m[1])
	if chassis == "" {
		return "", 0, false
	}
	return chassis, bayNum, true
}

// EnclosureSummary - агрегат по одной блейд-корзине, собранный со стороны
// серверов (см. Infrastructure.EnclosureByName/EnclosureBySN и
// ParseBladeNameBay) - в отличие от Server.EnclosureInfo, который
// заполняется отдельным запросом GetEnclosure на каждый сервер и может
// оставаться пустым при его сбое, этот агрегат строится из уже собранного
// списка серверов и не требует ни одного дополнительного запроса.
type EnclosureSummary struct {
	Name          string          // нормализованное имя, см. NormalizeEnclosureName
	SerialNumber  string          // лучшее известное значение (см. комментарий у Infrastructure.EnclosureBySN) - может быть пустым
	Model         string          // то же самое - best-effort из EnclosureInfo любого сервера корзины
	EnclosureType string
	// Status/State - health-статус ("OK"/"Warning"/"Critical"/...) и
	// состояние ресурса ("Configured"/"Removed"/...) корзины, лучшее
	// известное значение (см. комментарий у SerialNumber выше) - как и
	// SerialNumber/Model/EnclosureType, заполняется только если хотя бы для
	// ОДНОГО сервера корзины отработал GetEnclosure (см. EnclosureInfo).
	// Используется в cmdb/initrec.go для простановки
	// BladeSystems.PowerState/Active при синхронизации с OneView (2026-09-08).
	Status string
	State  string
	ByBay  map[int]*Server // сервера корзины по номеру отсека (bay)
}

// NormalizeEnclosureName приводит имя корзины к единому виду "PREFIX-NN"
// (номер дополняется нулём слева до 2 цифр), НЕ трогая регистр - см.
// комментарий у ParseBladeNameBay про согласованность с
// cmdb/syncbladesystemcomponents.go:bladeSystemNameFromEnclosure. Имена без
// числового суффикса через дефис (или с суффиксом, который не парсится как
// число) возвращаются как есть (только TrimSpace).
func NormalizeEnclosureName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Split(name, "-")
	if len(parts) == 2 {
		if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			return fmt.Sprintf("%s-%02d", strings.TrimSpace(parts[0]), num)
		}
	}
	return name
}
