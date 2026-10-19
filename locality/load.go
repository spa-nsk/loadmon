// Пакет locality загружает справочник населённых пунктов России (models.Locality)
// из открытых данных Росстата - ОКТМО (https://rosstat.gov.ru/opendata/7708234640-oktmo).
//
// Формат файла data-*.csv: разделитель ";", UTF-8, значения в кавычках, 13 колонок:
// TER;KOD1;KOD2;KOD3;KC;RAZDEL;NAME1;Centrum;NomDescr;NomAkt;Status;DateUtv;DateVved
// (см. itilliumimport/parsing/oktmo_structure.csv). RAZDEL=1 - муниципальные
// образования, RAZDEL=2 - населённые пункты (KOD3<>"000"). Код ОКТМО = TER+KOD1+KOD2+KOD3.
package locality

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

const batchSize = 500

// Parse читает файл ОКТМО и возвращает записи населённых пунктов.
func Parse(path string) ([]models.Locality, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = 13
	r.LazyQuotes = true

	regions := map[string]string{}        // TER -> название субъекта
	municipalities := map[string]string{} // TER+KOD1+KOD2 -> название МО
	var rows [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		ter, k1, k2, k3, razdel, name := rec[0], rec[1], rec[2], rec[3], rec[5], strings.TrimSpace(rec[6])
		switch {
		case razdel == "1" && k1 == "000" && k2 == "000" && k3 == "000":
			regions[ter] = strings.TrimSpace(strings.TrimPrefix(name, "Муниципальные образования"))
		case razdel == "1" && k3 == "000":
			// у строки "МО верхнего уровня" KOD2=000, у поселения внутри района - нет;
			// для справочника нужен верхний уровень (ключ TER+KOD1)
			if k2 == "000" {
				municipalities[ter+k1] = name
			}
		case razdel == "2" && k3 != "000":
			rows = append(rows, rec)
		}
	}

	out := make([]models.Locality, 0, len(rows))
	for _, rec := range rows {
		ter, k1, k2, k3 := rec[0], rec[1], rec[2], rec[3]
		full := strings.TrimSpace(rec[6])
		typ, name := "", full
		if i := strings.IndexByte(full, ' '); i > 0 {
			typ, name = full[:i], strings.TrimSpace(full[i+1:])
		}
		out = append(out, models.Locality{
			Oktmo:        ter + k1 + k2 + k3,
			RegionCode:   ter,
			RegionName:   regions[ter],
			Municipality: municipalities[ter+k1],
			TypeShort:    typ,
			Name:         name,
			FullName:     full,
		})
	}
	// В разделе 2 ОКТМО нет самих городов федерального значения (у них
	// только внутригородские МО) - добавляем их как города с кодом 8-значного
	// ОКТМО региона + "000".
	for ter, name := range federalCities {
		if _, ok := regions[ter]; !ok {
			continue
		}
		out = append(out, models.Locality{
			Oktmo:      ter + "000000" + "001",
			RegionCode: ter,
			RegionName: regions[ter],
			TypeShort:  "г",
			Name:       name,
			FullName:   "г " + name,
		})
	}
	return out, nil
}

// federalCities - города федерального значения: код региона -> название.
var federalCities = map[string]string{
	"45": "Москва",
	"40": "Санкт-Петербург",
	"67": "Севастополь",
}

// Load полностью перезаписывает таблицу cmdb_locality данными из файла ОКТМО
// (в одной транзакции). Связь с площадками идёт по коду ОКТМО, не по Id, поэтому
// пересоздание строк безопасно. Возвращает число загруженных записей.
func Load(path string) (int, error) {
	list, err := Parse(path)
	if err != nil {
		return 0, fmt.Errorf("разбор %s: %w", path, err)
	}
	if len(list) == 0 {
		return 0, fmt.Errorf("в %s нет населённых пунктов", path)
	}
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return 0, err
	}
	if _, err := tx.Raw("DELETE FROM cmdb_locality").Exec(); err != nil {
		tx.Rollback()
		return 0, err
	}
	for i := 0; i < len(list); i += batchSize {
		end := i + batchSize
		if end > len(list) {
			end = len(list)
		}
		if _, err := tx.InsertMulti(end-i, list[i:end]); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("вставка пачки %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(list), nil
}
