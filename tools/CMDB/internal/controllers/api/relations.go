package api

import (
	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/relation"
)

// RelationsController — /api/v1/relations: операции над связью по её собственному id (закрытие).
// Создание и список связей — на КЕ-источнике, см. CIController.Relations/AddRelation.
type RelationsController struct {
	BaseController
}

// classOfRelationSrc — класс КЕ-источника связи (для проверки области роли class_owner, как
// internal/controllers/ui.requireRelationWrite); "" — связь не найдена, отклонит сам сервис.
func (c *RelationsController) classOfRelationSrc(relID int64) string {
	row, err := repo.QueryRow(c.ctx(), `SELECT k.ci_class FROM cmdb.ci_rel r JOIN cmdb.ci k ON k.id = r.src_ci_id WHERE r.id = ?`, relID)
	if err != nil || row == nil {
		return ""
	}
	return row["ci_class"]
}

// closeRequest — тело POST /api/v1/relations/{id}/close.
type closeRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
}

// Close — POST /api/v1/relations/{id}/close: завершает действие связи (valid_period).
func (c *RelationsController) Close() {
	id := c.id()
	c.requireCIWrite(c.classOfRelationSrc(id))

	var req closeRequest
	c.decodeJSON(&req)
	csID, err := relation.Close(c.ctx(), relation.CloseInput{
		RelID: id, ExpectedVersion: req.ExpectedVersion, Reason: req.Reason, UserID: c.userID(), Source: changectx.SourceAPI,
	})
	c.must(err)
	c.writeJSON(200, map[string]any{"change_set_id": csID})
}
