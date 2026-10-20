package itilliumimport

import (
	"regexp"
	"strconv"
	"strings"


)

// storagePlacementSegment - одно размещение СХД в конкретном шкафу, разобранное
// из ОДНОГО сегмента колонки "Размещение в шкафу/стойке" (после разбиения по
// запятой) - см. parseStoragePlacementSegments.
//
// ИСТОРИЯ БАГА (2026-09-10, по запросу пользователя "в импорте СХД, если
// несколько шкафов то загрузка некорректна указывается 1 шкаф с пустым
// диапазоном начала и конца, второй шкаф выпадает"): composite rack_unit_split
// (rack_storage.yaml/convertstorage.go:compositeRackUnitSplitStorage) читал
// колонку "Размещение в шкафу/стойке" КАК ОДНО значение и разбирал его на
// ровно ОДИН шкаф + ОДИН диапазон юнитов через SplitN(val, "|", 2) - при
// нескольких шкафах в одной ячейке (формат "шкаф1 | старт1-конец1, шкаф2 |
// старт2-конец2, ...", подтверждено анализом примера СХД_темпл2.xlsx)
// разбор ломался: unitRangeRe пытался распознать диапазон юнитов в остатке
// строки ПОСЛЕ первого "|" целиком, включая последующие шкафы через запятую -
// регулярка не совпадала, warn писался в лог, StartUnitPos/EndUnitPos
// оставались пустыми (0), а второй и последующие шкафы вообще не
// рассматривались (rack_unit_split извлекает только ОДНО значение rack_num).
//
// Функции в этом файле заменяют путь через rack_unit_split для размещения:
// вместо extra.rack_num/start_unit_pos/end_unit_pos (composite, недостаточен
// для нескольких шкафов) runImportItiliumStorage теперь разбирает СЫРУЮ
// строку extra.rack_placement_raw (composite concat_extra, УЖЕ существовавший
// для Anotation) напрямую в Go - см. parseStoragePlacementSegments ниже.
// rack_unit_split в rack_storage.yaml оставлен как есть (безвреден - его
// extra.rack_num/start_unit_pos/end_unit_pos просто больше нигде не
// читаются), убирать необязательно.
type storagePlacementSegment struct {
	// AltName - альтернативное наименование ЦОД/шкафа перед "/" в сегменте
	// (например "ЦОД M1" в "ЦОД M1 / R15-5"). Это НЕ отдельная площадка:
	// шкаф всегда принадлежит площадке строки (колонка "Местоположение"),
	// AltName сохраняется в RackCabinet.AltName (см. resolveRackCabinet).
	AltName string
	// CabinetCode - код/имя шкафа (например "3.1-A-08", "R15-5", "2.4.8").
	CabinetCode string
	StartUnitPos int
	EndUnitPos   int
}

