package cmdb

import (
	"encoding/json"
	"testing"

	oneview "github.com/spa-nsk/oneview2"
)

// Фрагмент ответа /rest/enclosures по примеру из swagger HPE OneView API 300.
const enclosureSwaggerJSON = `{
 "name": "c7000-1", "enclosureType": "C7000",
 "interconnectBays": [
  {"bayNumber":1,"interconnectUri":"/rest/interconnects/a","logicalInterconnectUri":"/rest/logical-interconnects/x","interconnectModel":"HP VC FlexFabric-20/40 F8 Module","interconnectBayType":"C7000InterconnectBay","serialNumber":"WECFSED100"},
  {"bayNumber":2,"interconnectUri":"/rest/interconnects/b","interconnectModel":"HP VC FlexFabric-20/40 F8 Module","serialNumber":"WECFSED101"},
  {"bayNumber":3,"interconnectUri":null,"interconnectModel":null,"interconnectBayType":"C7000InterconnectBay","serialNumber":null}
 ],
 "managerBays": [
  {"ipAddress":"172.18.1.11","devicePresence":"Present","managerType":"OA"},
  {"ipAddress":"172.18.1.12","devicePresence":"Present","managerType":"OA"},
  {"devicePresence":"Absent","managerType":"OA"}
 ]
}`

func TestBuildInterconnectComponents_Swagger(t *testing.T) {
	var enc oneview.Enclosure
	if err := json.Unmarshal([]byte(enclosureSwaggerJSON), &enc); err != nil {
		t.Fatal(err)
	}
	// Без /rest/interconnects: модель из interconnectModel, пустой отсек пропущен.
	got := buildInterconnectComponents(&enc, nil)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2 занятых отсека, получено %d", len(got))
	}
	if got[0].Model != "HP VC FlexFabric-20/40 F8 Module" || got[0].SerialNum != "WECFSED100" {
		t.Errorf("неверные модель/серийный: %+v", got[0])
	}
	if got[0].Slot != "Bay 1, Present" {
		t.Errorf("Slot = %q", got[0].Slot)
	}

	// С /rest/interconnects: статус берётся оттуда.
	ics := map[string]oneview.Interconnect{"/rest/interconnects/a": {}}
	ic := ics["/rest/interconnects/a"]
	ic.URI = "/rest/interconnects/a"
	ic.Status = "OK"
	ics["/rest/interconnects/a"] = ic
	got = buildInterconnectComponents(&enc, ics)
	if got[0].Slot != "Bay 1, OK" {
		t.Errorf("Slot с /rest/interconnects = %q", got[0].Slot)
	}
}

func TestBuildManagerComponents_Swagger(t *testing.T) {
	var enc oneview.Enclosure
	if err := json.Unmarshal([]byte(enclosureSwaggerJSON), &enc); err != nil {
		t.Fatal(err)
	}
	got := buildManagerComponents(&enc)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2 присутствующих OA, получено %d", len(got))
	}
	if got[0].Model != "Onboard Administrator" {
		t.Errorf("Model = %q", got[0].Model)
	}
	if got[0].Slot != "Bay 1, IP 172.18.1.11" || got[1].Slot != "Bay 2, IP 172.18.1.12" {
		t.Errorf("Slot: %q / %q", got[0].Slot, got[1].Slot)
	}
}

func TestManagerTypeTitle(t *testing.T) {
	if managerTypeTitle("FLM", "") != "Synergy Frame Link Module" {
		t.Error("FLM")
	}
	if managerTypeTitle("", "Synergy") != "Synergy Frame Link Module" {
		t.Error("пустой тип у Synergy")
	}
	if managerTypeTitle("", "C7000") != "Onboard Administrator" {
		t.Error("пустой тип у c7000")
	}
}
