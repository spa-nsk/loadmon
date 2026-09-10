package itilliumimport

import (
	"strconv"
	"strings"
	"time"

	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
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
	// DatacenterSiteName - имя ЦОД ПЕРЕД "/" в сегменте (например "ЦОД
	// Невателеком", "ЦОД M1") - заполняется ТОЛЬКО для colocation-площадок,
	// где оно СВОЕ (может текстуально отличаться от имени ЦОД в колонке
	// "Местоположение" той же строки - подтверждено анализом примера, см.
	// CHANGELOG запись про анализ СХД_темпл2.xlsx). Пусто для собственных
	// площадок - там используется ЦОД, уже резолвленный для всей строки по
	// "Местоположение" (location_split).
	DatacenterSiteName string
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
			DatacenterSiteName: dcName,
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

// resolveDatacenterSiteByName ищет/заводит models.DatacenterSite СТРОГО по
// имени name (без склейки с этажом/комнатой, в отличие от
// runImportItiliumStorage, где Name площадки для "Местоположение" строится
// как datacenterRoom+" "+location_room_code) - используется ТОЛЬКО для
// colocation-сегментов "Размещение в шкафу/стойке" (ЦОД_имя перед "/"), где
// имя ЦОД - это уже готовый, самостоятельный идентификатор площадки (имя
// в реестре шкафов), а не текст адресной строки, который нужно было бы
// парсить дальше. См. комментарий у storagePlacementSegment.DatacenterSiteName.
func resolveDatacenterSiteByName(trn orm.TxOrmer, name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}
	site := models.DatacenterSite{Name: name}
	if err := trn.Read(&site, "Name"); err == nil {
		return site.Id, nil
	}
	site = models.DatacenterSite{
		Name:       name,
		Annotation: "Заведено автоматически импортом СХД из Итиллиум по имени ЦОД в колонке \"Размещение в шкафу/стойке\"",
		Id_User:    9999,
		Time_Mfy:   time.Now().Local(),
	}
	if _, err := trn.Insert(&site); err != nil {
		return 0, err
	}
	return site.Id, nil
}
