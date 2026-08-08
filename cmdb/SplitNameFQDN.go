package cmdb

import (
	"github.com/beego/beego/v2/core/validation"
)

func SplitNameFQDN(nameFQDN string) (string, string) {
	name := nameFQDN
	domain := ""
	valid := validation.Validation{}
	if valid.IP(nameFQDN, "IP").Ok { //если это ip address
		name = nameFQDN
	} else {
		for i, ch := range nameFQDN {
			if ch == '.' {
				v := []rune(nameFQDN)
				name = string(v[:i])
				l := len(nameFQDN)
				l = l - (i + 1)
				if l > 0 {
					domain = string(v[i+1:])
				}
				break
			}
		}
	}
	return name, domain
}
