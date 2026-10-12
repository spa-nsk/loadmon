// Package baselineconfig — эталонные конфигурации (plan.md E7.8, ref.baseline_config, создана как
// заготовка в 0003_ref_naming.sql/E1, достроена в 0024): требуемые значения атрибутов класса и
// обязательное ПО, отчёт отклонений на дату (live/?asof=) или против замороженного среза (E6.2,
// «связь с baseline-срезами»). Само правило хранится в ref.baseline_config — не через changectx
// (как и весь ref.*: это настройка контроля, а не учётные данные, версионирования через hist нет).
package baselineconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
	"cmdb/internal/services/baseline"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/software"
	"cmdb/internal/services/svcerr"
)

// RequiredSoftware — запись ref.software, обязательная для этой конфигурации.
type RequiredSoftware struct {
	SoftwareID int64
	Vendor     string
	Product    string
	Version    string
	Edition    string
}

// Label — человекочитаемая подпись для отчёта отклонений.
func (r RequiredSoftware) Label() string {
	s := r.Vendor + " " + r.Product
	if r.Version != "" {
		s += " " + r.Version
	}
	if r.Edition != "" {
		s += " (" + r.Edition + ")"
	}
	return s
}

// Config — эталонная конфигурация: требования для класса, опционально суженные по
// окружению/компании/площадке («класс/роль» из плана — Role остаётся свободной подписью для
// человека, отдельного справочника ролей и колонки роли у cmdb.ci в модели нет, автоматическое
// сопоставление КЕ по нему было бы гаданием).
type Config struct {
	ID               int64
	Name             string
	Description      string
	CIClass          string
	ClassName        string
	Role             string
	EnvironmentCode  string
	CompanyCode      string
	LocationCode     string
	Attrs            map[string]any // имя колонки таблицы-расширения -> требуемое значение
	RequiredSoftware []RequiredSoftware
	Enabled          bool
}

func fromRow(r repo.Row) Config {
	c := Config{Name: r["name"], Description: r["description"], CIClass: r["ci_class"], ClassName: r["class_name"],
		Role: r["role"], EnvironmentCode: r["environment_code"], CompanyCode: r["company_code"], LocationCode: r["location_code"],
		Enabled: r["enabled"] == "true"}
	fmt.Sscan(r["id"], &c.ID)
	_ = jsonUnmarshalAttrs(r["required_attrs"], &c.Attrs)
	return c
}

func jsonUnmarshalAttrs(raw string, out *map[string]any) error {
	if strings.TrimSpace(raw) == "" {
		*out = map[string]any{}
		return nil
	}
	return json.Unmarshal([]byte(raw), out)
}

const selectCols = `bc.id::text AS id, bc.name, bc.description, bc.ci_class, COALESCE(k.name, bc.ci_class) AS class_name,
	COALESCE(bc.role, '') AS role, COALESCE(bc.environment_code, '') AS environment_code,
	COALESCE(bc.company_code, '') AS company_code, COALESCE(bc.location_code, '') AS location_code,
	bc.required_attrs::text AS required_attrs, bc.enabled`

// List возвращает все конфигурации (включая выключенные — для страницы администрирования), с
// требуемым ПО каждой. onlyEnabled — только включённые (для расчёта отклонений).
func List(ctx context.Context, onlyEnabled bool) ([]Config, error) {
	where := ""
	if onlyEnabled {
		where = "WHERE bc.enabled"
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT `+selectCols+`
		FROM ref.baseline_config bc LEFT JOIN ref.ci_class k ON k.class_code = bc.ci_class
		`+where+` ORDER BY bc.name`)
	if err != nil {
		return nil, fmt.Errorf("baselineconfig: список: %w", err)
	}
	out := make([]Config, len(rows))
	for i, r := range rows {
		out[i] = fromRow(r)
	}
	if err := attachSoftware(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// attachSoftware дозаполняет RequiredSoftware у уже загруженных конфигураций одним запросом на все
// ref.baseline_config_software (таблица небольшая, 20–200 конфигураций — intent §9, фильтр по
// списку id в один запрос через ANY() здесь не даёт заметного выигрыша, а без проверенного в
// проекте способа передать Go-срез в параметр это лишний риск).
func attachSoftware(ctx context.Context, configs []Config) error {
	if len(configs) == 0 {
		return nil
	}
	byID := make(map[int64]*Config, len(configs))
	for i := range configs {
		byID[configs[i].ID] = &configs[i]
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT bcs.baseline_config_id::text AS cfg_id, sw.id::text AS sw_id, sw.vendor, sw.product,
		       COALESCE(sw.version, '') AS version, COALESCE(sw.edition, '') AS edition
		FROM ref.baseline_config_software bcs JOIN ref.software sw ON sw.id = bcs.software_id
		ORDER BY sw.vendor, sw.product`)
	if err != nil {
		return fmt.Errorf("baselineconfig: требуемое ПО: %w", err)
	}
	for _, r := range rows {
		var cfgID, swID int64
		fmt.Sscan(r["cfg_id"], &cfgID)
		fmt.Sscan(r["sw_id"], &swID)
		cfg, ok := byID[cfgID]
		if !ok {
			continue
		}
		cfg.RequiredSoftware = append(cfg.RequiredSoftware, RequiredSoftware{SoftwareID: swID, Vendor: r["vendor"], Product: r["product"], Version: r["version"], Edition: r["edition"]})
	}
	return nil
}

