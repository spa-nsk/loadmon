// Package seqalloc выделяет очередной свободный порядковый номер для имени КЕ и строит из него
// готовое имя по правилам пакета naming (intent §5.3, plan.md E1.6).
package seqalloc

import (
	"context"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/naming"
)

const defaultProfileCode = "CMDB-NC-2026"

// AllocateHostName выделяет следующий номер для (site, designationCode) в активном профиле
// CMDB-NC-2026 и возвращает готовое имя хоста (naming.BuildHostName). Вызывается внутри
// транзакции ChangeContext (E2.7), поэтому принимает tx, а не открывает свою транзакцию.
func AllocateHostName(ctx context.Context, tx orm.TxOrmer, reg *naming.Registry, site, designationCode string) (string, error) {
	seq, err := repo.AllocateSeq(ctx, tx, defaultProfileCode, designationCode, site)
	if err != nil {
		return "", err
	}
	return naming.BuildHostName(reg, site, designationCode, seq)
}
