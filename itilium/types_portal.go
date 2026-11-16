package itilium

// Типы HTTP-сервиса веб-портала /hs/web-portal-api (раздел «Пример
// интеграции с внешним порталом» в external_systems). Роль пользователя —
// «Потребитель услуг».

// PortalUID — объект с идентификатором (Category, MembershipServices, CI…).
type PortalUID struct {
	UID string `json:"UID"`
}

// PortalNamed — UID + Name.
type PortalNamed struct {
	UID  string `json:"UID"`
	Name string `json:"Name"`
}

// PortalMembershipServices — состав услуги в запросе/ответе портала.
type PortalMembershipServices struct {
	UID     string       `json:"UID"`
	Name    string       `json:"Name,omitempty"`
	Service *PortalNamed `json:"Service,omitempty"`
}

// PortalNewIncident — тело POST addNewIncident.
type PortalNewIncident struct {
	Topic              string                    `json:"Topic"`
	Data               string                    `json:"Data"` // "d.m.Y G:i:s"
	Description        string                    `json:"Description"`
	MembershipServices *PortalMembershipServices `json:"MembershipServices,omitempty"`
	Category           *PortalUID                `json:"Category,omitempty"`
	ConfigurationItems []PortalUID               `json:"ConfigurationItems,omitempty"`
}

// PortalIncident — краткая карточка обращения портала.
type PortalIncident struct {
	UID                      string                    `json:"UID"`
	Number                   string                    `json:"Number"`
	Data                     string                    `json:"Data"`
	Topic                    string                    `json:"Topic"`
	Description              string                    `json:"Description"`
	Status                   *PortalStatus             `json:"Status"`
	TheRegulatoryClosureDate string                    `json:"TheRegulatoryClosureDate"`
	TheActualClosingDate     string                    `json:"TheActualClosingDate"`
	MembershipServices       *PortalMembershipServices `json:"MembershipServices"`
	Category                 *PortalNamed              `json:"Category"`
	ThereAreFiles            FlexBool                  `json:"ThereAreFiles"`
	Rating                   FlexString                `json:"Rating"`
	CommentToTheRating       string                    `json:"CommentToTheRating"`
	RequiresConfirmation     FlexBool                  `json:"RequiresConfirmation"`
	RequiresEvaluation       FlexBool                  `json:"RequiresEvaluation"`
	RequiresCommentToRating  FlexBool                  `json:"RequiresCommentToRating"`
}

// PortalStatus — состояние обращения портала.
type PortalStatus struct {
	UID      string   `json:"UID"`
	Name     string   `json:"Name"`
	IsClosed FlexBool `json:"IsClosed"`
}

// PortalIncidentDetail — детальная карточка (getDetailInfoIncindent/{id}).
type PortalIncidentDetail struct {
	PortalIncident
	TheHistoryOfCommunication []PortalMessage           `json:"TheHistoryOfCommunication"`
	ConfigurationItem         []PortalConfigurationItem `json:"Configuration item"`
	FilesDefinitions          []PortalFileDef           `json:"FilesDefinitions"`
}

// PortalMessage — сообщение в истории общения портала.
type PortalMessage struct {
	Text  string `json:"Text"`
	Data  string `json:"Data"`
	Autor string `json:"Autor"`
}

// PortalConfigurationItem — КЕ пользователя/обращения на портале.
type PortalConfigurationItem struct {
	UID        string        `json:"UID"`
	Name       string        `json:"Name"`
	Cod        string        `json:"Cod"`
	Classifier *PortalNamed  `json:"Classifier"`
	Services   []PortalNamed `json:"Services"`
}

// PortalFileDef — описание файла на портале.
type PortalFileDef struct {
	UID  string `json:"UID"`
	Name string `json:"Name"`
}

// PortalConfirmIncident — подтверждение выполнения.
type PortalConfirmIncident struct {
	UID     string     `json:"UID"`
	Comment string     `json:"Comment"`
	Rating  FlexString `json:"Rating"`
}

// PortalRejectIncident — отклонение выполнения (тот же путь confirmIncident
// в документации; тело без Rating).
type PortalRejectIncident struct {
	UID     string `json:"UID"`
	Comment string `json:"Comment"`
}

// PortalAddFile — тело addFileToIncident (Data — base64).
type PortalAddFile struct {
	UID      string `json:"UID"`
	Data     string `json:"Data"`
	NameFile string `json:"NameFile"`
}

// PortalAddComment — тело addCommentToIncident.
type PortalAddComment struct {
	UID     string `json:"UID"`
	Comment string `json:"Comment"`
}
