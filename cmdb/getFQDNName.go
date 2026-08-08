package cmdb

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// return NameFQDN, IP, err
func GetFQDNNameAndIP(startname string) (string, string, error) {

	//s.NameFQDN = strings.TrimSuffix(s.NameFQDN, ".")

	ips, err := net.LookupIP(startname)
	if err != nil {
		//fmt.Println(startname, "Нет данных в DNS")
		return startname, "", err
	}

	ipstr := ""
	for _, ip := range ips {
		addr := ip.String()
		ipstr = fmt.Sprintf("%s%s;", ipstr, addr)
		hosts, err := net.LookupAddr(addr)
		if err == nil {
			foundNameFQDN := false
			for _, h := range hosts {
				host := strings.TrimSuffix(h, ".")

				if !foundNameFQDN {
					hostUP := strings.ToUpper(host)
					nameUP := strings.ToUpper(startname)
					if strings.Contains(hostUP, nameUP) {
						foundNameFQDN = true
						return host, ipstr, nil
					}
				}
			}
		}
	}
	return startname, ipstr, errors.New("Нет данных")
}
