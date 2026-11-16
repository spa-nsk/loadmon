package itilium

import (
	"context"
	"net/http"
	"net/url"
)

// PortalClient — методы HTTP-сервиса веб-портала (`/hs/web-portal-api/...`).
// Пользователь Basic Auth должен иметь роль «Потребитель услуг».
type PortalClient struct {
	c *Client
}

func (p *PortalClient) get(ctx context.Context, path string, out any) error {
	return p.c.doJSON(ctx, http.MethodGet, p.c.resolve(p.c.porPre, path), nil, out)
}

func (p *PortalClient) post(ctx context.Context, path string, req, out any) error {
	return p.c.doJSON(ctx, http.MethodPost, p.c.resolve(p.c.porPre, path), req, out)
}

// Authenticate — GET authenticate (проверка учётных данных).
func (p *PortalClient) Authenticate(ctx context.Context) error {
	return p.get(ctx, "authenticate", nil)
}

// AddNewIncident — POST addNewIncident.
func (p *PortalClient) AddNewIncident(ctx context.Context, in PortalNewIncident) error {
	return p.post(ctx, "addNewIncident", in, nil)
}

// GetCI — GET getCI (конфигурационные единицы пользователя).
func (p *PortalClient) GetCI(ctx context.Context) ([]PortalConfigurationItem, error) {
	var out []PortalConfigurationItem
	if err := p.get(ctx, "getCI", &out); err != nil {
		return nil, err
	}
	return nonNil(out), nil
}

// GetListServices — GET getListServices (составы услуг пользователя).
func (p *PortalClient) GetListServices(ctx context.Context) ([]PortalMembershipServices, error) {
	var out []PortalMembershipServices
	if err := p.get(ctx, "getListServices", &out); err != nil {
		return nil, err
	}
	return nonNil(out), nil
}

// GetListCategories — GET getListCategories.
func (p *PortalClient) GetListCategories(ctx context.Context) ([]PortalNamed, error) {
	var out []PortalNamed
	if err := p.get(ctx, "getListCategories", &out); err != nil {
		return nil, err
	}
	return nonNil(out), nil
}

// ConfirmIncident — POST confirmIncident (подтверждение с оценкой).
func (p *PortalClient) ConfirmIncident(ctx context.Context, in PortalConfirmIncident) error {
	return p.post(ctx, "confirmIncident", in, nil)
}

// RejectIncident — POST confirmIncident без Rating (отклонение по документации).
func (p *PortalClient) RejectIncident(ctx context.Context, in PortalRejectIncident) error {
	return p.post(ctx, "confirmIncident", in, nil)
}

// AddFileToIncident — POST addFileToIncident.
func (p *PortalClient) AddFileToIncident(ctx context.Context, in PortalAddFile) error {
	return p.post(ctx, "addFileToIncident", in, nil)
}

// AddCommentToIncident — POST addCommentToIncident.
func (p *PortalClient) AddCommentToIncident(ctx context.Context, in PortalAddComment) error {
	return p.post(ctx, "addCommentToIncident", in, nil)
}

// GetListIncidents — GET getListIncidents.
func (p *PortalClient) GetListIncidents(ctx context.Context) ([]PortalIncident, error) {
	return p.listIncidents(ctx, "getListIncidents")
}

// GetListIncidentsOpen — GET getListIncidentsOpen.
func (p *PortalClient) GetListIncidentsOpen(ctx context.Context) ([]PortalIncident, error) {
	return p.listIncidents(ctx, "getListIncidentsOpen")
}

// GetListIncidentsClosed — GET getListIncidentsClosed.
func (p *PortalClient) GetListIncidentsClosed(ctx context.Context) ([]PortalIncident, error) {
	return p.listIncidents(ctx, "getListIncidentsClosed")
}

func (p *PortalClient) listIncidents(ctx context.Context, path string) ([]PortalIncident, error) {
	var out []PortalIncident
	if err := p.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return nonNil(out), nil
}

// GetDetailInfoIncident — GET getDetailInfoIncindent/{idDoc}
// (опечатка Incindent — как в официальной документации).
func (p *PortalClient) GetDetailInfoIncident(ctx context.Context, idDoc string) (*PortalIncidentDetail, error) {
	var out PortalIncidentDetail
	path := "getDetailInfoIncindent/" + url.PathEscape(idDoc)
	if err := p.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetFileData — GET getFileData/{idFile}.
func (p *PortalClient) GetFileData(ctx context.Context, idFile string) (*FileData, error) {
	var out FileData
	path := "getFileData/" + url.PathEscape(idFile)
	if err := p.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
