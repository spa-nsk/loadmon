// Команда sqliteadmin — служебный HTTP-сервер для просмотра и администрирования
// SQLite-баз loadmon (основной и дополнительной): выполнение запросов и
// просмотр таблиц через веб-интерфейс. Рекомендуется запускать только с
// привязкой к localhost.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9090", "адрес HTTP сервера (только локальный доступ рекомендуется)")
	config := flag.String("config", "server_web_new/conf/app2.json", "путь к app2.json с DBFile/DBFileAdd")
	db := flag.String("db", "", "путь к sqlite базе (перекрывает DBFile из config)")
	dbAdd := flag.String("db-add", "", "путь ко второй sqlite базе (перекрывает DBFileAdd из config)")
	flag.Parse()

	pool := newDBPool()

	mainPath := *db
	addPath := *dbAdd
	if mainPath == "" || addPath == "" {
		cfgMain, cfgAdd, err := loadDBPaths(*config)
		if err != nil && mainPath == "" {
			log.Fatalf("не удалось прочитать config %q: %v", *config, err)
		}
		if mainPath == "" {
			mainPath = cfgMain
		}
		if addPath == "" {
			addPath = cfgAdd
		}
	}
	if mainPath == "" {
		log.Fatal("не указан путь к базе: задайте -db или DBFile в app2.json")
	}

	if err := pool.add("default", mainPath); err != nil {
		log.Fatalf("подключение default (%s): %v", mainPath, err)
	}
	if addPath != "" && addPath != mainPath {
		if err := pool.add("default-add", addPath); err != nil {
			log.Fatalf("подключение default-add (%s): %v", addPath, err)
		}
	}

	host := strings.Split(*addr, ":")[0]
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		fmt.Fprintln(os.Stderr, "предупреждение: sqliteadmin открыт не только на localhost")
	}

	if err := startServer(*addr, pool); err != nil {
		log.Fatal(err)
	}
}
