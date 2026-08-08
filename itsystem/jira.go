package itsystem

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"loadmon/models"
	"net/http"
	"sync"
	"time"
)

var it_system_spr map[string]*models.ItSystemRecord
var it_system []models.ItSystemRecord

var itSystemMutex sync.RWMutex

type Avatar struct {
	URL16    string `json:"url16"`
	URL48    string `json:"url48"`
	URL72    string `json:"url72"`
	URL144   string `json:"url144"`
	URL288   string `json:"url288"`
	ObjectID int    `json:"objectId"`
}

type Icon struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	URL16 string `json:"url16"`
	URL48 string `json:"url48"`
}

type ObjectType struct {
	ID                        int       `json:"id"`
	Name                      string    `json:"name"`
	Type                      int       `json:"type"`
	Description               string    `json:"description"`
	Icon                      Icon      `json:"icon"`
	Position                  int       `json:"position"`
	Created                   time.Time `json:"created"`
	Updated                   time.Time `json:"updated"`
	ObjectCount               int       `json:"objectCount"`
	ParentObjectTypeID        int       `json:"parentObjectTypeId"`
	ObjectSchemaID            int       `json:"objectSchemaId"`
	Inherited                 bool      `json:"inherited"`
	AbstractObjectType        bool      `json:"abstractObjectType"`
	ParentObjectTypeInherited bool      `json:"parentObjectTypeInherited"`
}

type ObjectAttributeValues struct {
	Value          string `json:"value"`
	DisplayValue   string `json:"displayValue"`
	SearchValue    string `json:"searchValue"`
	ReferencedType bool   `json:"referencedType"`
}

type Attributes struct {
	ID                    int                     `json:"id"`
	ObjectTypeAttributeID int                     `json:"objectTypeAttributeId"`
	ObjectAttributeValues []ObjectAttributeValues `json:"objectAttributeValues"`
	ObjectID              int                     `json:"objectId"`
}

type Links struct {
	Self string `json:"self"`
}

type ObjectEntries struct {
	ID         int          `json:"id"`
	Label      string       `json:"label"`
	ObjectKey  string       `json:"objectKey"`
	Avatar     Avatar       `json:"avatar"`
	ObjectType ObjectType   `json:"objectType"`
	Created    time.Time    `json:"created"`
	Updated    time.Time    `json:"updated"`
	HasAvatar  bool         `json:"hasAvatar"`
	Timestamp  int64        `json:"timestamp"`
	Attributes []Attributes `json:"attributes"`
	Links      Links        `json:"_links"`
	Name       string       `json:"name"`
}

type DefaultType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ReferenceType struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Color          string `json:"color"`
	URL16          string `json:"url16"`
	Removable      bool   `json:"removable"`
	ObjectSchemaID int    `json:"objectSchemaId"`
}

type ReferenceObjectType struct {
	ID                        int       `json:"id"`
	Name                      string    `json:"name"`
	Type                      int       `json:"type"`
	Icon                      Icon      `json:"icon"`
	Position                  int       `json:"position"`
	Created                   time.Time `json:"created"`
	Updated                   time.Time `json:"updated"`
	ObjectCount               int       `json:"objectCount"`
	ParentObjectTypeID        int       `json:"parentObjectTypeId"`
	ObjectSchemaID            int       `json:"objectSchemaId"`
	Inherited                 bool      `json:"inherited"`
	AbstractObjectType        bool      `json:"abstractObjectType"`
	ParentObjectTypeInherited bool      `json:"parentObjectTypeInherited"`
}

