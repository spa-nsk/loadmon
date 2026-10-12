// Package vnxmock — заглушка naviseccli для тестов и стенда (plan.md E5.6): исполняемый bash-скрипт,
// который печатает подготовленный вывод getagent/getdisk/storagepool/getlun из файлов рядом с собой.
// Так проверяется настоящий запуск внешнего процесса. Формат вывода — по документации и практике VNX,
// не проверен на живом массиве (harness/14-source-storage.md).
package vnxmock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cmdb/internal/importer/connectors/storage"
)

// Array — состояние заглушки; после изменения полей вызвать Write.
type Array struct {
	Dir    string
	Name   string
	Serial string
	Model  string
	Disks  []Disk
	Pools  []Pool
	LUNs   []LUN
	Groups []Group
	Fail   bool
}

// Group — группа хранения: HBA UID (WWNN:WWPN) и номера презентованных LUN (ALU).
type Group struct {
	Name string
	HBAs []string
	ALUs []int
}

type Disk struct {
	Bus, Encl int
	Slot      string
	Vendor    string
	Product   string
	Serial    string
	MB        int
	Type      string
	State     string
}

type Pool struct {
	Name string
	ID   int
	GBs  float64
}

type LUN struct {
	N    int
	Name string
	UID  string
	GBs  float64
	Thin bool
	Pool string
}

const script = `#!/bin/bash
dir="$(dirname "$0")"
echo "$*" >> "$dir/calls.log"
if [ -f "$dir/FAIL" ]; then echo "Error: could not connect to the array" >&2; exit 1; fi
case "$*" in
  *getagent*) cat "$dir/getagent.txt" ;;
  *getdisk*) cat "$dir/getdisk.txt" ;;
  *"storagegroup -list"*) cat "$dir/storagegroup.txt" ;;
  *"storagepool -list"*) cat "$dir/storagepool.txt" ;;
  *getlun*) cat "$dir/getlun.txt" ;;
  *) echo "unknown command: $*" >&2; exit 2 ;;
esac
`

// Default — массив с 1 полкой (3 диска, один слот пуст), пулом и 3 LUN (два в пуле, один тонкий).
// tag (10 hex) делает серийные номера, UID и имена уникальными в общей тестовой БД.
func Default(dir, tag string) *Array {
	uid := func(n int) string {
		return "60:06:01:60:" + tag[0:2] + ":" + tag[2:4] + ":" + tag[4:6] + ":" + tag[6:8] + ":aa:bb:cc:dd:00:00:00:0" + fmt.Sprint(n)
	}
	return &Array{
		Groups: []Group{{"SG_esx_" + tag, []string{"20:00:00:00:00:00:00:02:" + storage.SampleWWPN(tag, 6)}, []int{0, 1}}},
		Dir:    dir, Name: "vnx-" + tag, Serial: "APM00" + tag, Model: "VNX5300",
		Disks: []Disk{
			{0, 0, "A0", "SEAGATE", "ST3300657SS", "S0" + tag, 549691, "SAS", "Enabled"},
			{0, 0, "A1", "HITACHI", "HUS156045", "S1" + tag, 549691, "NL SAS", "Enabled"},
			{0, 0, "A2", "SAMSUNG", "SSD800", "S2" + tag, 762000, "SAS Flash", "Enabled"},
			{0, 0, "A3", "", "", "", 0, "", "Removed"},
		},
		Pools: []Pool{{"Pool 0", 0, 1000}},
		LUNs: []LUN{
			{0, "lun-a-" + tag, uid(1), 100, true, "Pool 0"},
			{1, "lun-b-" + tag, uid(2), 200, false, "Pool 0"},
			{2, "lun-rg-" + tag, uid(3), 50, false, "N/A"},
		},
	}
}

