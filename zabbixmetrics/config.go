// Package zabbixmetrics собирает поминутные метрики хостов из Zabbix API
// для заданной группы серверов (Windows и Linux, разные версии шаблонов)
// и сохраняет их через Beego v2 ORM (пакет models, таблица server_metric).
//
// Перед запуском задайте переменные пакета Endpoint, Token, HostGroupID,
// затем вызовите Run или Collect.
package zabbixmetrics

import (
	"time"

	"loadmon/utilities"
)

// Параметры подключения к Zabbix API. Задайте их до вызова Collect / Run.
var (
	// Endpoint — полный URL JSON-RPC, например http://zabbix.example.com/api_jsonrpc.php
	Endpoint = "https://zabbixcorp.global.bcs/api_jsonrpc.php"

	// Token — API-токен Zabbix (Authorization: Bearer).
	Token = utilities.GetKeyData("spausvc")

	// HostGroupID — идентификатор группы хостов Zabbix, с которой собираются
	// метрики (1185 — только физические серверы).
	HostGroupID = 1185

	// Interval — период сбора. По умолчанию раз в минуту.
	Interval = time.Minute

	// Timeout — таймаут HTTP-запроса к Zabbix.
	Timeout = 120 * time.Second

	// HistoryWindow — окно history.get, из которого берётся последнее значение item.
	HistoryWindow = 3 * time.Minute

	// BytesPerMB — делитель для перевода байт в мегабайты (SI: 1 000 000).
	BytesPerMB = 1000.0 * 1000.0

	// InsecureSkipVerify отключает проверку TLS-сертификата Zabbix.
	InsecureSkipVerify = false
)

// ItemKeys — дополнительные пользовательские фрагменты ключей.
// Встроенный классификатор уже понимает Linux/Windows шаблоны разных поколений;
// эти списки расширяют его, если в инсталляции свои UserParameter.
var ItemKeys = Keys{
	CPUUtil:          nil,
	RAMUtil:          nil,
	DiskReadMBps:     nil,
	DiskWriteMBps:    nil,
	DiskReadLatency:  nil,
	DiskWriteLatency: nil,
	DiskReadQueue:    nil,
	DiskWriteQueue:   nil,
	DiskQueue:        nil,
	NetIn:            nil,
	NetOut:           nil,
}

// Keys описывает дополнительные item key_ сверх встроенных Linux/Windows шаблонов.
type Keys struct {
	CPUUtil          []string
	RAMUtil          []string
	DiskReadMBps     []string
	DiskWriteMBps    []string
	DiskReadLatency  []string
	DiskWriteLatency []string
	DiskReadQueue    []string
	DiskWriteQueue   []string
	DiskQueue        []string
	NetIn            []string
	NetOut           []string
}

// netSkipExact — точные имена интерфейсов (короткие, чтобы не задеть
// Windows "Local Area Connection" префиксом "lo").
var netSkipExact = []string{"lo"}

// netSkipPrefix — префиксы виртуальных/служебных интерфейсов Linux и Windows.
var netSkipPrefix = []string{
	"docker", "veth", "br-", "virbr", "tun", "tap",
	"cali", "flannel", "cni", "kube", "nodelocal",
	"isatap", "teredo", "6to4", "npcap",
}

// netSkipContains — фрагменты в имени интерфейса или в name item.
var netSkipContains = []string{
	"loopback", "wan miniport", "pseudo-interface",
	"bluetooth", "kernel debug",
}

// diskSkipExact / diskSkipPrefix — loop/ram/cdrom, не физические диски нагрузки.
var diskSkipExact = []string{"sr", "fd"}
var diskSkipPrefix = []string{"loop", "ram", "sr", "fd"}