type ObjectTypeAttributes struct {
	ID                      int                 `json:"id"`
	Name                    string              `json:"name"`
	Label                   bool                `json:"label"`
	Type                    int                 `json:"type"`
	DefaultType             DefaultType         `json:"defaultType,omitempty"`
	Editable                bool                `json:"editable"`
	System                  bool                `json:"system"`
	Sortable                bool                `json:"sortable"`
	Summable                bool                `json:"summable"`
	Indexed                 bool                `json:"indexed"`
	MinimumCardinality      int                 `json:"minimumCardinality"`
	MaximumCardinality      int                 `json:"maximumCardinality"`
	Removable               bool                `json:"removable"`
	Hidden                  bool                `json:"hidden"`
	IncludeChildObjectTypes bool                `json:"includeChildObjectTypes"`
	UniqueAttribute         bool                `json:"uniqueAttribute"`
	Options                 string              `json:"options"`
	Position                int                 `json:"position"`
	Description             string              `json:"description,omitempty"`
	Suffix                  string              `json:"suffix,omitempty"`
	RegexValidation         string              `json:"regexValidation,omitempty"`
	QlQuery                 string              `json:"qlQuery,omitempty"`
	Iql                     string              `json:"iql,omitempty"`
	ReferenceType           ReferenceType       `json:"referenceType,omitempty"`
	ReferenceObjectTypeID   int                 `json:"referenceObjectTypeId,omitempty"`
	ReferenceObjectType     ReferenceObjectType `json:"referenceObjectType,omitempty"`
	AdditionalValue         string              `json:"additionalValue,omitempty"`
}

type ItSystem struct {
	ObjectEntries         []ObjectEntries        `json:"objectEntries"`
	ObjectTypeAttributes  []ObjectTypeAttributes `json:"objectTypeAttributes"`
	ObjectTypeID          int                    `json:"objectTypeId"`
	ObjectTypeIsInherited bool                   `json:"objectTypeIsInherited"`
	AbstractObjectType    bool                   `json:"abstractObjectType"`
	TotalFilterCount      int                    `json:"totalFilterCount"`
	StartIndex            int                    `json:"startIndex"`
	ToIndex               int                    `json:"toIndex"`
	PageObjectSize        int                    `json:"pageObjectSize"`
	PageNumber            int                    `json:"pageNumber"`
	OrderWay              string                 `json:"orderWay"`
	QlQuery               string                 `json:"qlQuery"`
	QlQuerySearchResult   bool                   `json:"qlQuerySearchResult"`
	ConversionPossible    bool                   `json:"conversionPossible"`
	PageSize              int                    `json:"pageSize"`
	Iql                   string                 `json:"iql"`
	IqlSearchResult       bool                   `json:"iqlSearchResult"`
}

type ObjectTypeAttribute struct {
	ID                      int                 `json:"id"`
	Name                    string              `json:"name"`
	Label                   bool                `json:"label"`
	Type                    int                 `json:"type"`
	Description             string              `json:"description"`
	ReferenceType           ReferenceType       `json:"referenceType"`
	ReferenceObjectTypeID   int                 `json:"referenceObjectTypeID"`
	ReferenceObjectType     ReferenceObjectType `json:"referenceObjectType"`
	Editable                bool                `json:"editable"`
	System                  bool                `json:"system"`
	Sortable                bool                `json:"sortable"`
	Summable                bool                `json:"summable"`
	Indexed                 bool                `json:"indexed"`
	MinimumCardinality      int                 `json:"minimumCardinality"`
	MaximumCardinality      int                 `json:"maximumCardinality"`
	Suffix                  string              `json:"suffix"`
	Removable               bool                `json:"removable"`
	Hidden                  bool                `json:"hidden"`
	IncludeChildObjectTypes bool                `json:"includeChildObjectTypes"`
	UniqueAttribute         bool                `json:"uniqueAttribute"`
	RegexValidation         string              `json:"regexValidation"`
	QlQuery                 string              `json:"qlQuery"`
	Options                 string              `json:"options"`
	Position                int                 `json:"position"`
	Iql                     string              `json:"iql"`
}

type ITObjAttributes []struct {
	ID                    int                     `json:"id"`
	ObjectID              int                     `json:"objectId"`
	ObjectAttributeValues []ObjectAttributeValues `json:"objectAttributeValues"`
	ObjectTypeAttribute   []ObjectTypeAttribute   `json:"objectTypeAttribute,omitempty"`
}

