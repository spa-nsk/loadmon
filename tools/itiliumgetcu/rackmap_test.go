package main

import (
	"encoding/json"
	"testing"

	"loadmon/itilium"
)

func TestParsePlacementAndLocation(t *testing.T) {
	p := parsePlacement("6.2-A-04 | 23-23")
	if !p.OK || p.Cabinet != "6.2-A-04" || p.StartUnit != 23 || p.SizeUnit != 1 {
		t.Fatalf("%+v", p)
	}
	p2 := parsePlacement("9.2-A-04 | 23-25")
	if !p2.OK || p2.SizeUnit != 3 {
		t.Fatalf("%+v", p2)
	}
	loc := parseLocation("Новосибирск, КЦ (на Советской) | 6 | 6.2 (623)")
	if loc.SiteText == "" || loc.ZoneCode != "6" || loc.RoomCode != "6.2 (623)" {
		t.Fatalf("%+v", loc)
	}
}

func TestMapRackServerFromSample3(t *testing.T) {
	const raw = `{
		"author": {"label":"Автор","value":"Одежкин Андрей Андреевич"},
		"code": {"label":"Инв.номер IT","value":"IT-0000001425"},
		"cuClass": "145",
		"deleted": false,
		"description": {"label":"Описание","value":"HP DL360 Gen10"},
		"hostname": {"label":"Имя (hostname)","value":"S-VEEAMBKP-05"},
		"ip": {"label":"ip","value":"10.2.21.24"},
		"linksUnit": "6.2-A-01 | 9-9",
		"modelServ": {"label":"Марка/модель (Норматив)","value":"HPE ProLiant DL360 Gen10 PK Veeam proxy"},
		"modelVendS": {"label":"Модель от производителя","value":"ProLiant DL360 Gen10"},
		"name": {"label":"Стар.инв.№","value":"BCS_S_205"},
		"rowKey": "IT-0000001425",
		"sn": {"label":"s/n","value":"CZJ02604DM"},
		"status": {"label":"В работе (эксплуатации)","value":"3"},
		"statusHistory": "10.02.2023",
		"statusHistoryYY": "2023-02-10",
		"uid": "IT-0000001425",
		"units": {"label":"Высота в юнитах","value":1},
		"user": {"label":"Пользователь","value":"Пушкин Александр Евгеньевич"},
		"where": {"label":"Местоположение","value":"Новосибирск, КЦ (на Советской) | 6 | 6.2 (623)"}
	}`
	var row itilium.CURow
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	v := MapRackServer(row)
	if v.TechNumberIT != "IT-0000001425" || v.Hostname != "S-VEEAMBKP-05" || v.SerialNum != "CZJ02604DM" {
		t.Fatalf("ids: %+v", v)
	}
	if v.OldInvNum != "BCS_S_205" || v.VendorModel != "ProLiant DL360 Gen10" {
		t.Fatalf("inv/model: %+v", v)
	}
	if !v.StatusKnown || v.Active != 1 || v.ItiliumStatus != 1 || v.PowerState != 1 {
		t.Fatalf("status: %+v", v)
	}
	if v.Placement.Cabinet != "6.2-A-01" || v.Placement.StartUnit != 9 {
		t.Fatalf("place: %+v", v)
	}
	if v.Location.ZoneCode != "6" || v.UnitsH != 1 || v.EffectiveSizeUnit() != 1 {
		t.Fatalf("loc/units: %+v", v)
	}
}

func TestMapStatusUpdateYAML(t *testing.T) {
	st, ok := mapStatus("Черновик")
	if !ok || st.ItiliumStatus != 9 || st.Active != 2 || st.Power != 2 {
		t.Fatalf("Черновик: %+v ok=%v", st, ok)
	}
	st, ok = mapStatus("Списан")
	if !ok || st.ItiliumStatus != 3 || st.Stat != 2 || st.Active != 2 {
		t.Fatalf("Списан: %+v ok=%v", st, ok)
	}
	st, ok = mapStatus("Свободно")
	if !ok || st.ItiliumStatus != 4 || st.Active != 2 {
		t.Fatalf("Свободно (в update/ — Active=2): %+v ok=%v", st, ok)
	}
	st, ok = mapStatus("В работе (эксплуатация)")
	if !ok || st.ItiliumStatus != 1 || st.Active != 1 {
		t.Fatalf("вариант без «и»: %+v ok=%v", st, ok)
	}
}
