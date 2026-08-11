package vspheremetrics

import (
	"log"
	"strings"

	"net/url"

	"github.com/vmware/govmomi"

	"golang.org/x/net/context"

	"errors"
)

func normalizeVCenterHost(hostname string) string {
	host := strings.TrimSpace(hostname)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimSuffix(host, "/sdk")
	return host
}

// Connect устанавливает соединение govmomi с сервером vCenter по учётным данным
// структуры VCenter (небезопасное TLS-соединение, insecure=true).
func (vcenter *VCenter) Connect() (*govmomi.Client, error) {

	if vcenter == nil {
		return nil, errors.New("ошибка VCenter не инициализирован")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	username := url.QueryEscape(vcenter.Username)
	password := url.QueryEscape(vcenter.Password)
	host := normalizeVCenterHost(vcenter.Hostname)
	u, err := url.Parse("https://" + username + ":" + password + "@" + host + "/sdk")
	if err != nil {
		log.Printf("vcenter %s: ошибка разбора строки кодключения vcenter - %s\n", vcenter.Hostname, err)
		return nil, err
	}
	client, err := govmomi.NewClient(ctx, u, true) //insecure false
	if err != nil {
		log.Printf("vcenter %s: невозможно подключиться к vcenter - %s username=%s\n", vcenter.Hostname, err, vcenter.Username)
		return nil, err
	}
	return client, nil
}

// Init инициализирует все карты (cmap) глобальной инфраструктуры VMInfrastructure перед началом работы.
