package main

import (
	"encoding/json"
	"testing"

	"loadmon/itilium"
)

func TestValidateDumpSampleOKAndQuestions(t *testing.T) {
	rows := []itilium.CURow{}
	must := func(raw string) itilium.CURow {
		var r itilium.CURow
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	rows = append(rows, must(`{
		"code":{"value":"IT-0000001425"},"uid":"IT-0000001425","cuClass":"145",
		"hostname":{"value":"S-VEEAMBKP-05"},"ip":{"value":"10.2.21.24"},
		"sn":{"value":"CZJ02604DM"},"modelVendS":{"value":"ProLiant DL360 Gen10"},
		"status":{"label":"В работе (эксплуатации)","value":"3"},
		"linksUnit":"6.2-A-01 | 9-9",
		"where":{"value":"Новосибирск, КЦ (на Советской) | 6 | 6.2 (623)"},
		"units":{"value":1}
	}`))
	rows = append(rows, must(`{
		"code":{"value":"IT-0000001000"},"uid":"IT-0000001000",
		"hostname":{"value":"test"},"ip":{"value":"вввв"},
		"sn":{"value":""},
		"status":{"label":"Черновик","value":"12"},
		"where":{"value":"Москва, пр-т Мира, 69 | Склад | Неразмещенные"}
	}`))
	rows = append(rows, must(`{
		"code":{"value":"IT-DUP"},"uid":"IT-DUP-A","sn":{"value":"SAME"},
		"status":{"label":"Списан","value":"8"},"hostname":{"value":"a"}
	}`))
	rows = append(rows, must(`{
		"code":{"value":"IT-DUP2"},"uid":"IT-DUP-B","sn":{"value":"SAME"},
		"status":{"label":"Списан","value":"8"},"hostname":{"value":"b"}
	}`))

	rep, q := validateDump(rows, "test")
	if rep.TotalRows != 4 {
		t.Fatalf("total=%d", rep.TotalRows)
	}
	if q.ProblemRows == 0 {
		t.Fatal("ожидались вопросы (bad ip, missing sn у черновика ок, dup sn)")
	}
	if rep.ByCode["dup_serial"] < 2 {
		t.Fatalf("dup_serial: %+v", rep.ByCode)
	}
	if rep.ByCode["bad_ip"] < 1 {
		t.Fatalf("bad_ip: %+v", rep.ByCode)
	}
	// первая строка должна быть ok или только units soft
	ok0 := rep.Rows[0].OK
	for _, is := range rep.Rows[0].Issues {
		if is.Code != "units_mismatch" && is.Severity == "error" {
			ok0 = false
		}
	}
	if !rep.Rows[0].OK && len(rep.Rows[0].Issues) > 0 {
		// допустимы только question/warn без критичных для размещённого сервера
		for _, is := range rep.Rows[0].Issues {
			if is.Severity == sevError {
				t.Fatalf("sample3 unexpected error: %+v", rep.Rows[0].Issues)
			}
		}
	}
	_ = ok0
	if len(q.Items) == 0 || len(q.Items[0].Questions) == 0 {
		t.Fatalf("questions empty: %+v", q)
	}
}

func TestValidateUnknownStatusAndBadUnits(t *testing.T) {
	var row itilium.CURow
	raw := `{
		"code":{"value":"IT-1"},"uid":"IT-1",
		"status":{"label":"Неведомый статус","value":"99"},
		"sn":{"value":"X"},"hostname":{"value":"h"},
		"linksUnit":"CAB | abc",
		"where":{"value":"Новосибирск, тест | 1 | зал"}
	}`
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	rep, _ := validateDump([]itilium.CURow{row}, "t")
	codes := map[string]bool{}
	for _, is := range rep.Rows[0].Issues {
		codes[is.Code] = true
	}
	if !codes["unknown_status"] || !codes["placement_bad_units"] {
		t.Fatalf("%+v", codes)
	}
}
