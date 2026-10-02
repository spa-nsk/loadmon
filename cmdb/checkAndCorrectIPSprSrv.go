package cmdb

import (
	"fmt"
	"loadmon/models"
	"net"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// CheckAndCorrectIPSprSrv проверяет и корректирует по данным DNS записи серверов в таблице
// server_spr_records: приводит FQDN-имя к каноничному виду, обновляет IP-адреса и
// при отсутствии сервера в DNS помечает запись как неактивную.
func CheckAndCorrectIPSprSrv() {
	records := make([]*models.ServerSprRecords, 0)
	o := orm.NewOrm()
	qs := models.ServerSprQS(o)
	qs = qs.OrderBy("name").Filter("Type__ne", 1) // сортировка по именам серверов и фильтр по имени и Type !=1
	num, err := qs.All(&records)
	if err != nil || num < 1 {
		fmt.Println("Запрос серверов по фильтру Name=", "Ошибка: ", err)
		return
	}

	for _, rec := range records {
		s := models.ServerSprRecords{Id: rec.Id}

		o.Read(&s, "Id")
		s.NameFQDN = strings.TrimSuffix(s.NameFQDN, ".")
		o.Update(&s)

		ips, err := net.LookupIP(rec.Name)
		if err != nil {
			fmt.Println(rec.Name, rec.NameFQDN, "Нет данных в DNS")
			srv := models.ServerSprRecords{Id: rec.Id}
			err := o.Read(&srv, "Id")
			if err == nil {
				srv.Stat = 0
				// 2026-09-22: audit-helper вместо прямого присвоения -
				// источник models.UserSourceManual, т.к. это внутренняя
				// проверка наличия записи в DNS.
				models.SetActiveWithAudit(&srv.Active, &srv.TimeActiveState, &srv.ActivatorUser, &srv.TimeDeActiveState, &srv.DeActivatorUser, 2, models.UserSourceManual)
				srv.PowerState = 2
				o.Update(&srv)
			}
			continue
		}
		fmt.Println(rec.Name, rec.NameFQDN)
		ipstr := ""
		for _, ip := range ips {
			addr := ip.String()
			ipstr = fmt.Sprintf("%s%s;", ipstr, addr)
			fmt.Printf("%s\n", addr)
			hosts, err := net.LookupAddr(addr)
			if err == nil {
				foundNameFQDN := false
				for _, h := range hosts {
					host := strings.TrimSuffix(h, ".")
					fmt.Printf("\t%s\n", host)
					if !foundNameFQDN {
						hostUP := strings.ToUpper(host)
						nameUP := strings.ToUpper(rec.Name)
						nameFQDNUP := strings.ToUpper(rec.NameFQDN)
						if strings.Contains(hostUP, nameUP) && (rec.NameFQDN == "" || nameFQDNUP == nameUP) {
							foundNameFQDN = true
							srv := models.ServerSprRecords{Id: rec.Id}
							err := o.Read(&srv, "Id")
							if err == nil {
								srv.NameFQDN = host
								o.Update(&srv)
							}
						}
					}
				}
			}
		}
		if rec.IPAddress == "" && ipstr != "" {
			srv := models.ServerSprRecords{Id: rec.Id}
			err := o.Read(&srv, "Id")
			if err == nil {
				srv.IPAddress = ipstr
				o.Update(&srv)
			}
		}
	}
}
