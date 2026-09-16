package dorado

import (
	"context"
	"fmt"
)

// Enclosure - физическая полка/шкаф-контроллер СХД Huawei Dorado/OceanStor
// (объект типа TypeEnclosure=21 в терминологии DeviceManager REST API).
// Добавлено 2026-09-16 по запросу пользователя loadmon "расширить SDK для
// сбора дисков/полок" - аналог models.StorageEnclosure на стороне loadmon
// (см. cmdb.syncHuaweiHardware), объединяет и управляющие полки (DPE -
// Disk-Processor Enclosure, с контроллерами) и чисто дисковые (DAE - Disk
// Array Enclosure) - различаются по полю TYPE/MODEL, как и у Unity DPE/DAE
// в этом же проекте (см. storages.GetUnityDPE/GetUnityDAE).
//
// ⚠️ Конкретные названия полей могут отличаться в зависимости от версии
// прошивки СХД - при необходимости скорректировать после первого реального
// запроса к устройству (см. комментарий cmdb.syncHuaweiHardware).
type Enclosure struct {
	ID            string `json:"ID"`
	NAME          string `json:"NAME"`
	SERIALNUMBER  string `json:"SERIALNUMBER"`
	MANUFACTURER  string `json:"MANUFACTURER"`
	MODEL         string `json:"MODEL"`
	HEALTHSTATUS  string `json:"HEALTHSTATUS"`
	RUNNINGSTATUS string `json:"RUNNINGSTATUS"`
	// TYPE - подтип полки: различает управляющую полку с контроллерами
	// (DPE-подобная, содержит CTRL) от чисто дисковой полки расширения
	// (DAE-подобная) - конкретные числовые коды зависят от прошивки,
	// поэтому здесь не заводится отдельная константа-классификатор, текст
	// сохраняется в models.StorageEnclosure.Model/EnclosureType по факту
	// значения LOCATION/NAME (см. cmdb.syncHuaweiHardware).
	TYPE       string `json:"TYPE"`
	LOCATION   string `json:"LOCATION"`
	PARENTID   string `json:"PARENTID"` // Id родительской системы (StorageSystems по SerialNum, не по этому Id)
	SLOTNUM    string `json:"SLOTNUM"`
	BUSID      string `json:"BUSID"` // ⚠️ на части прошивок номер шины называется иначе (например MOVEDBUS) - скорректировать при необходимости после первого реального запроса
}

// GetEnclosures возвращает список всех физических полок/шкафов-контроллеров
// СХД. query может уточнять выборку - см. AddSearchQuery/SearchQuery в
// query.go, тот же приём, что у GetStoragePools/GetDisks.
func (d *Device) GetEnclosures(ctx context.Context, query *SearchQuery) ([]Enclosure, error) {
	spath := "/enclosure"
	req, err := d.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, fmt.Errorf(ErrCreateRequest+": %w", err)
	}
	req = AddSearchQuery(req, query)

	var enclosures []Enclosure
	if err = d.requestWithRetry(req, &enclosures, DefaultHTTPRetryCount); err != nil {
		return nil, fmt.Errorf(ErrRequestWithRetry+": %w", err)
	}

	if len(enclosures) == 0 {
		return nil, ErrEnclosureNotFound
	}

	return enclosures, nil
}