func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}

	err := fmt.Errorf("request failed. Please analyze the request body for more details. Status code: %d", r.StatusCode)
	return err
}

func GetAllItSystem() ([]models.ItSystemRecord, error) {
	var sys []ItSystem
	var errExit error
	var typeAttributes map[int]*ObjectTypeAttributes
	var itSystem []models.ItSystemRecord

	typeAttributes = make(map[int]*ObjectTypeAttributes)
	errExit = nil

	endPoint := "https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeId=21&includeAttributes=true&page=%d"
	token := "NzQ4ODU5NTExNjU1OgSo6DUdlpFFJ80jx465xJuWLZD"

	for page := 1; ; page++ {
		var sysPage ItSystem
		urlJira := fmt.Sprintf(endPoint, page)
		req, err := http.NewRequest("GET", urlJira, nil)

		req.Header.Add("Accept", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			errExit = err
			break
		}
		defer res.Body.Close()

		if CheckResponse(res) != nil {
			errExit = err
			break
		}

		body, err := io.ReadAll(res.Body)

		if err != nil {
			//return sys, err
			errExit = err
			break
		}
		err = json.Unmarshal(body, &sysPage)
		if err != nil {
			//         return sys, err
			errExit = err
			break
		}
		if len(sysPage.ObjectEntries) == 0 {
			break
		}
		sys = append(sys, sysPage)
		if sysPage.PageNumber == sysPage.PageSize {
			break
		}
	}

	for i := range sys {
		for j := range sys[i].ObjectTypeAttributes {
			typeAttributes[sys[i].ObjectTypeAttributes[j].ID] = &sys[i].ObjectTypeAttributes[j]
			//fmt.Println(sys[i].ObjectTypeAttributes[j].Name)
		}
	}

	for i := range sys {
		for j := range sys[i].ObjectEntries {
			var rec models.ItSystemRecord
			rec.ID = sys[i].ObjectEntries[j].ID
			rec.ObjectKey = sys[i].ObjectEntries[j].ObjectKey
			rec.Label = sys[i].ObjectEntries[j].Label
			rec.Name = sys[i].ObjectEntries[j].Name
			rec.Created = sys[i].ObjectEntries[j].Created
			rec.Updated = sys[i].ObjectEntries[j].Updated
			rec.Timestamp = sys[i].ObjectEntries[j].Timestamp
			setFiledObjectAttrib(&rec, &sys[i].ObjectEntries[j], typeAttributes)
			itSystem = append(itSystem, rec)
		}
	}

	return itSystem, errExit
}

