package dorado

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Disk - физический диск СХД Huawei Dorado/OceanStor (объект типа TypeDisk=10
// в терминологии DeviceManager REST API). Добавлено 2026-09-16 по запросу
// пользователя loadmon "расширить SDK для сбора дисков/полок" (в отличие от
// StoragePool - логического пула, уже поддержанного этим SDK, - Disk
// описывает конкретную физическую единицу хранения в конкретном слоте
// конкретной полки).
//
// Имена полей и их типы взяты из публичной документации Huawei DeviceManager
// REST API (объект "disk") - как и у остальных типов этого SDK
// (см. StoragePool/System), почти все значения приходят строками даже для
// числовых величин. ⚠️ Конкретные названия полей могут отличаться в
// зависимости от версии прошивки СХД - при необходимости скорректировать
// после первого реального запроса к устройству (см. комментарий
// cmdb.syncHuaweiHardware).
type Disk struct {
	ID               string `json:"ID"`
	NAME             string `json:"NAME"`
	SERIALNUMBER     string `json:"SERIALNUMBER"`
	MANUFACTURER     string `json:"MANUFACTURER"`
	MODEL            string `json:"MODEL"`
	HEALTHSTATUS     string `json:"HEALTHSTATUS"`
	RUNNINGSTATUS    string `json:"RUNNINGSTATUS"`
	DISKTYPE         string `json:"DISKTYPE"` // тип носителя (SSD/SAS/NL_SAS и т.п.)
	CAPACITY         string `json:"CAPACITY"` // в секторах по 512 байт, см. DiskCapacityGb ниже
	PARENTID         string `json:"PARENTID"` // Id родительской полки (Enclosure.ID) - для увязки диска с полкой
	PARENTTYPE       int    `json:"PARENTTYPE"`
	ENCLOSUREID      string `json:"ENCLOSUREID"` // на некоторых прошивках дублирует PARENTID отдельным полем
	SLOTNUM          string `json:"SLOTNUM"`     // номер слота в полке
	POSITION         string `json:"POSITION"`
	SPEED            string `json:"SPEED"`
	LOGICTYPE        string `json:"LOGICTYPE"` // 0-обычный, 1-hot spare и т.п.
	FIRMWAREVERSION  string `json:"FIRMWAREVERSION"`
}

// DiskCapacityGb переводит Disk.CAPACITY (сектора по 512 байт - тот же
// формат, что и у остальных объёмных полей Huawei REST API, см.
// GetHuaweyVolume/GetDoradoLun в cmdb) в гигабайты.
func (d Disk) DiskCapacityGb() float64 {
	sectors, err := parseHuaweiFloat(d.CAPACITY)
	if err != nil {
		return 0
	}
	return sectors * 512 / (1024 * 1024 * 1024)
}

// GetDisks возвращает список всех физических дисков СХД. query может
// уточнять выборку (например, по PARENTID конкретной полки) - см.
// AddSearchQuery/SearchQuery в query.go, тот же приём, что у GetStoragePools.
func (d *Device) GetDisks(ctx context.Context, query *SearchQuery) ([]Disk, error) {
	spath := "/disk"
	req, err := d.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, fmt.Errorf(ErrCreateRequest+": %w", err)
	}
	req = AddSearchQuery(req, query)

	var disks []Disk
	if err = d.requestWithRetry(req, &disks, DefaultHTTPRetryCount); err != nil {
		return nil, fmt.Errorf(ErrRequestWithRetry+": %w", err)
	}

	if len(disks) == 0 {
		return nil, ErrDiskNotFound
	}

	return disks, nil
}

// parseHuaweiFloat разбирает строковое числовое поле, как их присылает
// Huawei DeviceManager REST API (почти все объёмные/числовые величины
// приходят строками) - общий небольшой helper для Disk.DiskCapacityGb.
func parseHuaweiFloat(raw string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(raw), 64)
}
