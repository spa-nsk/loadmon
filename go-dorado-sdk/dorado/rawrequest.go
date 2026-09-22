package dorado

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// RawRequestDump выполняет ПРОИЗВОЛЬНЫЙ запрос к REST API Dorado/OceanStor
// (метод+путь задаёт вызывающий код) и сохраняет СЫРОЙ (до json.Unmarshal)
// ответ устройства в файл dumpPath - по тому же принципу, что и dumpPath у
// Device.GetDisk (disk.go, см. её комментарий и CHANGELOG (174)), но не
// привязан к конкретному эндпоинту /disk/{id}.
//
// Добавлено 2026-09-22 по прямому запросу пользователя loadmon "сделай
// сохранение сырых запросов по huawei" - продолжение починки живых KPI СХД
// (см. cmdb/collectstoragemetrics.go, комментарий у case 3/4, и
// kontext/11-storage-metrics.md): используемый в проекте SDK
// (github.com/lovi-cloud/go-dorado-sdk, здесь - локальный форк) не содержит
// ни одного метода для эндпоинта живой статистики Huawei DeviceManager
// (`/performance_statistic/...`), а официальная документация с точной
// схемой запроса/ответа недоступна из текущего окружения (см.
// kontext/10-huawei-dorado-rest-api.md). Вместо того чтобы угадывать поля
// ответа без проверки ("не гадать" - тот же принцип, что уже приводил к
// находкам в CHANGELOG (77)/(84)/(86)/(174)), этот метод позволяет отправить
// ЛЮБОЙ GET/POST-запрос к реальному массиву и сохранить его настоящий ответ
// в файл для последующего разбора - как для /performance_statistic/..., так
// и для любого другого не реализованного пока в SDK эндпоинта.
//
// body - тело запроса для POST (например, JSON с параметрами подписки на
// объекты статистики) - nil для GET. Ответ разбирается в общем виде (через
// Result.Data с типом map[string]interface{}) - для целей этого метода важен
// не успешный разбор (он может не получиться, если структура ответа
// незнакома SDK), а сам факт, что dumpPath получит настоящие байты ответа:
// decodeBodyDump (http.go) пишет файл ДО попытки json.Unmarshal, поэтому
// дамп сохраняется, даже если разбор в конце вернёт ошибку.
func (d *Device) RawRequestDump(ctx context.Context, method, spath string, body []byte, dumpPath string) error {
	var reqBody io.Reader
	if len(body) > 0 {
		reqBody = bytes.NewReader(body)
	}

	req, err := d.newRequest(ctx, method, spath, reqBody)
	if err != nil {
		return fmt.Errorf(ErrCreateRequest+": %w", err)
	}

	var out map[string]interface{}
	// dumpPath сохранит сырой ответ независимо от того, успеет ли этот
	// Unmarshal - см. комментарий выше и decodeBodyDump (http.go).
	if err := d.requestWithRetryDump(req, &out, DefaultHTTPRetryCount, dumpPath); err != nil {
		return fmt.Errorf("запрос %s %s: %w (сырой ответ, если получен, всё равно сохранён в %s)", method, spath, err, dumpPath)
	}
	return nil
}
