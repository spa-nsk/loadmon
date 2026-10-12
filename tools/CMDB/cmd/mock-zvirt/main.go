// mock-zvirt — стенд-заглушка oVirt REST API для разработки и демонстрации коннектора zVirt
// (plan.md E5.6): go run ./cmd/mock-zvirt -addr :18443 -tag 0000000001. Без TLS.
package main

import (
	"flag"
	"log"
	"net/http"

	"cmdb/internal/importer/connectors/zvirt/zvirtmock"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18443", "адрес прослушивания")
	tag := flag.String("tag", "0000000001", "10 hex-символов для идентификаторов и имён объектов")
	flag.Parse()

	m := zvirtmock.Default(*tag)
	log.Printf("mock zVirt на http://%s (пользователь %s, пароль %s)", *addr, m.User, m.Password)
	log.Fatal(http.ListenAndServe(*addr, m))
}