// parseStoragePlacementSegments разбирает СЫРУЮ строку колонки "Размещение в
// шкафу/стойке" (raw, из extra.rack_placement_raw) на список сегментов
// размещения - один сегмент на каждую запись вида "[ЦОД_имя /] код_шкафа |
// старт-конец", разделённых запятой. Несколько сегментов с ОДИНАКОВЫМ (ЦОД,
// шкаф) - это несмежные диапазоны юнитов В ОДНОМ шкафу (см. анализ примера
// СХД_темпл2.xlsx: "5.1-A-04 | 9-22, 5.1-A-04 | 24-40") - они СХЛОПЫВАЮТСЯ в
// один сегмент с объединённым диапазоном min(start)..max(end), т.к. текущая
// схема models.StorageSystemPlacement хранит РОВНО один Start/End на пару
// (СХД, шкаф) - хранить несколько несмежных диапазонов в одном шкафу схема
// пока не позволяет (задел на будущее, если понадобится точность до
// отдельных несмежных промежутков).
//
// Сегменты, которые не удалось разобрать (нет "|", не распознан диапазон
// "начало-конец" через unitRangeRe) - пропускаются молча (warnings в этом
// месте не собираются, т.к. вызывается уже после ConvertRowStorage; при
// необходимости диагностировать такие строки - см. Anotation, куда сырое
// значение колонки попадает целиком в любом случае).
func parseStoragePlacementSegments(raw string) []storagePlacementSegment {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	type key struct {
		dc, cabinet string
	}
	order := make([]key, 0, 2)
	merged := map[key]*storagePlacementSegment{}

	for _, chunk := range strings.Split(raw, ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.SplitN(chunk, "|", 2)
		if len(parts) != 2 {
			continue
		}
		cabinetPart := strings.TrimSpace(parts[0])
		rangePart := strings.TrimSpace(parts[1])
		if cabinetPart == "" || rangePart == "" {
			continue
		}

		dcName := ""
		cabinetCode := cabinetPart
		if idx := strings.Index(cabinetPart, "/"); idx >= 0 {
			dcName = strings.TrimSpace(cabinetPart[:idx])
			cabinetCode = strings.TrimSpace(cabinetPart[idx+1:])
		}
		if cabinetCode == "" {
			continue
		}

		m := unitRangeRe.FindStringSubmatch(rangePart)
		if m == nil {
			continue
		}
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])

		k := key{dc: dcName, cabinet: cabinetCode}
		if seg, ok := merged[k]; ok {
			if start < seg.StartUnitPos {
				seg.StartUnitPos = start
			}
			if end > seg.EndUnitPos {
				seg.EndUnitPos = end
			}
			continue
		}
		merged[k] = &storagePlacementSegment{
			AltName: dcName,
			CabinetCode:        cabinetCode,
			StartUnitPos:       start,
			EndUnitPos:         end,
		}
		order = append(order, k)
	}

	if len(order) == 0 {
		return nil
	}
	result := make([]storagePlacementSegment, 0, len(order))
	for _, k := range order {
		result = append(result, *merged[k])
	}
	return result
}

// locationCabinetRe/locationUnitRe распознают токены "Шкаф <код>" и
// "Unit <число>[-<число>]" внутри extra["location_tail"] (см.
// compositeLocationSplit в convert.go) - формат встречается у части строк
// Итиллиум, где шкаф/юнит указаны ПРЯМО в колонке "Местоположение" (пример:
// "Новосибирск, ЦОД РТК, Менделеева 1 | Colocation | Зал 205 | Шкаф 2.4.4 |
// Unit 15"), а отдельная колонка "Размещение в шкафу/стойке" при этом пуста.
var locationCabinetRe = regexp.MustCompile(`(?i)^шкаф\s*(.+)$`)
var locationUnitRe = regexp.MustCompile(`(?i)^unit\s*(\d+)(?:\s*-\s*(\d+))?$`)

// parseLocationEmbeddedPlacement разбирает tail (extra["location_tail"],
// сегменты "Местоположение" после индекса 2, разделённые тем же separator,
// что и сам location_split) на код шкафа + диапазон юнитов - запасной
// источник размещения, когда parseStoragePlacementSegments (колонка
// "Размещение в шкафу/стойке") не дала ни одного сегмента. Если в tail
// указан только "Unit N" (одно число, без диапазона "N-M") - StartUnitPos ==
// EndUnitPos == N (один юнит, SizeUnit=1 - вычисляется дальше в
// upsertStorageSystemPlacement). Если "Шкаф" не найден - возвращает ok=false
// (нечем размещать, вызывающий код падает обратно на site-only размещение).
func parseLocationEmbeddedPlacement(tail string) (cabinetCode string, startUnit, endUnit int, ok bool) {
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return "", 0, 0, false
	}
	for _, chunk := range strings.Split(tail, "|") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if m := locationCabinetRe.FindStringSubmatch(chunk); m != nil {
			cabinetCode = strings.TrimSpace(m[1])
			continue
		}
		if m := locationUnitRe.FindStringSubmatch(chunk); m != nil {
			startUnit, _ = strconv.Atoi(m[1])
			if m[2] != "" {
				endUnit, _ = strconv.Atoi(m[2])
			} else {
				endUnit = startUnit
			}
		}
	}
	if cabinetCode == "" {
		return "", 0, 0, false
	}
	return cabinetCode, startUnit, endUnit, true
}
