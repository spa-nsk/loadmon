package oneview

import (
	"encoding/json"

	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
)

// SetQueryParamsNew формирует карту query-параметров для запросов к /rest/server-hardware
// и /rest/server-profiles (GetServerHardwareListNew, GetServerProfileListNew).
func SetQueryParamsNew(c *ov.OVClient, userQuery string, query string, groupUris string, sort string, start string, count string) map[string]interface{} {
	q := make(map[string]interface{})

	if userQuery != "" {
		q["userQuery"] = userQuery
	}
	if query != "" {
		q["query"] = query
	}
	if groupUris != "" {
		q["groupUris"] = groupUris
	}
	if sort != "" {
		q["sort"] = sort
	}
	if start != "" {
		q["start"] = start
	}
	if count != "" {
		q["count"] = count
	}
	return q
}

// GetServerHardwareListNew запрашивает список серверов (/rest/server-hardware) через
// собственный Redfish-парсер (используется для appliance с endpoint.Version == 300,
// см. LoadServerHardwareList в oneview.go). Ошибка разбора JSON не прерывает выполнение
// сразу - она возвращается вызывающему коду вместе с уже частично заполненным serverlist,
// чтобы можно было продолжить постраничную загрузку остальных записей.
func GetServerHardwareListNew(c *ov.OVClient, userQuery string, query string, groupUris string, sort string, start string, count string) (ServerHardwareListNew, error) {
	var (
		uri        = "/rest/server-hardware"
		serverlist ServerHardwareListNew
	)

	q := SetQueryParamsNew(c, userQuery, query, groupUris, sort, start, count)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri, nil, q)
	if err != nil {
		return serverlist, err
	}

	err = json.Unmarshal([]byte(data), &serverlist)
	return serverlist, err
}

// GetServerProfileListNew запрашивает список серверных профилей (/rest/server-profiles)
// через собственный Redfish-парсер.
func GetServerProfileListNew(c *ov.OVClient, userQuery string, query string, groupUris string, sort string, start string, count string) (interface{}, error) {
	var (
		uri        = "/rest/server-profiles"
		serverlist interface{}
	)

	q := SetQueryParamsNew(c, userQuery, query, groupUris, sort, start, count)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri, nil, q)
	if err != nil {
		return serverlist, err
	}
	if err := json.Unmarshal([]byte(data), &serverlist); err != nil {
		return serverlist, err
	}
	return serverlist, nil
}

// GetServerHardwareMemory запрашивает данные о памяти сервера по uuid
// (/rest/server-hardware/{uuid}/memory) и отфильтровывает пустые слоты (CapacityMiB == 0).
func GetServerHardwareMemory(c *ov.OVClient, uuid utils.Nstring) (ServerHardwareMemory, error) {
	var serverHardwareMemory ServerHardwareMemory

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	str := "/rest/server-hardware/" + uuid.String() + "/memory"
	data, err := c.RestAPICall(rest.GET, str, nil)
	if err != nil {
		return serverHardwareMemory, err
	}

	if err := json.Unmarshal([]byte(data), &serverHardwareMemory); err != nil {
		return serverHardwareMemory, err
	}

	dataMem := make([]MemoryModule, 0, len(serverHardwareMemory.Data))
	for _, rec := range serverHardwareMemory.Data {
		if rec.CapacityMiB > 0 {
			dataMem = append(dataMem, rec)
		}
	}
	serverHardwareMemory.Data = dataMem
	serverHardwareMemory.Count = len(dataMem)
	return serverHardwareMemory, nil
}

// GetServerHardwareExt запрашивает расширенную информацию о сервере по uuid
// (/rest/server-hardware/{uuid}).
func GetServerHardwareExt(c *ov.OVClient, uuid utils.Nstring) (ServerHardwareExt, error) {
	var serverHardware ServerHardwareExt

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	str := "/rest/server-hardware/" + uuid.String()
	data, err := c.RestAPICall(rest.GET, str, nil)
	if err != nil {
		return serverHardware, err
	}

	if err := json.Unmarshal([]byte(data), &serverHardware); err != nil {
		return serverHardware, err
	}
	return serverHardware, nil
}

// GetServerEnvConfig запрашивает конфигурацию окружения сервера по uuid
// (/rest/server-hardware/{uuid}/environmentalConfiguration): питание, охлаждение,
// расположение в стойке.
func GetServerEnvConfig(c *ov.OVClient, uuid utils.Nstring) (EnvironmentalConfiguration, error) {
	var envConf EnvironmentalConfiguration

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, "/rest/server-hardware/"+uuid.String()+"/environmentalConfiguration", nil)
	if err != nil {
		return envConf, err
	}

	if err := json.Unmarshal([]byte(data), &envConf); err != nil {
		return envConf, err
	}
	return envConf, nil
}