// Get возвращает одну конфигурацию с требуемым ПО.
func Get(ctx context.Context, id int64) (Config, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT `+selectCols+`
		FROM ref.baseline_config bc LEFT JOIN ref.ci_class k ON k.class_code = bc.ci_class
		WHERE bc.id = ?`, id)
	if err != nil {
		return Config{}, fmt.Errorf("baselineconfig: чтение: %w", err)
	}
	if row == nil {
		return Config{}, svcerr.NotFound("baseline_config", id)
	}
	cfg := fromRow(row)
	if err := attachSoftware(ctx, []Config{cfg}); err != nil {
		return Config{}, err
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT sw.id::text AS sw_id, sw.vendor, sw.product, COALESCE(sw.version, '') AS version, COALESCE(sw.edition, '') AS edition
		FROM ref.baseline_config_software bcs JOIN ref.software sw ON sw.id = bcs.software_id
		WHERE bcs.baseline_config_id = ? ORDER BY sw.vendor, sw.product`, id)
	if err != nil {
		return Config{}, fmt.Errorf("baselineconfig: требуемое ПО: %w", err)
	}
	for _, r := range rows {
		var swID int64
		fmt.Sscan(r["sw_id"], &swID)
		cfg.RequiredSoftware = append(cfg.RequiredSoftware, RequiredSoftware{SoftwareID: swID, Vendor: r["vendor"], Product: r["product"], Version: r["version"], Edition: r["edition"]})
	}
	return cfg, nil
}

// Input — данные формы создания/правки.
type Input struct {
	Name             string
	Description      string
	CIClass          string
	Role             string
	EnvironmentCode  string
	CompanyCode      string
	LocationCode     string
	Attrs            map[string]any
	RequiredSoftware []int64 // id записей ref.software
	Enabled          bool
	UserID           int64
}

// validateAttrs проверяет, что все ключи Attrs — реальные колонки таблицы-расширения класса (как
// ci.UpsertExtensionTx для атрибутов КЕ — тот же принцип: имя, которое подставляется в SQL,
// сверяется со списком реальных колонок, а не берётся на веру из формы).
func validateAttrs(ctx context.Context, ciClass string, attrs map[string]any) error {
	if len(attrs) == 0 {
		return nil
	}
	extTable, err := extTableFor(ctx, ciClass)
	if err != nil {
		return err
	}
	if extTable == "" {
		return fmt.Errorf("%w: у класса %q нет таблицы-расширения — атрибуты задавать нечему", svcerr.ErrInvalidInput, ciClass)
	}
	cols, err := ci.ExtensionColumns(ctx, orm.NewOrm(), extTable)
	if err != nil {
		return err
	}
	for k := range attrs {
		if _, ok := cols[k]; !ok {
			return fmt.Errorf("%w: %q не колонка cmdb.%s", svcerr.ErrInvalidInput, k, extTable)
		}
	}
	return nil
}

