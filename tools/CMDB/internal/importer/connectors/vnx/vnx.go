// Package vnx — коннектор Dell EMC VNX через CLI naviseccli (harness/03: единственный способ для VNX,
// работает в loadmon). Внешний бинарник запускается без оболочки (аргументы списком, без shell);
// разбираются getagent (массив), getdisk (диски и полки), storagepool -list -all (пулы), getlun (LUN).
//
// Аутентификация: предпочтительно файл безопасности naviseccli (-secfilepath, создаётся один раз через
// naviseccli -AddUserSecurity). Запасной вариант — username + password_env: naviseccli принимает пароль
// ТОЛЬКО аргументом командной строки, то есть он виден в списке процессов на хосте cmdb-worker — используйте
// его лишь там, где это допустимо. Формат вывода — по документации и практике VNX, НЕ проверен на живом массиве
// (harness/14-source-storage.md).
package vnx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "vnx"

func init() { connector.Register(Kind, New) }

// Runner выполняет команду naviseccli (без -h и учётных данных — их добавляет реализация).
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type execRunner struct {
	binary  string
	base    []string
	timeout time.Duration
}

func (r execRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.binary, append(append([]string{}, r.base...), args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", r.binary, args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// Client — коннектор одного массива VNX.
type Client struct {
	code   string
	runner Runner
	name   string // имя массива из настроек (необязательно)
}

// New создаёт коннектор. Настройки imp.connector.config: host (адрес SP A или SP B), binary (по умолчанию
// naviseccli), secfile_path либо username + password_env (+ scope, по умолчанию 0), array_name (имя КЕ
// массива; по умолчанию Name из getagent), timeout_sec (на команду, 120).
func New(cfg connector.Config) (connector.Connector, error) {
	host := cfg.String("host")
	if host == "" {
		return nil, fmt.Errorf("vnx %s: не задан host", cfg.Code)
	}
	bin := cfg.String("binary")
	if bin == "" {
		bin = "naviseccli"
	}
	base := []string{"-h", host}
	if sec := cfg.String("secfile_path"); sec != "" {
		base = append(base, "-secfilepath", sec)
	} else {
		user := cfg.String("username")
		if user == "" {
			return nil, fmt.Errorf("vnx %s: задайте secfile_path либо username и password_env", cfg.Code)
		}
		pass, err := cfg.Secret("password_env")
		if err != nil {
			return nil, err
		}
		scope := cfg.String("scope")
		if scope == "" {
			scope = "0"
		}
		base = append(base, "-User", user, "-Password", pass, "-Scope", scope)
	}
	return &Client{code: cfg.Code, name: cfg.String("array_name"),
		runner: execRunner{binary: bin, base: base, timeout: time.Duration(cfg.Int("timeout_sec", 120)) * time.Second}}, nil
}

// Test проверяет доступность и учётные данные командой getagent.
func (c *Client) Test(ctx context.Context) error {
	_, err := c.runner.Run(ctx, "getagent")
	return err
}

// block — один блок вывода: первая строка без «:» (заголовок) и пары «ключ: значение».
type block struct {
	header string
	kv     map[string]string
}

func (b block) get(keys ...string) string {
	for _, k := range keys {
		if v := b.kv[strings.ToLower(k)]; v != "" {
			return v
		}
	}
	return ""
}

// parseBlocks разбивает вывод по пустым строкам. Ключи приводятся к нижнему регистру.
func parseBlocks(out []byte) []block {
	var blocks []block
	cur := block{kv: map[string]string{}}
	flush := func() {
		if cur.header != "" || len(cur.kv) > 0 {
			blocks = append(blocks, cur)
		}
		cur = block{kv: map[string]string{}}
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if i := strings.Index(line, ":"); i > 0 && !strings.HasPrefix(strings.TrimSpace(line), "LOGICAL UNIT NUMBER") {
			cur.kv[strings.ToLower(strings.TrimSpace(line[:i]))] = strings.TrimSpace(line[i+1:])
		} else if cur.header == "" {
			cur.header = strings.TrimSpace(line)
		}
	}
	flush()
	return blocks
}

var enclRe = regexp.MustCompile(`(?i)^Bus\s+(\d+)\s+Enclosure\s+(\d+)\s+Disk\s+(\S+)`)
var lunRe = regexp.MustCompile(`(?i)^LOGICAL UNIT NUMBER\s+(\d+)`)

const gib = 1.073741824 // «GBs» VNX — гибибайты

// Extract строит полный снимок массива четырьмя командами; ошибка любой — ошибка снимка.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	run := func(args ...string) ([]block, error) {
		out, err := c.runner.Run(ctx, args...)
		if err != nil {
			return nil, err
		}
		return parseBlocks(out), nil
	}

	agent, err := run("getagent")
	if err != nil {
		return nil, err
	}
	if len(agent) == 0 {
		return nil, storage.Errorf(Kind, c.code, "getagent вернул пустой вывод")
	}
	a := agent[0]
	snap := storage.Snapshot{Array: storage.Array{
		Name: firstNonEmpty(c.name, a.get("name"), c.code), Serial: a.get("serial no"), Family: strings.TrimSpace("Dell EMC " + a.get("model")),
		Microcode: a.get("revision"), Vendor: map[string]any{"vendor": "Dell EMC", "agent_rev": a.get("agent rev")}}}

	disks, err := run("getdisk")
	if err != nil {
		return nil, err
	}
	seenEnc := map[string]bool{}
	for _, d := range disks {
		m := enclRe.FindStringSubmatch(d.header)
		if m == nil || strings.EqualFold(d.get("state"), "Removed") {
			continue // пустой/удалённый слот
		}
		encID := "b" + m[1] + "e" + m[2]
		if !seenEnc[encID] {
			seenEnc[encID] = true
			snap.Enclosures = append(snap.Enclosures, storage.Component{ID: encID, Name: "Bus " + m[1] + " Enclosure " + m[2]})
		}
		mb, _ := strconv.ParseFloat(strings.Fields(d.get("capacity") + " 0")[0], 64)
		snap.Disks = append(snap.Disks, storage.Disk{ID: encID + "d" + m[3], Serial: d.get("serial number"), Model: d.get("product id"), Manufacturer: d.get("vendor id"),
			Firmware: d.get("product revision"), Location: "Disk " + m[3], EnclosureID: encID, Type: diskType(d.get("drive type")),
			CapacityGB: mb * 1.048576 / 1000}) // capacity в МиБ
	}

	pools, err := run("storagepool", "-list", "-all")
	if err != nil {
		return nil, err
	}
	poolID := map[string]string{}
	for _, p := range pools {
		name := p.get("pool name")
		if name == "" {
			continue
		}
		id := firstNonEmpty(p.get("pool id"), name)
		poolID[name] = id
		userGB, _ := strconv.ParseFloat(p.get("user capacity (gbs)"), 64)
		snap.Pools = append(snap.Pools, storage.Pool{ID: id, Name: name, RAID: p.get("raid type"), CapacityTB: userGB * gib / 1000})
	}

	luns, err := run("getlun")
	if err != nil {
		return nil, err
	}
	for _, l := range luns {
		m := lunRe.FindStringSubmatch(l.header)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		gb, _ := strconv.ParseFloat(l.get("user capacity (gbs)"), 64)
		thin := strings.EqualFold(l.get("is thin lun"), "yes")
		v := storage.Volume{ID: m[1], Name: firstNonEmpty(l.get("name"), "LUN "+m[1]), LunID: &n, WWN: storage.NormalizeWWN(l.get("uid")),
			SizeGB: gb * gib, Thin: &thin, Protocol: "block"}
		if id, ok := poolID[l.get("pool name")]; ok {
			v.PoolID = id
		}
		snap.Volumes = append(snap.Volumes, v)
	}

	// Презентация: storagegroup -list — HBA UID (WWNN:WWPN) и пары HLU/ALU (ALU = номер LUN).
	sgOut, err := c.runner.Run(ctx, "storagegroup", "-list")
	if err != nil {
		return nil, err
	}
	snap.Mappings = parseStorageGroups(sgOut)
	snap.MappingsKnown = true

	return storage.Items(storage.Build(snap, c.code)), nil
}

