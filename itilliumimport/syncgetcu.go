package itilliumimport

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/itilium"
	"loadmon/models"
	"loadmon/utilities"
)

// Порядок: корзины до блейдов.
var syncCUClasses = []int{
	itilium.CUClassBladeEnclosures, // 159
	itilium.CUClassBlades,          // 142
	itilium.CUClassServers,         // 145
	itilium.CUClassStorage,         // 158
	itilium.CUClassTapeLibraries,   // 198
	itilium.CUClassNetwork,         // 146
}

// SyncGetCU — синхронизация CMDB из 1С:ITILIUM getCU (все syncCUClasses).
// Параметры — ConfigItilium; пароль только через utilities.GetKeyData(Login).
// Вызывается из metrics_grpc после обновления оборудования; не паникует —
// ошибки логирует и возвращает.
func SyncGetCU() {
	cfg := config.ConfigData.ConfigItilium
	if !cfg.Enabled || !cfg.SyncRackServers {
		return
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Login) == "" {
		log.Printf("itilliumimport.SyncGetCU: ConfigItilium.BaseURL/Login пусты — пропуск")
		return
	}

	pass := utilities.GetKeyData(strings.TrimSpace(cfg.Login))
	if pass == "" || pass == "key" {
		log.Printf("itilliumimport.SyncGetCU: пароль для Login=%q не найден в GetKeyData — пропуск", cfg.Login)
		return
	}

	// Перезапись журнала только когда sync реально стартует.
	_ = resetImportLog()
	defer flushImportLog()

	// http(s)_proxy часто ломает NTLM к публикации 1С.
	httpKey, httpsKey := "http_proxy", "https_proxy"
	httpEnv, httpsEnv := os.Getenv(httpKey), os.Getenv(httpsKey)
	defer os.Setenv(httpKey, httpEnv)
	defer os.Setenv(httpsKey, httpsEnv)
	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	auth := strings.TrimSpace(cfg.AuthMode)
	if auth == "" {
		if strings.TrimSpace(cfg.Domain) != "" {
			auth = itilium.AuthNTLM
		} else {
			auth = itilium.AuthBasic
		}
	}
	timeout := 2 * time.Minute
	if cfg.TimeoutSec > 0 {
		timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	cuStatus := strings.TrimSpace(cfg.CUStatus)
	if cuStatus == "" {
		cuStatus = itilium.CUStatusAll
	}

	client, err := itilium.NewClient(itilium.Config{
		Name: "metrics-getcu", BaseURL: cfg.BaseURL, Login: cfg.Login, Password: pass,
		Domain: cfg.Domain, AuthMode: auth, APIPrefix: cfg.APIPrefix,
		Timeout: timeout, InsecureSkipVerify: cfg.InsecureSkipVerify,
	})
	if err != nil {
		log.Printf("itilliumimport.SyncGetCU: NewClient: %v", err)
		appendImportLogRow(0, models.ItiliumImportResultError, "", "NewClient: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	api := client.API()

	log.Printf("itilliumimport.SyncGetCU: старт classes=%v status=%s", syncCUClasses, cuStatus)
	for _, cuClass := range syncCUClasses {
		setImportLogCUClass(cuClass)
		rows, err := fetchAllCU(ctx, api, cuClass, cuStatus)
		if err != nil {
			log.Printf("itilliumimport.SyncGetCU: getCU cuClass=%d: %v", cuClass, err)
			appendImportLogRow(cuClass, models.ItiliumImportResultError, "", "getCU: "+err.Error())
			continue
		}
		st := importCURows(rows, cuClass)
		log.Printf("itilliumimport.SyncGetCU: cuClass=%d done inserted=%d updated=%d skipped=%d errors=%d",
			cuClass, st.Inserted, st.Updated, st.Skipped, st.Errors)
	}
	log.Printf("itilliumimport.SyncGetCU: завершено")
}

func fetchAllCU(ctx context.Context, api *itilium.APIClient, cuClass int, cuStatus string) ([]itilium.CURow, error) {
	q := itilium.GetCUQuery{
		CUClass:  cuClass,
		CUStatus: cuStatus,
		Fields:   itilium.DefaultFieldsForClass(cuClass),
	}
	var all []itilium.CURow
	for {
		resp, err := api.GetCU(ctx, q)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Rows...)
		next, ok := resp.NextPageQuery(q)
		if !ok {
			break
		}
		q = next
	}
	return all, nil
}

func importCURows(rows []itilium.CURow, cuClass int) ImportCUStats {
	switch cuClass {
	case itilium.CUClassServers:
		return ImportRackServersFromCU(rows)
	case itilium.CUClassBlades:
		return ImportBladeServersFromCU(rows)
	case itilium.CUClassBladeEnclosures:
		return ImportBladeEnclosuresFromCU(rows)
	case itilium.CUClassStorage:
		return ImportStorageFromCU(rows)
	case itilium.CUClassTapeLibraries:
		return ImportTapeLibrariesFromCU(rows)
	case itilium.CUClassNetwork:
		return ImportNetworkEquipmentFromCU(rows)
	default:
		st := ImportCUStats{Total: len(rows), Skipped: len(rows)}
		return st
	}
}
