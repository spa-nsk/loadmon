package itilium

import (
	"context"
	"net/http"
)

// ExternalClient — методы HTTP-сервиса внешнего API обращений
// (`/hs/externalapi/...`, POST + JSON).
type ExternalClient struct {
	c *Client
}

func (e *ExternalClient) post(ctx context.Context, method string, req, out any) error {
	return e.c.doJSON(ctx, http.MethodPost, e.c.resolve(e.c.extPre, method), req, out)
}

// GetIncidentsList — POST getIncidentsList. startFrom/count обязательны
// (передавайте как строки, как в примерах документации).
func (e *ExternalClient) GetIncidentsList(ctx context.Context, filter IncidentListFilter) ([]IncidentBrief, error) {
	var resp IncidentListResponse
	if err := e.post(ctx, "getIncidentsList", filter, &resp); err != nil {
		return nil, err
	}
	if resp.Incidents == nil {
		return []IncidentBrief{}, nil
	}
	return resp.Incidents, nil
}

// ChangeIncidentState — POST changeIncidentState.
func (e *ExternalClient) ChangeIncidentState(ctx context.Context, docUUID, newStateUUID string) error {
	return e.post(ctx, "changeIncidentState", ChangeIncidentStateRequest{
		DocUUID: docUUID, NewStateUUID: newStateUUID,
	}, nil)
}

// GetDetailIncidentInfo — POST extGetDetailIncidentInfo.
func (e *ExternalClient) GetDetailIncidentInfo(ctx context.Context, uuid string) (*IncidentDetail, error) {
	var out IncidentDetail
	if err := e.post(ctx, "extGetDetailIncidentInfo", IncidentDetailRequest{UUID: uuid}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetFileData — POST extGetFileData (Data — base64).
func (e *ExternalClient) GetFileData(ctx context.Context, idFile string) (*FileData, error) {
	var out FileData
	if err := e.post(ctx, "extGetFileData", FileDataRequest{IDFile: idFile}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- справочники для отборов ---

func (e *ExternalClient) GetWorkgroups(ctx context.Context) ([]Workgroup, error) {
	var resp struct {
		Workgroups []Workgroup `json:"Workgroups"`
	}
	if err := e.post(ctx, "getWorkgroups", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Workgroups), nil
}

// GetEmployers — POST getEmployers. empWorkgroupUUID — опциональный отбор.
func (e *ExternalClient) GetEmployers(ctx context.Context, empWorkgroupUUID string) ([]Employer, error) {
	body := map[string]any{}
	if empWorkgroupUUID != "" {
		body["empWorkgroupUuid"] = empWorkgroupUUID
	}
	var resp struct {
		Employers []Employer `json:"Employers"`
	}
	if err := e.post(ctx, "getEmployers", body, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Employers), nil
}

func (e *ExternalClient) GetClients(ctx context.Context) ([]ClientRef, error) {
	var resp struct {
		Clients []ClientRef `json:"Clients"`
	}
	if err := e.post(ctx, "getClients", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Clients), nil
}

// GetInitiators — POST getInitiators. initClientUUID — опциональный отбор.
func (e *ExternalClient) GetInitiators(ctx context.Context, initClientUUID string) ([]Initiator, error) {
	body := map[string]any{}
	if initClientUUID != "" {
		body["initClientUuid"] = initClientUUID
	}
	var resp struct {
		Initiators []Initiator `json:"Initiators"`
	}
	if err := e.post(ctx, "getInitiators", body, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Initiators), nil
}

func (e *ExternalClient) GetCategories(ctx context.Context) ([]Category, error) {
	var resp struct {
		Categories []Category `json:"Categories"`
	}
	if err := e.post(ctx, "getCategories", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Categories), nil
}

func (e *ExternalClient) GetPriorities(ctx context.Context) ([]Priority, error) {
	var resp struct {
		Priorities []Priority `json:"Priorities"`
	}
	if err := e.post(ctx, "getPriorities", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Priorities), nil
}

func (e *ExternalClient) GetSources(ctx context.Context) ([]Source, error) {
	var resp struct {
		Sources []Source `json:"Sources"`
	}
	if err := e.post(ctx, "getSources", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Sources), nil
}

// GetServices — POST getServices. servClientUUID — опциональный отбор по SLA.
func (e *ExternalClient) GetServices(ctx context.Context, servClientUUID string) ([]Service, error) {
	body := map[string]any{}
	if servClientUUID != "" {
		body["servClientUuid"] = servClientUUID
	}
	var resp struct {
		Services []Service `json:"Services"`
	}
	if err := e.post(ctx, "getServices", body, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Services), nil
}

// GetServiceComponents — POST getServiceComponents.
func (e *ExternalClient) GetServiceComponents(ctx context.Context, clientUUID, serviceUUID string) ([]ServiceComponent, error) {
	body := map[string]any{}
	if clientUUID != "" {
		body["servCompClientUuid"] = clientUUID
	}
	if serviceUUID != "" {
		body["servCompServiceUuid"] = serviceUUID
	}
	var resp struct {
		ServiceComponents []ServiceComponent `json:"ServiceComponents"`
	}
	if err := e.post(ctx, "getServiceComponents", body, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.ServiceComponents), nil
}

func (e *ExternalClient) GetTypes(ctx context.Context) ([]IncidentType, error) {
	var resp struct {
		Types []IncidentType `json:"Types"`
	}
	if err := e.post(ctx, "getTypes", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.Types), nil
}

func (e *ExternalClient) GetClosingCodes(ctx context.Context) ([]ClosingCode, error) {
	var resp struct {
		ClosingCodes []ClosingCode `json:"ClosingCodes"`
	}
	if err := e.post(ctx, "getClosingCodes", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.ClosingCodes), nil
}

func (e *ExternalClient) GetStates(ctx context.Context) ([]State, error) {
	var resp struct {
		States []State `json:"States"`
	}
	if err := e.post(ctx, "getStates", map[string]any{}, &resp); err != nil {
		return nil, err
	}
	return nonNil(resp.States), nil
}

func nonNil[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}
