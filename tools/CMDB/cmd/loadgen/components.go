package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// maxRowsPerInsert — предел строк на один многострочный INSERT: постраничный протокол Postgres
// ограничен 65535 параметрами на запрос — при 500 КЕ × 48 интерфейсов на сетевое устройство это
// превышалось бы почти в полтора раза (96000 параметров); поймано именно так, на первом реальном
// прогоне (`extended protocol limited to 65535 parameters`), не задано с запасом заранее.
const maxRowsPerInsert = 1000

// bulkInsertComponents — compsPerCI(class) компонентов на каждую КЕ пакета, циклически по
// componentTypesByClass (см. classes.go). Возвращает число вставленных строк.
func bulkInsertComponents(ctx context.Context, tx orm.TxOrmer, class string, ids []int64) int {
	n := compsPerCI(class)
	if n == 0 || len(ids) == 0 {
		return 0
	}
	type row struct {
		ciID          int64
		typ, slot, sn string
	}
	var rows []row
	for _, id := range ids {
		for i := 0; i < n; i++ {
			rows = append(rows, row{id, componentTypeAt(class, i, n), fmt.Sprintf("slot-%d", i), serialFor(id, i)})
		}
	}
	for start := 0; start < len(rows); start += maxRowsPerInsert {
		chunk := rows[start:min(start+maxRowsPerInsert, len(rows))]
		var sb strings.Builder
		sb.WriteString(`INSERT INTO cmdb.component (ci_id, component_type, slot, serial_number) VALUES `)
		args := make([]any, 0, len(chunk)*4)
		for i, r := range chunk {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("(?,?,?,?)")
			args = append(args, r.ciID, r.typ, r.slot, r.sn)
		}
		if _, err := tx.RawWithCtx(ctx, sb.String(), args...).Exec(); err != nil {
			fatalf("bulkInsertComponents %s: %v", class, err)
		}
	}
	return len(rows)
}

// bulkInsertInterfaces — ifacesPerCI(class) интерфейсов на каждую КЕ пакета: сети (ethernet) для
// серверов/гипервизоров, порты (ethernet) для сетевых устройств.
func bulkInsertInterfaces(ctx context.Context, tx orm.TxOrmer, class string, ids []int64) int {
	n := ifacesPerCI(class)
	if n == 0 || len(ids) == 0 {
		return 0
	}
	type row struct {
		ciID int64
		name string
	}
	var rows []row
	for _, id := range ids {
		for i := 0; i < n; i++ {
			rows = append(rows, row{id, fmt.Sprintf("eth%d", i)})
		}
	}
	for start := 0; start < len(rows); start += maxRowsPerInsert {
		chunk := rows[start:min(start+maxRowsPerInsert, len(rows))]
		var sb strings.Builder
		sb.WriteString(`INSERT INTO cmdb.interface (ci_id, name, if_type, enabled) VALUES `)
		args := make([]any, 0, len(chunk)*4)
		for i, r := range chunk {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString("(?,?,?,?)")
			args = append(args, r.ciID, r.name, "ethernet", true)
		}
		if _, err := tx.RawWithCtx(ctx, sb.String(), args...).Exec(); err != nil {
			fatalf("bulkInsertInterfaces %s: %v", class, err)
		}
	}
	return len(rows)
}
