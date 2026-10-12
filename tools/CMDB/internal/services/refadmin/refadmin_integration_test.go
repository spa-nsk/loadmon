// Интеграционные тесты refadmin (E1.8): реальный PostgreSQL с применёнными миграциями и seed
// nc2026. Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/refadmin/...
package refadmin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

func TestCreateGetUpdateDelete_SimpleTable(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	spec, ok := Spec("environment")
	if !ok {
		t.Fatal("спецификация environment не найдена")
	}
	code := "T" + testutil.Suffix()
	if len(code) > 20 {
		code = code[:20]
	}

	if err := Create(ctx, spec, map[string]string{"environment_code": code, "name": "Тестовое окружение"}, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	row, err := Get(ctx, spec, code)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if row["name"] != "Тестовое окружение" {
		t.Errorf("name = %q, want %q", row["name"], "Тестовое окружение")
	}

	if err := Update(ctx, spec, code, map[string]string{"environment_code": code, "name": "Переименовано"}, 1); err != nil {
		t.Fatalf("Update: %v", err)
	}
	row, err = Get(ctx, spec, code)
	if err != nil {
		t.Fatalf("Get после Update: %v", err)
	}
	if row["name"] != "Переименовано" {
		t.Errorf("после Update name = %q, want %q", row["name"], "Переименовано")
	}

	if err := Delete(ctx, spec, code); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := Get(ctx, spec, code); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("Get после Delete: err = %v, want ErrNotFound", err)
	}
}

func TestCreate_RequiredFieldMissing(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	spec, _ := Spec("environment")
	err := Create(ctx, spec, map[string]string{"environment_code": "", "name": "x"}, 1)
	if !errors.Is(err, svcerr.ErrInvalidInput) {
		t.Fatalf("Create без обязательного поля: err = %v, want ErrInvalidInput", err)
	}
}

func TestCreate_DuplicatePK(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	spec, _ := Spec("environment")
	code := "T" + testutil.Suffix()
	if len(code) > 20 {
		code = code[:20]
	}
	if err := Create(ctx, spec, map[string]string{"environment_code": code, "name": "x"}, 1); err != nil {
		t.Fatalf("первый Create: %v", err)
	}
	defer Delete(ctx, spec, code)

	err := Create(ctx, spec, map[string]string{"environment_code": code, "name": "y"}, 1)
	if !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторный Create: err = %v, want ErrDuplicate", err)
	}
}

// TestNamingProfile_IsDefaultToggle — E1.8: "версионирование профиля (новая версия не меняет
// статус ранее выданных имён)". Новая версия с is_default=true снимает этот флаг у остальных
// версий ТОГО ЖЕ profile_code (BeforeSave, specs.go), но не трогает ref.designation.
func TestNamingProfile_IsDefaultToggle(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	spec, _ := Spec("naming_profile")

	profileCode := "TESTNP" + testutil.Suffix()
	if len(profileCode) > 20 {
		profileCode = profileCode[:20]
	}
	if _, err := db.Exec(`INSERT INTO ref.naming_profile_code (profile_code, name) VALUES ($1, 'тест')`, profileCode); err != nil {
		t.Fatalf("подготовка naming_profile_code: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM ref.naming_profile_code WHERE profile_code = $1`, profileCode) })

	if err := Create(ctx, spec, map[string]string{
		"profile_code": profileCode, "version": "1.0", "effective_from": "2026-01-01", "is_default": "true",
	}, 1); err != nil {
		t.Fatalf("Create v1.0: %v", err)
	}
	if err := Create(ctx, spec, map[string]string{
		"profile_code": profileCode, "version": "1.1", "effective_from": "2026-02-01", "is_default": "true",
	}, 1); err != nil {
		t.Fatalf("Create v1.1: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM ref.naming_profile WHERE profile_code = $1`, profileCode) })

	rows, err := db.Query(`SELECT version, is_default FROM ref.naming_profile WHERE profile_code = $1 ORDER BY version`, profileCode)
	if err != nil {
		t.Fatalf("проверочный запрос: %v", err)
	}
	defer rows.Close()
	defaults := map[string]bool{}
	for rows.Next() {
		var v string
		var d bool
		if err := rows.Scan(&v, &d); err != nil {
			t.Fatal(err)
		}
		defaults[v] = d
	}
	if defaults["1.0"] != false || defaults["1.1"] != true {
		t.Errorf("is_default после второй версии = %+v, want 1.0=false, 1.1=true", defaults)
	}
}

func TestExportImportCSV_RoundTrip(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	spec, _ := Spec("environment")
	code := "T" + testutil.Suffix()
	if len(code) > 20 {
		code = code[:20]
	}
	if err := Create(ctx, spec, map[string]string{"environment_code": code, "name": "исходное"}, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer Delete(ctx, spec, code)

	var buf strings.Builder
	if err := ExportCSV(ctx, spec, &buf); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if !strings.Contains(buf.String(), code) {
		t.Fatalf("экспорт не содержит созданную строку: %s", buf.String())
	}

	// Импорт того же CSV с изменённым именем — апдейт по PK (environment_code).
	csv := "environment_code,name\n" + code + ",из CSV\nNEWENV" + code + ",новое окружение\n"
	res, err := ImportCSV(ctx, spec, strings.NewReader(csv), 1)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	defer Delete(ctx, spec, "NEWENV"+code)
	if res.Updated != 1 || res.Inserted != 1 || len(res.Errors) != 0 {
		t.Fatalf("ImportCSV result = %+v, want 1 updated, 1 inserted, 0 errors", res)
	}

	row, err := Get(ctx, spec, code)
	if err != nil {
		t.Fatalf("Get после импорта: %v", err)
	}
	if row["name"] != "из CSV" {
		t.Errorf("после импорта name = %q, want %q", row["name"], "из CSV")
	}
}