var (
	hbaLineRe = regexp.MustCompile(`^\s*([0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2}){7,15})\s+SP\s+[AB]\b`)
	aluLineRe = regexp.MustCompile(`^\s*(\d+)\s+(\d+)\s*$`)
)

// parseStorageGroups разбирает «storagegroup -list»: группа = имя, секция «HBA/SP Pairs» (HBA UID),
// секция «HLU/ALU Pairs» (номер LUN на массиве — ALU). Секции разделены пустыми строками и заголовками,
// поэтому разбор построчный, по состоянию.
func parseStorageGroups(out []byte) []storage.Mapping {
	var maps []storage.Mapping
	var hbas, alus []string
	section := ""
	flush := func() {
		for _, alu := range alus {
			maps = append(maps, storage.Mapping{VolumeID: alu, WWPNs: append([]string(nil), hbas...)})
		}
		hbas, alus, section = nil, nil, ""
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n") {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "Storage Group Name:"):
			flush()
		case strings.HasPrefix(trim, "HBA/SP Pairs"):
			section = "hba"
		case strings.HasPrefix(trim, "HLU/ALU Pairs"):
			section = "alu"
		case strings.HasPrefix(trim, "Shareable:"):
			section = ""
		case section == "hba":
			if m := hbaLineRe.FindStringSubmatch(line); m != nil {
				hbas = append(hbas, m[1])
			}
		case section == "alu":
			if m := aluLineRe.FindStringSubmatch(line); m != nil {
				alus = append(alus, m[2])
			}
		}
	}
	flush()
	return maps
}

// diskType — «Drive Type» getdisk (SAS, NL SAS, SATA, Flash/SSD, EFD) в тип компонента.
func diskType(t string) string {
	t = strings.ToUpper(t)
	if strings.Contains(t, "FLASH") || strings.Contains(t, "SSD") || strings.Contains(t, "EFD") {
		return "ssd"
	}
	return "disk"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// NewWithRunner — коннектор с подменяемым исполнителем команд (тесты).
func NewWithRunner(code string, r Runner) connector.Connector { return &Client{code: code, runner: r} }
