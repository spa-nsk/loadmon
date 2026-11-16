package itilium

import "strings"

// Типы внешнего HTTP-сервиса /hs/externalapi (документация
// instruktsiyapo-rabote-s-vneshnim-servisom). Имена JSON-ключей — как в
// примерах ответа 1С (camelCase / PascalCase смешанный стиль оригинала).

// IncidentListFilter — тело POST getIncidentsList.
// startFrom/count обязательны; andOr обязателен при любом отборе.
type IncidentListFilter struct {
	StartFrom FlexString `json:"startFrom"`
	Count     FlexString `json:"count"`
	AndOr     string     `json:"andOr,omitempty"` // "and" | "or"

	DateStart string `json:"dateStart,omitempty"` // yyyymmdd
	DateEnd   string `json:"dateEnd,omitempty"`
	Date      string `json:"date,omitempty"`

	DeadlineDateStart string `json:"deadlineDateStart,omitempty"`
	DeadlineDateEnd   string `json:"deadlineDateEnd,omitempty"`
	DeadlineDate      string `json:"deadlineDate,omitempty"`

	ResponsibleUUID      string `json:"responsibleUuid,omitempty"`
	WorkgroupUUID        string `json:"workgroupUuid,omitempty"`
	ClientUUID           string `json:"clientUuid,omitempty"`
	InitiatorUUID        string `json:"initiatorUuid,omitempty"`
	StateUUID            string `json:"stateUuid,omitempty"`
	CategoryUUID         string `json:"categoryUuid,omitempty"`
	PriorityUUID         string `json:"priorityUuid,omitempty"`
	SourceUUID           string `json:"sourceUuid,omitempty"`
	ServiceUUID          string `json:"serviceUuid,omitempty"`
	ServiceComponentUUID string `json:"serviceComponentUuid,omitempty"`
	TypeUUID             string `json:"typeUuid,omitempty"`
	OperatorUUID         string `json:"operatorUuid,omitempty"`
	CuratorUUID          string `json:"curatorUuid,omitempty"`
	ClosingCodeUUID      string `json:"closingCodeUuid,omitempty"`

	Mass   *FlexBool `json:"mass,omitempty"`
	Number string    `json:"number,omitempty"`

	Topic       string `json:"topic,omitempty"`
	Description string `json:"description,omitempty"`
	Solution    string `json:"solution,omitempty"`
}

// IncidentBrief — элемент списка getIncidentsList.
type IncidentBrief struct {
	LinkUUID        string           `json:"linkUuid"`
	Date            string           `json:"date"`
	Number          string           `json:"number"`
	State           string           `json:"state"`
	StateUUID       string           `json:"stateUuid"`
	Topic           string           `json:"topic"`
	Description     string           `json:"description"`
	AvailableStates []AvailableState `json:"availableStates"`
}

// AvailableState — допустимый переход состояния в кратком списке.
type AvailableState struct {
	State     string `json:"state"`
	StateUUID string `json:"stateUuid"`
}

// IncidentListResponse — обёртка ответа getIncidentsList.
type IncidentListResponse struct {
	Incidents []IncidentBrief `json:"Incidents"`
}

// ChangeIncidentStateRequest — тело changeIncidentState.
type ChangeIncidentStateRequest struct {
	DocUUID      string `json:"docUuid"`
	NewStateUUID string `json:"newStateUuid"`
}

// IncidentDetailRequest — тело extGetDetailIncidentInfo.
type IncidentDetailRequest struct {
	UUID string `json:"uuid"`
}

// FileDataRequest — тело extGetFileData.
type FileDataRequest struct {
	IDFile string `json:"idFile"`
}

// FileData — ответ extGetFileData (Data — base64).
type FileData struct {
	Name string `json:"Name"`
	Data string `json:"Data"`
}

// NamedRef — типовая тройка имя/uuid/код справочника.
type NamedRef struct {
	Name string     `json:"-"`
	UUID string     `json:"-"`
	Code FlexString `json:"-"`
}

// Workgroup / Employer / … — конкретные поля ответов справочников (PascalCase).

type Workgroup struct {
	Workgroup     string     `json:"Workgroup"`
	WorkgroupUUID string     `json:"WorkgroupUuid"`
	WorkgroupCode FlexString `json:"WorkgroupCode"`
}

type Employer struct {
	Employer     string     `json:"Employer"`
	EmployerUUID string     `json:"EmployerUuid"`
	EmployerCode FlexString `json:"EmployerCode"`
}

type ClientRef struct {
	Client     string     `json:"Client"`
	ClientUUID string     `json:"ClientUuid"`
	ClientCode FlexString `json:"ClientCode"`
}

type Initiator struct {
	Initiator     string     `json:"Initiator"`
	InitiatorUUID string     `json:"InitiatorUuid"`
	InitiatorCode FlexString `json:"InitiatorCode"`
}

