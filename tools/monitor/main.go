// loadmon-monitor — периодическая проверка доступности metrics_grpc (gRPC)
// и server_web (HTTP) с уведомлением по электронной почте при сбоях.
//
// Сборка (из корня репозитория):
//
//	go build -o loadmon-monitor ./tools/monitor
//
// Запуск:
//
//	./loadmon-monitor -config conf/monitor.json
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	targetGRPC = "metrics_grpc (gRPC)"
	targetHTTP = "server_web (HTTP)"
)

func main() {
	configFlag := flag.String("config", "", "путь к monitor.json (по умолчанию conf/monitor.json)")
	once := flag.Bool("once", false, "выполнить одну проверку и выйти (код 0 — всё OK, 1 — есть ошибки)")
	flag.Parse()

	cfgPath := resolveConfigPath(*configFlag)
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("конфигурация: %v", err)
	}

	log.Printf("loadmon-monitor: конфиг %s, интервал %ds", cfgPath, cfg.CheckIntervalSec)
	log.Printf("  gRPC: %s", cfg.GRPC.address())
	log.Printf("  HTTP: %s", cfg.HTTP.URL)
	log.Printf("  email: %s -> %v", cfg.Email.From, cfg.Email.To)

	grpcState := &alertState{up: true}
	httpState := &alertState{up: true}
	cooldown := time.Duration(cfg.AlertCooldownSec) * time.Second

	run := func() int {
		exitCode := 0
		if err := checkGRPC(cfg); err != nil {
			exitCode = 1
			handleFailure(cfg, grpcState, targetGRPC, err, cooldown)
		} else {
			handleRecovery(cfg, grpcState, targetGRPC)
		}
		if err := checkHTTP(cfg); err != nil {
			exitCode = 1
			handleFailure(cfg, httpState, targetHTTP, err, cooldown)
		} else {
			handleRecovery(cfg, httpState, targetHTTP)
		}
		return exitCode
	}

	if *once {
		os.Exit(run())
	}

	ticker := time.NewTicker(time.Duration(cfg.CheckIntervalSec) * time.Second)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	run()
	for {
		select {
		case <-ticker.C:
			run()
		case sig := <-sigCh:
			log.Printf("остановка: %v", sig)
			return
		}
	}
}

func handleFailure(cfg *Config, st *alertState, name string, err error, cooldown time.Duration) {
	log.Printf("ОШИБКА [%s]: %v", name, err)
	if st.up || st.shouldNotify(cooldown) {
		subject := fmt.Sprintf("[loadmon-monitor] недоступен: %s", name)
		body := fmt.Sprintf(
			"Сервис loadmon недоступен.\r\n\r\n"+
				"Компонент: %s\r\n"+
				"Время: %s\r\n"+
				"Ошибка: %v\r\n",
			name, time.Now().Format(time.RFC3339), err,
		)
		if mailErr := sendEmail(cfg, subject, body); mailErr != nil {
			log.Printf("не удалось отправить email: %v", mailErr)
		} else {
			st.markNotified()
			log.Printf("отправлено email-уведомление: %s", name)
		}
	}
	st.up = false
}

func handleRecovery(cfg *Config, st *alertState, name string) {
	if st.up {
		return
	}
	log.Printf("восстановлен [%s]", name)
	subject := fmt.Sprintf("[loadmon-monitor] восстановлен: %s", name)
	body := fmt.Sprintf(
		"Сервис loadmon снова доступен.\r\n\r\n"+
			"Компонент: %s\r\n"+
			"Время: %s\r\n",
		name, time.Now().Format(time.RFC3339),
	)
	if err := sendEmail(cfg, subject, body); err != nil {
		log.Printf("не удалось отправить email о восстановлении: %v", err)
	}
	st.up = true
	st.lastAlert = time.Time{}
}