// Write записывает скрипт и файлы вывода; Script возвращает путь к скрипту.
func (a *Array) Write() error {
	if err := os.WriteFile(filepath.Join(a.Dir, "naviseccli"), []byte(script), 0o755); err != nil {
		return err
	}
	files := map[string]string{"getagent.txt": a.agentText(), "getdisk.txt": a.diskText(), "storagepool.txt": a.poolText(), "getlun.txt": a.lunText(), "storagegroup.txt": a.groupText()}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(a.Dir, name), []byte(text), 0o644); err != nil {
			return err
		}
	}
	fail := filepath.Join(a.Dir, "FAIL")
	if a.Fail {
		return os.WriteFile(fail, nil, 0o644)
	}
	if err := os.Remove(fail); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Script — путь к исполняемой заглушке.
func (a *Array) Script() string { return filepath.Join(a.Dir, "naviseccli") }

// Calls возвращает записанные вызовы (по строке на вызов).
func (a *Array) Calls() []string {
	b, _ := os.ReadFile(filepath.Join(a.Dir, "calls.log"))
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func (a *Array) agentText() string {
	return fmt.Sprintf("Agent Rev:           7.33.9 (2.99)\nName:                %s\nDescr:               \nNode:                A-%s\nPhysical Node:       K10\nSignature:           1234\nRevision:            05.33.009.5.155\nSCSI Id:             0\nModel:               %s\nModel Type:          Rackmount\nSerial No:           %s\n", a.Name, a.Name, a.Model, a.Serial)
}

func (a *Array) diskText() string {
	var sb strings.Builder
	for _, d := range a.Disks {
		fmt.Fprintf(&sb, "Bus %d Enclosure %d  Disk %s\nVendor Id:           %s\nProduct Id:          %s\nProduct Revision:    ES64\nLun:                 Unbound\nType:                N/A\nState:               %s\nHot Spare:           NO\nSerial Number:       %s\nCapacity:            %d\nDrive Type:          %s\n\n",
			d.Bus, d.Encl, d.Slot, d.Vendor, d.Product, d.State, d.Serial, d.MB, d.Type)
	}
	return sb.String()
}

func (a *Array) poolText() string {
	var sb strings.Builder
	for _, p := range a.Pools {
		fmt.Fprintf(&sb, "Pool Name:  %s\nPool ID:  %d\nRAID Type:  r_5\nPercent Full Threshold:  70\nDescription:  \nDisk Drive Types:  SAS\nState:  Ready\nStatus:  OK(0x0)\nRaw Capacity (GBs):  %.3f\nUser Capacity (GBs):  %.3f\n\n", p.Name, p.ID, p.GBs*1.2, p.GBs)
	}
	return sb.String()
}

func (a *Array) lunText() string {
	var sb strings.Builder
	for _, l := range a.LUNs {
		thin := "NO"
		if l.Thin {
			thin = "YES"
		}
		fmt.Fprintf(&sb, "LOGICAL UNIT NUMBER %d\nName:  %s\nUID:  %s\nCurrent Owner:  SP A\nDefault Owner:  SP A\nUser Capacity (GBs):  %.3f\nRAID Type:  RAID5\nState:  Bound\nIs Thin LUN:  %s\nPool Name:  %s\n\n", l.N, l.Name, l.UID, l.GBs, thin, l.Pool)
	}
	return sb.String()
}

func (a *Array) groupText() string {
	var sb strings.Builder
	for _, g := range a.Groups {
		fmt.Fprintf(&sb, "Storage Group Name:    %s\nStorage Group UID:     11:22:33\nHBA/SP Pairs:\n\n  HBA UID                                          SP Name     SPPort\n  -------                                          -------     ------\n", g.Name)
		for _, h := range g.HBAs {
			fmt.Fprintf(&sb, "  %s   SP A         0\n", strings.ToUpper(h))
		}
		sb.WriteString("\nHLU/ALU Pairs:\n\n  HLU Number     ALU Number\n  ----------     ----------\n")
		for i, alu := range g.ALUs {
			fmt.Fprintf(&sb, "    %d               %d\n", i, alu)
		}
		sb.WriteString("Shareable:             YES\n\n")
	}
	return sb.String()
}