type Category struct {
	Category     string     `json:"Category"`
	CategoryUUID string     `json:"CategoryUuid"`
	CategoryCode FlexString `json:"CategoryCode"`
}

type Priority struct {
	Priority     string     `json:"Priority"`
	PriorityUUID string     `json:"PriorityUuid"`
	PriorityCode FlexString `json:"PriorityCode"`
}

type Source struct {
	Source     string     `json:"Source"`
	SourceUUID string     `json:"SourceUuid"`
	SourceCode FlexString `json:"SourceCode"`
}

type Service struct {
	Service     string     `json:"Service"`
	ServiceUUID string     `json:"ServiceUuid"`
	ServiceCode FlexString `json:"ServiceCode"`
}

type ServiceComponent struct {
	ServiceComponent     string     `json:"ServiceComponent"`
	ServiceComponentUUID string     `json:"ServiceComponentUuid"`
	ServiceComponentCode FlexString `json:"ServiceComponentCode"`
}

type IncidentType struct {
	Type     string     `json:"Type"`
	TypeUUID string     `json:"TypeUuid"`
	TypeCode FlexString `json:"TypeCode"`
}

type ClosingCode struct {
	ClosingCode     string     `json:"ClosingCode"`
	ClosingCodeUUID string     `json:"ClosingCodeUuid"`
	ClosingCodeCode FlexString `json:"ClosingCodeCode"`
}

type State struct {
	State     string     `json:"State"`
	StateUUID string     `json:"StateUuid"`
	StateCode FlexString `json:"StateCode"`
}

// IncidentDetail — ответ extGetDetailIncidentInfo (основные поля + вложенные
// коллекции). Булевы/числовые поля в примерах часто строки — Flex*.
type IncidentDetail struct {
	UUID                          string     `json:"uuid"`
	Number                        string     `json:"number"`
	Date                          string     `json:"date"`
	Description                   string     `json:"description"`
	FormattedDescription          string     `json:"formattedDescription"`
	Solution                      string     `json:"solution"`
	FormattedSolution             string     `json:"formattedSolution"`
	CommentaryOnTheNotice         string     `json:"commentaryOnTheNotice"`
	FormattedCommentaryOnNotice   string     `json:"formattedCommentaryOnTheNotice"`
	Topic                         string     `json:"topic"`
	Responsible                   string     `json:"responsible"`
	ResponsibleUUID               string     `json:"responsibleUuid"`
	Workgroup                     string     `json:"workgroup"`
	WorkgroupUUID                 string     `json:"workgroupUuid"`
	State                         string     `json:"state"`
	StateUUID                     string     `json:"stateUuid"`
	Priority                      string     `json:"priority"`
	PriorityUUID                  string     `json:"priorityUuid"`
	Client                        string     `json:"client"`
	ClientUUID                    string     `json:"clientUuid"`
	Initiator                     string     `json:"initiator"`
	InitiatorUUID                 string     `json:"initiatorUuid"`
	Service                       string     `json:"service"`
	ServiceUUID                   string     `json:"serviceUuid"`
	ServiceComponent              string     `json:"serviceComponent"`
	ServiceComponentUUID          string     `json:"serviceComponentUuid"`
	Mass                          FlexBool   `json:"mass"`
	Operator                      string     `json:"operator"`
	OperatorUUID                  string     `json:"operatorUuid"`
	Curator                       string     `json:"curator"`
	CuratorUUID                   string     `json:"curatorUuid"`
	Workplace                     string     `json:"workplace"`
	WorkplaceUUID                 string     `json:"workplaceUuid"`
	CommentaryOnTheAssessment     string     `json:"commentaryOnTheAssessment"`
	Grade                         FlexString `json:"grade"`
	LastModifiedDate              string     `json:"lastModifiedDate"`
	Type                          string     `json:"type"`
	TypeUUID                      string     `json:"typeUuid"`
	Deadline                      string     `json:"deadline"`
	Confirmed                     FlexString `json:"confirmed"`
	Rejected                      FlexString `json:"rejected"`
	Category                      string     `json:"category"`
	CategoryUUID                  string     `json:"categoryUuid"`
	DocumentBase                  string     `json:"documentBase"`
	DocumentBaseUUID              string     `json:"documentBaseUuid"`
	Timezone                      string     `json:"timezone"`
	TimezoneUUID                  string     `json:"timezoneUuid"`
	Source                        string     `json:"source"`
	SourceUUID                    string     `json:"sourceUuid"`
	ClosingCode                   string     `json:"closingCode"`
	ClosingCodeUUID               string     `json:"closingCodeUuid"`
	RegistrationStartDate         string     `json:"registrationStartDate"`
	FirstLineSolution             FlexBool   `json:"firstLineSolution"`
	MainDocument                  string     `json:"mainDocument"`
	MainDocumentUUID              string     `json:"mainDocumentUuid"`
	Organization                  string     `json:"organization"`
	OrganizationUUID              string     `json:"organizationUuid"`
	CounterpartyAgreement         string     `json:"counterpartyAgreement"`
	CounterpartyAgreementUUID     string     `json:"counterpartyAgreementUuid"`
	ExternalTreaty                string     `json:"externalTreaty"`
	ExternalTreatyUUID            string     `json:"externalTreatyUuid"`
	TransferredToExternalOrg      FlexBool   `json:"transferredToExternalOrganization"`
	ExternalID                    string     `json:"externalID"`
	ExternalDeadline              string     `json:"externalDeadline"`
	ProcessingScript              string     `json:"processingScript"`
	ProcessingScriptUUID          string     `json:"processingScriptUuid"`
	ServiceUnavailable            FlexBool   `json:"serviceUnavailable"`
	CommentOnTheDeadlineChange    string     `json:"commentOnTheDeadlineChange"`
	BasedOnMass                   FlexBool   `json:"basedOnMass"`
	IssueRegistrationRequired     FlexBool   `json:"issueRegistrationRequired"`
	TemplateApprovalCommittee     string     `json:"templateApprovalCommittee"`
	TemplateApprovalCommitteeUUID string     `json:"templateApprovalCommitteeUuid"`
	UseDynamicWorkgroup           FlexBool   `json:"useDynamicWorkgroup"`
	ActualClosingDate             string     `json:"actualClosingDate"`

	TheHistoryOfCommunication []CommunicationMessage `json:"TheHistoryOfCommunication"`
	ConfigurationItems        []ConfigurationItem    `json:"ConfigurationItems"`
	FilesDefinitions          []FileDefinition       `json:"FilesDefinitions"`
	ThereAreFiles             FlexBool               `json:"ThereAreFiles"`
	AdditionalFields          *AdditionalFields      `json:"AdditionalFields"`
	AvailableStates           []DetailState          `json:"AvailaibleStates"` // опечатка в API как в документации
}

