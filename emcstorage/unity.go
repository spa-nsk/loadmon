// Package emcstorage реализует клиент для сбора информации со СХД EMC/Dell Unity через REST API
// (/api/types/..., /api/instances/...): аутентификация с CSRF-токеном (unity.go), а также получение
// сведений о системе, пулах, ресурсах хранения, дисках, LUN, хостах и метриках производительности.
package emcstorage

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
)

//Session struct
type Session struct {
	server   string
	insecure bool
	token    string
	http     http.Client
}

// EncodeCredentials purpose is to encode the Username and Password with base64 encoding which is
// required for UNITY
func EncodeCredentials(username string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// URL purpose is to return the main entry point for the UNITY API
func URL(server string, URI string) string {
	return "https://" + server + URI
}

//NewSession purpose is to build a session object and authenticate to the Unity array
func NewSession(adress string, insecure bool, login, pass string) (*Session, error) {
	if adress == "" || login == "" || pass == "" {
		return nil, errors.New("Missing server (Unity IP/FQDN), username, or password")
	}

	var httpClient http.Client
	cookieJar, _ := cookiejar.New(nil)

	// Create http client
	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = http.Client{Transport: tr, Jar: cookieJar}
	} else {
		httpClient = http.Client{Jar: cookieJar}
	}

	// Creates a http Request pointer
	var req *http.Request

	// Creates a GET HTTP Request to the UNITY API
	req, _ = http.NewRequest("GET", URL(adress, "/api/types/system/instances"), nil)

	// Add the Authorization header with the value of base64 encoded Username and Password
	// and X-EMC-REST-CLIENT header that is required if using basic authentication
	req.Header.Set("Authorization", "Basic "+EncodeCredentials(login, pass))
	req.Header.Set("X-EMC-REST-CLIENT", "true")

	// Execute request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Get the CSRF Token
	token := resp.Header.Get("Emc-Csrf-Token")

	return &Session{adress, insecure, token, httpClient}, nil
}

//Request purpose is to send a Rest API request to the Unity array.
func (session *Session) Request(method string, URI string, fields string, filter string, engineering bool, body, resp interface{}) error {
	return session.RequestPage(method, URI, fields, filter, engineering, 0, body, resp)
}

// RequestPage - то же самое, что Request, но с поддержкой постраничной
// выборки коллекций Unity REST API (`page=N` в query-строке). Добавлено
// 2026-09-23 по факту диагностики: `GET /api/types/metric/instances` без
// явного `fields` (как раньше делал storages.DumpUnityMetricCatalog) в
// compact-режиме отдаёт ТОЛЬКО поле `id` (подтверждено реальным дампом
// пользователя, `unity-metric-catalog1.json`) - для path/name/
// isRealtimeAvailable и т.п. нужно явно перечислять `fields`. При этом
// коллекция метрик на реальном массиве превышает один "page" (в том же
// дампе видно `"links": [..., {"rel": "next", "href": "&page=2"}]`
// - без обхода страниц `Request` молча возвращал бы только первую
// страницу (2000 записей из большего числа), и это осталось бы незаметным
// (в отличие от ошибки, пустой ИЛИ неполный ответ выглядит "успешно"). См.
// storages.DumpUnityMetricCatalog и kontext/12-unity-rest-api.md.
// page<=0 - без параметра page (как раньше, первая/единственная страница).
func (session *Session) RequestPage(method string, URI string, fields string, filter string, engineering bool, page int, body, resp interface{}) error {

	if method == "" || URI == "" || resp == nil {
		return errors.New("Missing method, URI or response interface")
	}

	// build endpoint URL
	endpoint := URL(session.server, URI)

	// create a http Request pointer
	var req *http.Request

	if body != nil {
		// Parse out body struct into JSON
		bodyBytes, _ := json.Marshal(body)

		// Create new http request
		req, _ = http.NewRequest(method, endpoint, bytes.NewBuffer(bodyBytes))

	} else {

		// Create new http request
		req, _ = http.NewRequest(method, endpoint, nil)

	}

	// In case of POST or DELETE method, add the CSRF token in the request header
	if method == "POST" || method == "DELETE" {
		req.Header.Set("Emc-Csrf-Token", session.token)
	}

	// Add the mandatory headers to the request
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-EMC-REST-CLIENT", "true")

	if method != "DELETE" {
		// Create an URL query object
		a := req.URL.Query()

		if fields != "" {
			a.Add("fields", fields)
		}

		if filter != "" {
			a.Add("filter", filter)
		}

		if engineering == true {
			a.Add("visibility", "Engineering")
		}

		if page > 0 {
			a.Add("page", fmt.Sprintf("%d", page))
		}

		a.Add("compact", "true")

		req.URL.RawQuery = a.Encode()
	}

	// Perform request
	httpResp, err := session.http.Do(req)
	if err != nil {
		return err
	}

	// Cleanup Response
	defer httpResp.Body.Close()

	switch {
	case httpResp.StatusCode == 200 || httpResp.StatusCode == 201 || httpResp.StatusCode == 202:
		// Decode JSON of response into our interface defined for the specific request sent
		body, err := ioutil.ReadAll(httpResp.Body)
		if err != nil {
			return err
		}

		//fmt.Println(string(body))
		// Unmarshal the body into a struct
		bodyByte := json.Unmarshal(body, resp)

		return bodyByte
	case httpResp.StatusCode == 204:
		return nil

	default:
		// ИСПРАВЛЕНО (2026-09-17 - изначально только для 422, по жалобе
		// пользователя loadmon "журнал СХД пустой, диски/полки/пулы не
		// появились": раньше тело ответа не читалось вовсе, ошибка
		// форматировалась как "HTTP status codes: %d" без единой детали,
		// откуда реально не так. РАСШИРЕНО 2026-09-23 на ВСЕ коды ошибок
		// (не только 422) - по жалобе "по Unity записывается только
		// UsedPercent, ошибка HTTP status codes: 500" у нового
		// collectUnityKpi (real-time KPI, metricRealTimeQuery): ветка
		// default для 500 (и любого другого кода, кроме
		// 200/201/202/204) точно так же не читала тело - реальное
		// сообщение Unity (обычно JSON вида {"error":{"messages":[...]}},
		// объясняющее ПОЧЕМУ запрос отклонён - например, недопустимая
		// комбинация путей метрик в одном real-time-запросе) терялось,
		// оставался только голый код. Теперь тело читается для ЛЮБОГО
		// кода ошибки, не только 422.
		body, readErr := ioutil.ReadAll(httpResp.Body)
		if readErr != nil {
			return fmt.Errorf("HTTP status codes: %d, detail: не удалось прочитать тело ответа: %v", httpResp.StatusCode, readErr)
		}
		return fmt.Errorf("HTTP status codes: %d, detail: %s", httpResp.StatusCode, string(body))
	}
}

//CloseSession purpose is to end the session with the Unity array.
func (session *Session) CloseSession() (err error) {
	err = session.Request("POST", "/api/types/loginSessionInfo/action/logout", "", "", false, nil, nil)
	return err
}