// GetServerHardwareLocalStorage - запрос на получение информации по локальным хранилищам сервера по uuid
// (/rest/server-hardware/{uuid}/localStorage).
func GetServerHardwareLocalStorage(c *ov.OVClient, uuid utils.Nstring) (ServerHardwareLocalStorage, error) {
	var serverHardwareLocalStorage ServerHardwareLocalStorage

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, "/rest/server-hardware/"+uuid.String()+"/localStorage", nil)
	if err != nil {
		return serverHardwareLocalStorage, err
	}

	if err := json.Unmarshal([]byte(data), &serverHardwareLocalStorage); err != nil {
		return serverHardwareLocalStorage, err
	}
	return serverHardwareLocalStorage, nil
}

// GetServerEnclosure запрашивает информацию о корзине (enclosure) по её URI.
func GetServerEnclosure(c *ov.OVClient, encuri string) (Enclosure, error) {
	var encHardware Enclosure

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, encuri, nil)
	if err != nil {
		return encHardware, err
	}

	if err := json.Unmarshal([]byte(data), &encHardware); err != nil {
		return encHardware, err
	}
	return encHardware, nil
}

// GetServerEnclosures запрашивает список корзин (/rest/enclosures) с фильтрами.
func GetServerEnclosures(c *ov.OVClient, groupUris string, sort string, start string, count string, query string, userQuery string) (ov.EnclosureList, error) {
	var (
		uri        = "/rest/enclosures"
		enclosures ov.EnclosureList
	)
	q := make(map[string]interface{})

	if len(groupUris) > 0 {
		q["groupUris"] = groupUris
	}
	if sort != "" {
		q["sort"] = sort
	}
	if start != "" {
		q["start"] = start
	}
	if count != "" {
		q["count"] = count
	}
	if userQuery != "" {
		q["userQuery"] = userQuery
	}
	if query != "" {
		q["query"] = query
	}

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri, nil, q)
	if err != nil {
		return enclosures, err
	}

	if err := json.Unmarshal([]byte(data), &enclosures); err != nil {
		return enclosures, err
	}
	return enclosures, nil
}

// GetServerILOssoUrl запрашивает ILO SSO url для сервера по uuid
// (/rest/server-hardware/{uuid}/iloSsoUrl) - используется для беспарольного входа в iLO.
func GetServerILOssoUrl(c *ov.OVClient, uuid string) (string, error) {
	var ssoILOUrl ServerSSOUrl

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, "/rest/server-hardware/"+uuid+"/iloSsoUrl", nil)
	if err != nil {
		return ssoILOUrl.IloSsoURL, err
	}

	if err := json.Unmarshal([]byte(data), &ssoILOUrl); err != nil {
		return ssoILOUrl.IloSsoURL, err
	}
	return ssoILOUrl.IloSsoURL, nil
}

// GetServerjavaRemoteConsoleUrl запрашивает url java-консоли удалённого управления
// для сервера по uuid (/rest/server-hardware/{uuid}/javaRemoteConsoleUrl).
func GetServerjavaRemoteConsoleUrl(c *ov.OVClient, uuid string) (string, error) {
	var javaILOUrl ServerjavaRemoteConsoleUrl

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, "/rest/server-hardware/"+uuid+"/javaRemoteConsoleUrl", nil)
	if err != nil {
		return javaILOUrl.JavaRemoteConsoleUrl, err
	}

	if err := json.Unmarshal([]byte(data), &javaILOUrl); err != nil {
		return javaILOUrl.JavaRemoteConsoleUrl, err
	}
	return javaILOUrl.JavaRemoteConsoleUrl, nil
}

// LoadDatcenterList запрашивает список дата-центров (/rest/datacenters) с фильтрами.
func LoadDatcenterList(c *ov.OVClient, filters []string, sort string, start string, count string) (DatacentersList, error) {
	var (
		uri         = "/rest/datacenters"
		datacenters DatacentersList
	)
	q := make(map[string]interface{})

	if len(filters) > 0 {
		q["filter"] = filters
	}
	if sort != "" {
		q["sort"] = sort
	}
	if start != "" {
		q["start"] = start
	}
	if count != "" {
		q["count"] = count
	}

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri, nil, q)
	if err != nil {
		return datacenters, err
	}

	if err := json.Unmarshal([]byte(data), &datacenters); err != nil {
		return datacenters, err
	}
	return datacenters, nil
}

// GetRacksByUri запрашивает информацию о стойке по её URI.
func GetRacksByUri(c *ov.OVClient, uri string) (Racks, error) {
	var racks Racks

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return racks, err
	}

	if err := json.Unmarshal([]byte(data), &racks); err != nil {
		return racks, err
	}
	return racks, nil
}

// GetRacksDeviceTopology запрашивает топологию размещения оборудования в стойке
// (uri + "/deviceTopology"): список устройств (корзин/серверов) и их позиции.
func GetRacksDeviceTopology(c *ov.OVClient, uri string) (RacksDeviceTopology, error) {
	var deviceTopology RacksDeviceTopology

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	data, err := c.RestAPICall(rest.GET, uri+"/deviceTopology", nil)
	if err != nil {
		return deviceTopology, err
	}

	if err := json.Unmarshal([]byte(data), &deviceTopology); err != nil {
		return deviceTopology, err
	}
	return deviceTopology, nil
}
