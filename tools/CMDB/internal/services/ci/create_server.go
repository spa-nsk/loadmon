// Package ci — сервисы работы с конфигурационными единицами: создание с именем по профилю
// (naming + seqalloc), правка с оптимистической блокировкой, переходы жизненного цикла,
// атрибуты классов (таблицы-расширения) и чтение для UI.
package ci

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/seqalloc"
	"cmdb/internal/services/svcerr"
)

// CreateInput — данные для ручного создания КЕ класса, у которого есть код в профиле именования.
type CreateInput struct {
	Class           string // ref.ci_class.class_code, например "server", "vm"
	LocationCode    string // полный код локации с дефисом, например "OVB-DC1" (обязателен: из него строится SITE)
	CompanyCode     string
	EnvironmentCode string
	ModelID         *int64
	SerialNumber    string
	Description     string
	Criticality     string
	// Attrs — атрибуты класса (колонки таблицы-расширения); nil/пусто — создаётся пустая запись расширения.
	Attrs  map[string]any
	Reason string // обязательна для ручного ввода (changectx.SourceManual)
	UserID int64
	Source changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI
}

// CreateResult — результат создания: id КЕ, присвоенное имя, id набора изменений.
type CreateResult struct {
	CIID        int64
	Name        string
	ChangeSetID int64
}

// Create выделяет имя по профилю CMDB-NC-2026 (код класса по Registry, номер из seqalloc) и
// создаёт КЕ вместе с записью в таблице-расширении одной транзакцией ChangeContext.
func Create(ctx context.Context, reg *naming.Registry, in CreateInput) (CreateResult, error) {
	var result CreateResult

	spec := SpecFor(in.Class)
	if spec == nil {
		return result, fmt.Errorf("%w: для класса %q нет формы создания", svcerr.ErrInvalidState, in.Class)
	}
	des, ok := reg.ByClass(in.Class)
	if !ok {
		return result, fmt.Errorf("%w: для класса %q нет активного кода в профиле именования", svcerr.ErrInvalidState, in.Class)
	}
	if in.LocationCode == "" {
		return result, fmt.Errorf("%w: не указана площадка", svcerr.ErrInvalidInput)
	}

	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		// SITE в имени — сжатый код площадки (location_code_c, без дефиса).
		var site string
		err := tx.RawWithCtx(ctx, `SELECT location_code_c FROM ref.location WHERE location_code = ?`, in.LocationCode).QueryRow(&site)
		if err == orm.ErrNoRows {
			return fmt.Errorf("%w: площадка %q не найдена", svcerr.ErrInvalidInput, in.LocationCode)
		}
		if err != nil {
			return fmt.Errorf("ci: чтение площадки: %w", err)
		}

		name, err := seqalloc.AllocateHostName(ctx, tx, reg, site, des.Code)
		if err != nil {
			return fmt.Errorf("ci: выделение имени: %w", err)
		}
		parsed, err := naming.ParseHostName(name)
		if err != nil {
			return fmt.Errorf("ci: разбор выделенного имени %q: %w", name, err)
		}

		var ciID int64
		err = tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.ci
				(ci_class, name, naming_status, naming_profile_code, designation_code, sequence_number,
				 location_code, company_code, environment_code, model_id, serial_number, description, criticality,
				 created_by, updated_by)
			VALUES (?, ?, 'compliant', 'CMDB-NC-2026', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING id
		`, in.Class, name, des.Code, parsed.Seq, in.LocationCode, nullableString(in.CompanyCode),
			nullableString(in.EnvironmentCode), repo.NullPtr(in.ModelID), nullableString(in.SerialNumber),
			nullableString(in.Description), nullableString(in.Criticality), in.UserID, in.UserID,
		).QueryRow(&ciID)
		if err != nil {
			return fmt.Errorf("ci: вставка cmdb.ci: %w", err)
		}

		if len(in.Attrs) > 0 {
			var res ExtensionResult
			ext := ExtensionInput{CIID: ciID, Table: spec.ExtTable, Attrs: in.Attrs, UserID: in.UserID}
			if err := UpsertExtensionTx(ctx, tx, ext, &res); err != nil {
				return err
			}
		} else if _, err := tx.RawWithCtx(ctx, `INSERT INTO cmdb.`+spec.ExtTable+` (ci_id, created_by, updated_by) VALUES (?, ?, ?)`,
			ciID, in.UserID, in.UserID).Exec(); err != nil {
			return fmt.Errorf("ci: вставка cmdb.%s: %w", spec.ExtTable, err)
		}

		result.CIID = ciID
		result.Name = name
		return nil
	})
	result.ChangeSetID = csID
	return result, err
}

// CreateServerInput — данные для ручного создания КЕ класса "физический сервер".
type CreateServerInput struct {
	Site            string // не используется: SITE берётся из площадки; поле оставлено для совместимости тестов
	LocationCode    string // полный код локации с дефисом, например "OVB-DC1"
	CompanyCode     string
	EnvironmentCode string
	ModelID         *int64
	SerialNumber    string
	Reason          string
	UserID          int64
}

// CreateServerResult — результат создания сервера.
type CreateServerResult = CreateResult

// CreateServer — создание физического сервера (частный случай Create).
func CreateServer(ctx context.Context, reg *naming.Registry, in CreateServerInput) (CreateServerResult, error) {
	return Create(ctx, reg, CreateInput{
		Class: "server", LocationCode: in.LocationCode, CompanyCode: in.CompanyCode,
		EnvironmentCode: in.EnvironmentCode, ModelID: in.ModelID, SerialNumber: in.SerialNumber,
		Reason: in.Reason, UserID: in.UserID,
	})
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
