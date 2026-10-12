package ui

import "cmdb/internal/repo"

// classOf возвращает класс КЕ-владельца объекта, найденного запросом (одна колонка ci_class). Пустая
// строка — объект не найден: правку отклонит сам сервис («не найдено»), проверка области не нужна.
func (c *CIController) classOf(query string, id int64) string {
	row, err := repo.QueryRow(c.ctx(), query, id)
	if err != nil || row == nil {
		return ""
	}
	return row["ci_class"]
}

// requireCIWrite проверяет область роли class_owner для КЕ по id (класс берётся из БД, а не из формы:
// поле «ci» в формах задаёт клиент и подделывается).
func (c *CIController) requireCIWrite(ciID int64) {
	if cl := c.classOf(`SELECT ci_class FROM cmdb.ci WHERE id = ?`, ciID); cl != "" {
		c.requireWriteClass(cl)
	}
}

func (c *CIController) requireComponentWrite(componentID int64) {
	if cl := c.classOf(`SELECT k.ci_class FROM cmdb.component p JOIN cmdb.ci k ON k.id = p.ci_id WHERE p.id = ?`, componentID); cl != "" {
		c.requireWriteClass(cl)
	}
}

func (c *CIController) requireIPWrite(ipID int64) {
	if cl := c.classOf(`SELECT k.ci_class FROM cmdb.ip_address p JOIN cmdb.ci k ON k.id = p.ci_id WHERE p.id = ?`, ipID); cl != "" {
		c.requireWriteClass(cl)
	}
}

func (c *CIController) requireRelationWrite(relID int64) {
	if cl := c.classOf(`SELECT k.ci_class FROM cmdb.ci_rel r JOIN cmdb.ci k ON k.id = r.src_ci_id WHERE r.id = ?`, relID); cl != "" {
		c.requireWriteClass(cl)
	}
}

func (c *CIController) requireBulkWrite(ids []int64) {
	seen := map[string]bool{}
	for _, id := range ids {
		if cl := c.classOf(`SELECT ci_class FROM cmdb.ci WHERE id = ?`, id); cl != "" && !seen[cl] {
			seen[cl] = true
			c.requireWriteClass(cl)
		}
	}
}
