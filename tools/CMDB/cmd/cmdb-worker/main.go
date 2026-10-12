// cmdb-worker — планировщик и исполнитель заданий импорта CMDB (E4.2): ставит плановые задания API-коннекторам и выполняет очередь imp.job.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/worker"
	"cmdb/internal/platform/config"
	"cmdb/internal/platform/logging"
	"cmdb/internal/repo"
)

func main() {
	config.ApplyEnvOverrides()
	logger := logging.Setup(config.LogLevel())

	if err := repo.Init(config.LoadDBConfig()); err != nil {
		logger.Error("db init failed", "error", err)
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("cmdb-worker starting", "runmode", config.RunMode())
	// Планировщик и исполнитель заданий импорта (imp.job, расписание из imp.connector) — E4.2.
	(&worker.Worker{}).Loop(logging.WithRequestID(ctx, "worker"))
	logger.Info("cmdb-worker stopping")
	_ = os.Stdout.Sync()
}
