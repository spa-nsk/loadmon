// Тестовая утилита: запускается без параметров, всё задано в переменных.
package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"loadmon/utilities"
	"loadmon/zabbixpower"
)

func main() {
	zabbixpower.Endpoint = "https://locator.global.bcs/api_jsonrpc.php"
	zabbixpower.Token = utilities.GetKeyData("spausvc2")
	zabbixpower.HostGroupID = 1744 // 62.1 NSK - HW
	zabbixpower.ItemName = "Present Power Reading"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := zabbixpower.GetServerPower(ctx)
	if err != nil {
		log.Fatal(err)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Name < res[j].Name })

	var total float64
	for _, r := range res {
		fmt.Printf("%-40s %8.1f %s  (%s)\n", r.Name, r.Watts, r.Units, r.Clock.Format("2006-01-02 15:04:05"))
		total += r.Watts
	}
	fmt.Printf("servers: %d, total: %.1f\n", len(res), total)
}