type ObjAttr struct {
	ID                  int `json:"id"`
	ObjectTypeAttribute struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Label       bool   `json:"label"`
		Type        int    `json:"type"`
		DefaultType struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"defaultType"`
		Editable                bool   `json:"editable"`
		System                  bool   `json:"system"`
		Sortable                bool   `json:"sortable"`
		Summable                bool   `json:"summable"`
		Indexed                 bool   `json:"indexed"`
		MinimumCardinality      int    `json:"minimumCardinality"`
		MaximumCardinality      int    `json:"maximumCardinality"`
		Removable               bool   `json:"removable"`
		Hidden                  bool   `json:"hidden"`
		IncludeChildObjectTypes bool   `json:"includeChildObjectTypes"`
		UniqueAttribute         bool   `json:"uniqueAttribute"`
		Options                 string `json:"options"`
		Position                int    `json:"position"`
	} `json:"objectTypeAttribute"`
	ObjectTypeAttributeID int `json:"objectTypeAttributeId"`
	ObjectAttributeValues []struct {
		Value          string `json:"value"`
		DisplayValue   string `json:"displayValue"`
		SearchValue    string `json:"searchValue"`
		ReferencedType bool   `json:"referencedType"`
	} `json:"objectAttributeValues"`
	ObjectID int `json:"objectId"`
}

// Business-owner
func GetBusinessOwner(obj []ObjAttr) string {
	for i := range obj {
		if obj[i].ObjectTypeAttribute.Name == "Business-owner" {
			businesOwners := ""
			for j := range obj[i].ObjectAttributeValues {
				businesOwners += obj[i].ObjectAttributeValues[j].DisplayValue
				if obj[i].ObjectAttributeValues[j].ReferencedType {
					businesOwners += fmt.Sprintf(" (%s);", obj[i].ObjectAttributeValues[j].SearchValue)
				}
			}
			return businesOwners
		}
	}
	return ""
}

func GetAttributes(obj []ObjAttr, name string) string {
	for i := range obj {
		if obj[i].ObjectTypeAttribute.Name == name {
			businesOwners := ""
			for j := range obj[i].ObjectAttributeValues {
				businesOwners += obj[i].ObjectAttributeValues[j].DisplayValue
				if obj[i].ObjectAttributeValues[j].ReferencedType {
					businesOwners += fmt.Sprintf(" (%s);", obj[i].ObjectAttributeValues[j].SearchValue)
				}
			}
			return businesOwners
		}
	}
	return ""
}

// IT-owner
func GetITOwner(obj []ObjAttr) string {
	return ""
}

// Support
func GetSupport(obj []ObjAttr) string {
	for i := range obj {
		if obj[i].ObjectTypeAttribute.Name == "Support" {
			businesOwners := ""
			for j := range obj[i].ObjectAttributeValues {
				businesOwners += obj[i].ObjectAttributeValues[j].DisplayValue
				if obj[i].ObjectAttributeValues[j].ReferencedType {
					businesOwners += fmt.Sprintf(" (%s);", obj[i].ObjectAttributeValues[j].SearchValue)
				}
			}
			return businesOwners
		}
	}
	return ""
}

func GetAttribString(obj []ObjAttr, keyName string) string {
	for i := range obj {
		if obj[i].ObjectTypeAttribute.Name == keyName {
			businesOwners := ""
			for j := range obj[i].ObjectAttributeValues {
				businesOwners += obj[i].ObjectAttributeValues[j].DisplayValue
				if obj[i].ObjectAttributeValues[j].ReferencedType {
					businesOwners += fmt.Sprintf(" (%s);", obj[i].ObjectAttributeValues[j].SearchValue)
				}
			}
			return businesOwners
		}
	}
	return ""
}

// Status
func GetStatus(obj []ObjAttr) string {
	return ""
}

func LoadITSystemObjectAttributes(objectfile string) ([]ObjAttr, error) {
	var sys []ObjAttr
	filebuffer, err := ioutil.ReadFile(objectfile)
	if err != nil {
		return sys, err
	}
	err = json.Unmarshal(filebuffer, &sys)
	if err != nil {
		return sys, err
	}
	return sys, nil
}

func GetITSystemObjectAttributes(objectKey string) ([]ObjAttr, error) {
	var sys []ObjAttr
	var errExit error

	errExit = nil

	endPoint := "https://jira.bcs.ru/rest/assets/latest/object/%s/attributes"
	token := "NzQ0ODU5NTExNju1OgOso6DUdlpFFJ8Ojx465xJUwLZD"

	urlJira := fmt.Sprintf(endPoint, objectKey)
	req, err := http.NewRequest("GET", urlJira, nil)

	req.Header.Add("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		errExit = err
	}

	defer res.Body.Close()

	if err != nil {
		return sys, err
	}

	if err = CheckResponse(res); err != nil {
		return sys, err
	}

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return sys, err
	}
	err = json.Unmarshal(body, &sys)
	if err != nil {
		return sys, err
	}
	return sys, errExit
}

func setFiledObjectAttrib(rec *models.ItSystemRecord, obj *ObjectEntries, attr map[int]*ObjectTypeAttributes) {

	for _, o := range obj.Attributes {
		c := attr[o.ObjectTypeAttributeID]
		value := o.ObjectAttributeValues[0].Value
		diaplayValue := o.ObjectAttributeValues[0].DisplayValue
		searchValue := o.ObjectAttributeValues[0].SearchValue
		switch c.Name {
		case "Key":
			rec.Key = value
			if searchValue != "" && value == "" {
				rec.Key = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Full Name":
			rec.FullName = value
			if searchValue != "" && value == "" {
				rec.FullName = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Short Name":
			rec.ShortName = value
			if searchValue != "" && value == "" {
				rec.ShortName = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Level criticality":
			rec.LevelCriticality = value
			if searchValue != "" && value == "" {
				rec.LevelCriticality = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Description":
			rec.Description = value
			if searchValue != "" && value == "" {
				rec.Description = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Status":
			rec.Status = value
			if searchValue != "" && value == "" {
				rec.Status = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "isMonitored":
			if value == "true" {
				rec.IsMonitored = true
			} else {
				rec.IsMonitored = false
			}
		case "isBackuped":
			if value == "true" {
				rec.IsBackuped = true
			} else {
				rec.IsBackuped = false
			}
		case "System type":
			rec.SystemType = value
			if searchValue != "" && value == "" {
				rec.SystemType = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Licensing type":
			rec.LicensingType = value
			if searchValue != "" && value == "" {
				rec.LicensingType = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Solution type":
			rec.SolutionType = value
			if searchValue != "" && value == "" {
				rec.SolutionType = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Deployment type":
			rec.DeploymentType = value
			if searchValue != "" && value == "" {
				rec.DeploymentType = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "IT-System users":
			rec.ITSystemUsers = value
			if searchValue != "" && value == "" {
				rec.ITSystemUsers = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Documentation":
			rec.Documentation = value
			if searchValue != "" && value == "" {
				rec.Documentation = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "CFO":
			rec.CFO = value
			if searchValue != "" && value == "" {
				rec.CFO = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Commissioning date":
			rec.Commissioning = value
			if searchValue != "" && value == "" {
				rec.Commissioning = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Decommissioning date":
			rec.Decommissioning = value
			if searchValue != "" && value == "" {
				rec.Decommissioning = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "URL":
			rec.URL = value
			if searchValue != "" && value == "" {
				rec.URL = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Developer team":
			rec.DeveloperTeam = value
			if searchValue != "" && value == "" {
				rec.DeveloperTeam = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Support team":
			rec.SupportTeam = value
			if searchValue != "" && value == "" {
				rec.SupportTeam = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Jira project":
			rec.JiraProject = value
			if searchValue != "" && value == "" {
				rec.JiraProject = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Accounting name":
			rec.AccountingName = value
			if searchValue != "" && value == "" {
				rec.AccountingName = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Referenced Applications":
			rec.ReferencedApplication = value
			if searchValue != "" && value == "" {
				rec.ReferencedApplication = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Itilium IT-service":
			rec.ItiliumItService = value
			if searchValue != "" && value == "" {
				rec.ItiliumItService = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Data center":
			rec.DataCenter = value
			if searchValue != "" && value == "" {
				rec.DataCenter = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Referenced Databases":
			rec.ReferencedDatabase = value
			if searchValue != "" && value == "" {
				rec.ReferencedDatabase = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Project GitLab":
			rec.ProjectGitlab = value
			if searchValue != "" && value == "" {
				rec.ProjectGitlab = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Location type":
			rec.LocationType = value
			if searchValue != "" && value == "" {
				rec.LocationType = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "IP address or pool of IP addresses in IPv4":
			rec.IPAdressesV4 = value
			if searchValue != "" && value == "" {
				rec.IPAdressesV4 = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "IP address or pool of IP addresses in IPv6 protocol format":
			rec.IPAdressesV6 = value
			if searchValue != "" && value == "" {
				rec.IPAdressesV6 = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Description of the object":
			rec.DescriptionObject = value
			if searchValue != "" && value == "" {
				rec.DescriptionObject = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Autonomous system number":
			rec.AutonomousSystemNumber = value
			if searchValue != "" && value == "" {
				rec.AutonomousSystemNumber = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Domain name of the electronic address of the information resource":
			rec.DomainNameElectronicResource = value
			if searchValue != "" && value == "" {
				rec.DomainNameElectronicResource = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Uniform Information Resource Identifier":
			rec.UIRI = value
			if searchValue != "" && value == "" {
				rec.UIRI = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Phone numbers":
			rec.PhoneNumbers = value
			if searchValue != "" && value == "" {
				rec.PhoneNumbers = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Operator ID:":
			rec.OperatorId = value
			if searchValue != "" && value == "" {
				rec.OperatorId = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Full corporate name of the telecom operator":
			rec.FullCorporateName = value
			if searchValue != "" && value == "" {
				rec.FullCorporateName = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Taxpayer Identification Number":
			rec.TaxpayerIdNumber = value
			if searchValue != "" && value == "" {
				rec.TaxpayerIdNumber = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Created":
		case "Updated":
		case "Link Service":
			rec.LinkService = value
			if searchValue != "" && value == "" {
				rec.LinkService = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Last confirmation date":
			rec.LactConfirmDate = value
			if searchValue != "" && value == "" {
				rec.LactConfirmDate = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Itilium IT-service new":
			rec.ItiliumITServiceNew = value
			if searchValue != "" && value == "" {
				rec.ItiliumITServiceNew = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "URL new":
			rec.URLNew = value
			if searchValue != "" && value == "" {
				rec.URLNew = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "SD notification date":
			rec.SDNotificationDate = value
			if searchValue != "" && value == "" {
				rec.SDNotificationDate = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "BizTalk ID":
			rec.BizTalkID = value
			if searchValue != "" && value == "" {
				rec.BizTalkID = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Inbound Integrations":
			rec.InboundIntegration = value
			if searchValue != "" && value == "" {
				rec.InboundIntegration = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Outbound Integrations":
			rec.OutboundIntegration = value
			if searchValue != "" && value == "" {
				rec.OutboundIntegration = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Business column":
			rec.BusinessColumn = value
			if searchValue != "" && value == "" {
				rec.BusinessColumn = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Business-owner":
			rec.BusinessOwner = value
			if searchValue != "" && value == "" {
				rec.BusinessOwner = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "IT-owner":
			rec.ITOwner = value
			if searchValue != "" && value == "" {
				rec.ITOwner = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Vendor":
			rec.Vendor = value
			if searchValue != "" && value == "" {
				rec.Vendor = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Product owners":
			rec.ProductOwner = value
			if searchValue != "" && value == "" {
				rec.ProductOwner = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "Analyst/Methodologist":
			rec.AnalystMethodologist = value
			if searchValue != "" && value == "" {
				rec.AnalystMethodologist = fmt.Sprintf("%s (%s)", diaplayValue, searchValue)
			}
		case "QA":
		case "Support":
		case "Release Manager":
		case "Developer":
		case "Method of registering requests":
			//default:
			//	fmt.Println(c.Name) убрать трассировку новых
		}
	}
}

func LoadITSystemSpr() error {
	var err error
	it_system_spr = make(map[string]*models.ItSystemRecord)
	itSystemMutex.Lock()
	it_system, err = GetAllItSystem()
	itSystemMutex.Unlock()
	if err == nil {
		for i := range it_system {
			it_system_spr[it_system[i].ObjectKey] = &it_system[i]
		}
	}
	return err
}

func GetITSystemRec(key string) (models.ItSystemRecord, bool) {
	itSystemMutex.RLock()
	defer itSystemMutex.RUnlock()
	pRec, ok := it_system_spr[key]
	if ok {
		return *pRec, ok
	} else {
		return models.ItSystemRecord{}, ok
	}
}

func ImportITSystemSpr() []models.ItSystemRecord {
	var spr []models.ItSystemRecord
	itSystemMutex.RLock()
	for i := range it_system {
		rec := it_system[i]
		spr = append(spr, rec)
	}
	itSystemMutex.RUnlock()
	return spr
}

//запрос списка бизнесов https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeId=244&includeAttributes=true&page=1
