package dorado

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Client is client for go-dorado-sdk
type Client struct {
	LocalDevice  *Device
	RemoteDevice *Device

	PortGroupName string

	Logger *log.Logger
}

// Device is device of dorado
type Device struct {
	Controllers []*url.URL
	URL         *url.URL
	HTTPClient  *http.Client
	DeviceID    string
	Token       string
	Jar         *cookiejar.Jar
	Logger      *log.Logger

	Username string
	Password string
}

// Result is response of REST API
type Result struct {
	Data  interface{} `json:"data"`
	Error ErrorResp   `json:"error"`
}

// ErrorResp is error response of REST API
type ErrorResp struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

var (
	userAgent = fmt.Sprintf("DoradoGoClient")
)

// httpMu lock to create *http.Request while to call setToken
var (
	httpMu sync.RWMutex
)

// NewClient create go-dorado-sdk client and set iBaseToken create by REST API.
func NewClient(localIPs, remoteIPs []string, username, password, portgroupName string, logger *log.Logger) (*Client, error) {
	client, err := NewClientDefaultToken(localIPs, remoteIPs, username, password, portgroupName, logger)
	if err != nil {
		return nil, err
	}

	err = client.SetToken()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewClientDefaultToken create go-dorado-sdk client.
// this function not call REST API.
func NewClientDefaultToken(localIPs, remoteIPs []string, username, password, portgroupName string, logger *log.Logger) (*Client, error) {
	// validate input value
	if len(username) == 0 {
		return nil, errors.New("username is required")
	}
	if len(password) == 0 {
		return nil, errors.New("password is required")
	}
	if len(localIPs) == 0 || len(remoteIPs) == 0 {
		return nil, errors.New("IPs is required")
	}

	if logger == nil {
		l := log.New(ioutil.Discard, "", log.LstdFlags)
		logger = l
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}

	// ИСПРАВЛЕНО (loadmon, 2026-09-06): раньше здесь стояло
	// `transport := *http.DefaultTransport.(*http.Transport)` - поверхностное
	// копирование структуры Transport. У http.Transport часть состояния
	// (карты пулов idle-соединений) - неэкспортируемые поля, ленивая
	// инициализация которых защищена собственным мьютексом ВНУТРИ ТОГО ЖЕ
	// экземпляра Transport. Если к моменту такого копирования у
	// http.DefaultTransport (общего на весь процесс - ЛЮБОЙ другой код в
	// этом же бинарнике, использующий http.Get/http.DefaultClient, мог его
	// "прогреть") эти карты уже были не nil, копия наследовала ССЫЛКУ на ТЕ
	// ЖЕ карты, но получала СВОЙ, отдельный, несинхронизированный с
	// оригиналом мьютекс - в результате DefaultTransport, эта копия и
	// любые другие такие же копии (а NewClientDefaultToken вызывается на
	// каждый опрос каждой СХД Dorado) начинали читать/писать одну и ту же
	// карту без общей блокировки. Это приводило к периодическим (зависящим
	// от того, что ещё в процессе успело "тронуть" DefaultTransport)
	// крэшам всего процесса с "fatal error: concurrent map read and map
	// write" внутри net/http.(*Transport).queueForIdleConn/getConn - см.
	// knowledge_base/CHANGELOG.md в проекте loadmon. Исправлено: строим
	// СОВЕРШЕННО НЕЗАВИСИМЫЙ Transport{} (без копирования чужого
	// экземпляра), с теми же значениями по умолчанию, что и у
	// http.DefaultTransport в текущих версиях Go, - никаких общих внутренних
	// карт/мьютексов с DefaultTransport или с другими такими же клиентами
	// больше нет.
	newTransport := func() *http.Transport {
		return &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSClientConfig:       &tlsConfig,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	// ИСПРАВЛЕНО (loadmon, 2026-09-06): раньше LocalDevice и RemoteDevice
	// получали ОДИН И ТОТ ЖЕ *http.Client (только с разными Jar), а
	// newRequest()/getToken() ниже безусловно делают `d.HTTPClient.Jar =
	// d.Jar` при КАЖДОМ запросе - т.е. один общий httpClient.Jar
	// перезаписывался то на Jar локального, то на Jar удалённого устройства.
	// При параллельном использовании Local/Remote (или просто при гонках в
	// вызывающем коде) это создавало доп. риск гонки на поле Jar. Теперь у
	// каждого устройства - свой независимый *http.Client (и свой Transport),
	// так что писать в чужой httpClient.Jar в принципе невозможно.
	localDevice, err := newDevice(localIPs, username, password, &http.Client{Transport: newTransport()}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Local Device: %w", err)
	}

	remoteDevice, err := newDevice(remoteIPs, username, password, &http.Client{Transport: newTransport()}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Remote Device: %w", err)
	}

	c := &Client{
		LocalDevice:   localDevice,
		RemoteDevice:  remoteDevice,
		PortGroupName: portgroupName,
		Logger:        logger,
	}

	return c, nil
}

func newDevice(ips []string, username, password string, httpClient *http.Client, logger *log.Logger) (*Device, error) {
	var parsedURLs []*url.URL
	for _, ipStr := range ips {
		parsed, err := url.Parse(ipStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse input urls: %w", err)
		}

		parsedURLs = append(parsedURLs, parsed)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookiejar: %w", err)
	}

	d := &Device{
		Controllers: parsedURLs,
		HTTPClient:  httpClient,
		Username:    username,
		Password:    password,
		Jar:         jar,
		Logger:      logger,
	}

	return d, nil
}

func (d *Device) setBaseURL(baseHost, token string) error {
	urlStr := fmt.Sprintf("%s/deviceManager/rest/%s", baseHost, token)
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("failed to parse url: %w", err)
	}

	d.URL = parsedURL
	return nil
}

func (d *Device) newRequest(ctx context.Context, method, spath string, body io.Reader) (*http.Request, error) {
	httpMu.RLock()
	u := *d.URL
	u.Path = path.Join(d.URL.Path, spath)
	httpMu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create new HTTP Request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	req.Header.Set("iBaseToken", d.Token)
	d.HTTPClient.Jar = d.Jar

	return req, nil
}
