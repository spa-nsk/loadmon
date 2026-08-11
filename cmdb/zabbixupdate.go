package cmdb

import (
	"fmt"

	zabbix "github.com/nixys/nxs-go-zabbix/v5"
)

func zabbixLogin(z *zabbix.Context, zbxHost, zbxUsername, zbxPassword string) {

	if err := z.Login(zbxHost, zbxUsername, zbxPassword); err != nil {
		fmt.Println("Login error:", err)
	} else {
		fmt.Println("Login: success")
	}
}

func zabbixLogout(z *zabbix.Context) {

	if err := z.Logout(); err != nil {
		fmt.Println("Logout error:", err)
	} else {
		fmt.Println("Logout: success")
	}
}

// ZabbixUpdate предназначена для синхронизации данных CMDB с сервером Zabbix.
// В текущей реализации основная логика закомментирована и функция ничего не выполняет.
func ZabbixUpdate() {

	/*
		var z zabbix.Context

		// Get variables from environment to login to Zabbix server
		zbxHost := "https://zabbixcorp.global.bcs/api_jsonrpc.php"
		zbxUsername := "spausvc"
		zbxPassword := "9SQ8Bc4ce0#U"
		if zbxHost == "" || zbxUsername == "" || zbxPassword == "" {
			fmt.Println("Login error: make sure environment variables 'ZABBIX_HOST', 'ZABBIX_USERNAME' and 'ZABBIX_PASSWORD' are defined")
			return
		}

		// Login to Zabbix server
		err := z.Login(zbxHost, zbxUsername, zbxPassword)
		defer z.Logout()

		if err != nil {
			fmt.Println("Login error:", err)
			return
		}

		// Get all hosts
		hObjects, _, err := z.HostGet(zabbix.HostGetParams{
			GetParameters: zabbix.GetParameters{
				Output: zabbix.SelectExtendedOutput,
			},
		})
		if err != nil {
			fmt.Println("Hosts get error:", err)
			return
		}

		// Print names of retrieved hosts
		fmt.Println("Hosts list:")
		for _, h := range hObjects {
			fmt.Println("-", h.Host)
		}
	*/

}