// CommunicationMessage — элемент истории общения.
type CommunicationMessage struct {
	Text        string           `json:"Text"`
	HTMLText    string           `json:"HTMLText"`
	Data        string           `json:"Data"`
	Autor       string           `json:"Autor"` // опечатка в API
	IsMyMessage FlexBool         `json:"IsMyMessage"`
	Files       []FileDefinition `json:"Files"`
}

// ConfigurationItem — КЕ, прикреплённая к обращению.
type ConfigurationItem struct {
	Metadata   string             `json:"Metadata"`
	Name       string             `json:"Name"`
	UID        string             `json:"UID"`
	Cod        string             `json:"Cod"`
	Classifier *Classifier        `json:"Classifier"`
	Services   []CIServiceWrapper `json:"Services"`
}

// Classifier — классификатор КЕ.
type Classifier struct {
	Name string `json:"Name"`
	UID  string `json:"UID"`
}

// CIServiceWrapper — обёртка Services[].Service из примера ответа.
type CIServiceWrapper struct {
	Service *NamedService `json:"Service"`
}

// NamedService — услуга, связанная с КЕ.
type NamedService struct {
	Name string `json:"Name"`
	UID  string `json:"UID"`
}

// FileDefinition — описание файла (UUID/Name; в Messages поле может быть UUID).
type FileDefinition struct {
	UID  string `json:"UID"`
	UUID string `json:"UUID"`
	Name string `json:"Name"`
}

// FileID возвращает идентификатор файла для extGetFileData.
func (f FileDefinition) FileID() string {
	if f.UID != "" {
		return strings.TrimSpace(f.UID)
	}
	return strings.TrimSpace(f.UUID)
}

// DetailState — доступное состояние в деталях (Name/UID).
type DetailState struct {
	Name string `json:"Name"`
	UID  string `json:"UID"`
}

// AdditionalFields — блок доп. полей обращения.
type AdditionalFields struct {
	Fields       []AdditionalField `json:"Fields"`
	Availability FlexBool          `json:"Availability"`
}

// AdditionalField — одно доп. поле.
type AdditionalField struct {
	GUID      string           `json:"GUID"`
	Name      string           `json:"Name"`
	Value     FlexString       `json:"Value"`
	ValueType string           `json:"ValueType"`
	Required  FlexBool         `json:"Required"`
	Comment   string           `json:"Comment"`
	Types     []AdditionalType `json:"Types"`
}

// AdditionalType — допустимый тип значения доп. поля.
type AdditionalType struct {
	TypeName   string `json:"TypeName"`
	ObjectType string `json:"ObjectType"`
	ObjectName string `json:"ObjectName"`
}