func saveSoftware(ctx context.Context, tx orm.TxOrmer, id int64, softwareIDs []int64) error {
	if _, err := tx.RawWithCtx(ctx, `DELETE FROM ref.baseline_config_software WHERE baseline_config_id = ?`, id).Exec(); err != nil {
		return fmt.Errorf("baselineconfig: очистка требуемого ПО: %w", err)
	}
	for _, swID := range softwareIDs {
		if _, err := tx.RawWithCtx(ctx, `INSERT INTO ref.baseline_config_software (baseline_config_id, software_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
			id, swID).Exec(); err != nil {
			return fmt.Errorf("baselineconfig: требуемое ПО id=%d: %w", swID, err)
		}
	}
	return nil
}

// Create создаёт конфигурацию вместе со списком требуемого ПО одной транзакцией.
func Create(ctx context.Context, in Input) (int64, error) {
	if strings.TrimSpace(in.Name) == "" {
		return 0, fmt.Errorf("%w: не задано имя конфигурации", svcerr.ErrInvalidInput)
	}
	if err := validateAttrs(ctx, in.CIClass, in.Attrs); err != nil {
		return 0, err
	}
	attrsJSON, err := json.Marshal(in.Attrs)
	if err != nil {
		return 0, err
	}
	var id int64
	err = orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		e := tx.RawWithCtx(ctx, `
			INSERT INTO ref.baseline_config (name, description, ci_class, role, environment_code, company_code, location_code,
				required_attrs, enabled, created_by, updated_by)
			VALUES (?, ?, ?, nullif(?, ''), nullif(?, ''), nullif(?, ''), nullif(?, ''), CAST(? AS jsonb), ?, ?, ?)
			RETURNING id
		`, in.Name, in.Description, in.CIClass, in.Role, in.EnvironmentCode, in.CompanyCode, in.LocationCode,
			string(attrsJSON), in.Enabled, in.UserID, in.UserID).QueryRow(&id)
		if svcerr.ViolatesConstraint(e, "baseline_config_name_uniq") {
			return fmt.Errorf("%w: конфигурация %q", svcerr.ErrDuplicate, in.Name)
		}
		if e != nil {
			return fmt.Errorf("baselineconfig: создание: %w", e)
		}
		return saveSoftware(ctx, tx, id, in.RequiredSoftware)
	})
	return id, err
}

// Update сохраняет правку конфигурации и полностью заменяет список требуемого ПО.
func Update(ctx context.Context, id int64, in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("%w: не задано имя конфигурации", svcerr.ErrInvalidInput)
	}
	if err := validateAttrs(ctx, in.CIClass, in.Attrs); err != nil {
		return err
	}
	attrsJSON, err := json.Marshal(in.Attrs)
	if err != nil {
		return err
	}
	return orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		res, e := tx.RawWithCtx(ctx, `
			UPDATE ref.baseline_config
			SET name = ?, description = ?, ci_class = ?, role = nullif(?, ''), environment_code = nullif(?, ''),
			    company_code = nullif(?, ''), location_code = nullif(?, ''), required_attrs = CAST(? AS jsonb),
			    enabled = ?, updated_by = ?, updated_at = now()
			WHERE id = ?
		`, in.Name, in.Description, in.CIClass, in.Role, in.EnvironmentCode, in.CompanyCode, in.LocationCode,
			string(attrsJSON), in.Enabled, in.UserID, id).Exec()
		if svcerr.ViolatesConstraint(e, "baseline_config_name_uniq") {
			return fmt.Errorf("%w: конфигурация %q", svcerr.ErrDuplicate, in.Name)
		}
		if e != nil {
			return fmt.Errorf("baselineconfig: правка: %w", e)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return svcerr.NotFound("baseline_config", id)
		}
		return saveSoftware(ctx, tx, id, in.RequiredSoftware)
	})
}

// Delete удаляет конфигурацию (требуемое ПО — каскадом).
func Delete(ctx context.Context, id int64) error {
	res, err := orm.NewOrm().RawWithCtx(ctx, `DELETE FROM ref.baseline_config WHERE id = ?`, id).Exec()
	if err != nil {
		return fmt.Errorf("baselineconfig: удаление: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return svcerr.NotFound("baseline_config", id)
	}
	return nil
}

func extTableFor(ctx context.Context, ciClass string) (string, error) {
	row, err := repo.QueryRow(ctx, `SELECT COALESCE(ext_table, '') AS ext_table FROM ref.ci_class WHERE class_code = ?`, ciClass)
	if err != nil {
		return "", fmt.Errorf("baselineconfig: ext_table класса %q: %w", ciClass, err)
	}
	if row == nil {
		return "", nil
	}
	return row["ext_table"], nil
}

// Deviation — одно отклонение действующей (или зафиксированной в срезе) КЕ от эталонной конфигурации.
type Deviation struct {
	ConfigID   int64
	ConfigName string
	CIID       int64
	CIName     string
	Kind       string // "attr" | "software"
	Field      string
	Expected   string
	Actual     string // "" у Kind=software значит «не установлено»
}

// Deviations считает отклонения. baselineID<=0 — по действующим КЕ (учитывает ?asof= в ctx, как и
// весь остальной UI, E6.1); baselineID>0 — по составу замороженного/утверждённого среза (E6.2,
// «связь с baseline-срезами»): проверяются атрибуты, зафиксированные в срезе, а не текущие. У среза
// нет установок ПО (cmdb.ci_software_install не входит в baseline.Entities — установки ПО в срез
// не попадают, E6.2 их не версионирует), поэтому в этом режиме проверяются только атрибуты.
func Deviations(ctx context.Context, baselineID int64) ([]Deviation, error) {
	if baselineID > 0 {
		return deviationsSnapshot(ctx, baselineID)
	}
	return deviationsLive(ctx)
}

func matchingCIs(ctx context.Context, cfg Config) ([]repo.Row, error) {
	from := "cmdb.ci c"
	args := []any{}
	if t, ok := asof.FromContext(ctx); ok {
		from = "cmdb.ci_asof(CAST(CAST(? AS text) AS timestamptz)) c"
		args = append(args, repo.TimeArg(t))
	}
	where := "c.ci_class = ? AND c.lifecycle_stage NOT IN ('Retired', 'Disposed')"
	args = append(args, cfg.CIClass)
	if cfg.EnvironmentCode != "" {
		where += " AND c.environment_code = ?"
		args = append(args, cfg.EnvironmentCode)
	}
	if cfg.CompanyCode != "" {
		where += " AND c.company_code = ?"
		args = append(args, cfg.CompanyCode)
	}
	if cfg.LocationCode != "" {
		where += " AND c.location_code = ?"
		args = append(args, cfg.LocationCode)
	}
	return repo.QueryRows(ctx, `SELECT c.id::text AS id, c.name FROM `+from+` WHERE `+where+` ORDER BY c.name`, args...)
}

// extAttrsForCI — атрибуты расширения одной КЕ, с ?asof=; тот же приём UNION cmdb./hist., что
// internal/services/ci.extAsOf (неэкспортирован — здесь маленькая копия ради одного нового
// вызывающего, не стоит расширять экспортируемую поверхность ci ради этого).
func extAttrsForCI(ctx context.Context, table string, ciID int64) (repo.Row, error) {
	cols, err := ci.ExtensionColumns(ctx, orm.NewOrm(), table)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(cols))
	for c := range cols {
		names = append(names, c)
	}
	sort.Strings(names)
	list := strings.Join(names, ", ")
	if t, ok := asof.FromContext(ctx); ok {
		ts := repo.TimeArg(t)
		return repo.QueryRow(ctx, `
			SELECT `+list+` FROM cmdb.`+table+` WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
			UNION ALL
			SELECT `+list+` FROM hist.`+table+` WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
			LIMIT 1`, ciID, ts, ciID, ts)
	}
	return repo.QueryRow(ctx, `SELECT `+list+` FROM cmdb.`+table+` WHERE ci_id = ?`, ciID)
}

func deviationsLive(ctx context.Context) ([]Deviation, error) {
	configs, err := List(ctx, true)
	if err != nil {
		return nil, err
	}
	var out []Deviation
	for _, cfg := range configs {
		extTable := ""
		if len(cfg.Attrs) > 0 {
			if extTable, err = extTableFor(ctx, cfg.CIClass); err != nil {
				return nil, err
			}
		}
		cis, err := matchingCIs(ctx, cfg)
		if err != nil {
			return nil, err
		}
		for _, row := range cis {
			var ciID int64
			fmt.Sscan(row["id"], &ciID)
			ciName := row["name"]

			if extTable != "" {
				ext, err := extAttrsForCI(ctx, extTable, ciID)
				if err != nil {
					return nil, fmt.Errorf("baselineconfig: атрибуты КЕ id=%d: %w", ciID, err)
				}
				for k, want := range cfg.Attrs {
					got := ""
					if ext != nil {
						got = ext[k]
					}
					if displayValue(want) != got {
						out = append(out, Deviation{ConfigID: cfg.ID, ConfigName: cfg.Name, CIID: ciID, CIName: ciName,
							Kind: "attr", Field: k, Expected: displayValue(want), Actual: got})
					}
				}
			}
			if len(cfg.RequiredSoftware) > 0 {
				installs, err := software.ListForCI(ctx, ciID)
				if err != nil {
					return nil, fmt.Errorf("baselineconfig: ПО КЕ id=%d: %w", ciID, err)
				}
				have := map[int64]bool{}
				for _, in := range installs {
					have[in.SoftwareID] = true
				}
				for _, req := range cfg.RequiredSoftware {
					if !have[req.SoftwareID] {
						out = append(out, Deviation{ConfigID: cfg.ID, ConfigName: cfg.Name, CIID: ciID, CIName: ciName,
							Kind: "software", Field: req.Label(), Expected: "установлено"})
					}
				}
			}
		}
	}
	return out, nil
}

func deviationsSnapshot(ctx context.Context, baselineID int64) ([]Deviation, error) {
	b, err := baseline.Get(ctx, baselineID)
	if err != nil {
		return nil, err
	}
	if b.Status == "draft" {
		return nil, fmt.Errorf("%w: срез — черновик, состав ещё может измениться", svcerr.ErrInvalidState)
	}
	configs, err := List(ctx, true)
	if err != nil {
		return nil, err
	}
	items, err := repo.QueryRows(ctx, `SELECT entity_id::text AS ci_id, row_version::text AS row_version FROM rpt.baseline_item WHERE baseline_id = ? AND entity = 'ci'`, baselineID)
	if err != nil {
		return nil, fmt.Errorf("baselineconfig: состав среза: %w", err)
	}

	var out []Deviation
	for _, cfg := range configs {
		if len(cfg.Attrs) == 0 {
			continue // требуемого ПО в срезе не проверить (см. комментарий Deviations) — без атрибутов конфигурации нечего сравнивать
		}
		extTable, err := extTableFor(ctx, cfg.CIClass)
		if err != nil {
			return nil, err
		}
		if extTable == "" {
			continue
		}
		for _, it := range items {
			var ciID int64
			var ver int
			fmt.Sscan(it["ci_id"], &ciID)
			fmt.Sscan(it["row_version"], &ver)

			data, err := baseline.VersionData(ctx, "ci", ciID, ver)
			if err != nil || data == nil {
				continue
			}
			if displayValue(data["ci_class"]) != cfg.CIClass {
				continue
			}
			if stage := displayValue(data["lifecycle_stage"]); stage == "Retired" || stage == "Disposed" {
				continue
			}
			if cfg.EnvironmentCode != "" && displayValue(data["environment_code"]) != cfg.EnvironmentCode {
				continue
			}
			if cfg.CompanyCode != "" && displayValue(data["company_code"]) != cfg.CompanyCode {
				continue
			}
			if cfg.LocationCode != "" && displayValue(data["location_code"]) != cfg.LocationCode {
				continue
			}
			ciName := displayValue(data["name"])

			extVerRow, err := repo.QueryRow(ctx, `SELECT row_version::text AS v FROM rpt.baseline_item WHERE baseline_id = ? AND entity = ? AND entity_id = ?`,
				baselineID, extTable, ciID)
			if err != nil || extVerRow == nil {
				continue // расширения этой КЕ нет в срезе (области не совпали при заморозке) — сравнивать нечего
			}
			extVer, _ := strconv.Atoi(extVerRow["v"])
			extData, err := baseline.VersionData(ctx, extTable, ciID, extVer)
			if err != nil || extData == nil {
				continue
			}
			for k, want := range cfg.Attrs {
				got := displayValue(extData[k])
				if displayValue(want) != got {
					out = append(out, Deviation{ConfigID: cfg.ID, ConfigName: cfg.Name, CIID: ciID, CIName: ciName,
						Kind: "attr", Field: k, Expected: displayValue(want), Actual: got})
				}
			}
		}
	}
	return out, nil
}

func displayValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
