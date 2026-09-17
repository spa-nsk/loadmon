package dorado

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func decodeBody(resp *http.Response, out interface{}, logger *log.Logger) error {
	return decodeBodyDump(resp, out, logger, "")
}

// decodeBodyDump - тот же decodeBody, но с опциональным сохранением СЫРОГО
// (до разбора json.Unmarshal) тела ответа СХД в файл dumpPath - добавлено
// 2026-09-17 по прямому запросу пользователя loadmon "добавь SDK Dorado
// сохранение сырого ответа от СХД в файл для запросов /disk/{id} ... это
// нужно для отладки". Используется точечно (сейчас - только
// Device.GetDisk, см. disk.go) там, где нужно посмотреть ИМЕННО то, что
// реально прислало устройство, без потерь на этапе json.Unmarshal - тот же
// мотив, что уже привёл к находкам в CHANGELOG (77)/(84)/(86) (реальные
// поля/типы расходились с документацией/предположениями). dumpPath=""
// (по умолчанию, через decodeBody) - поведение не меняется, запись не
// делается.
//
// Ошибка записи файла НЕ прерывает обработку ответа (тот же принцип, что и
// у остальных diagnostics-моментов в проекте loadmon - диагностика не
// должна ломать основную функциональность) - только логируется через
// logger.
func decodeBodyDump(resp *http.Response, out interface{}, logger *log.Logger, dumpPath string) error {
	defer resp.Body.Close()
	jb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if dumpPath != "" {
		if werr := ioutil.WriteFile(dumpPath, jb, 0o644); werr != nil {
			logger.Printf("dorado: не удалось сохранить сырой ответ в %s (не прерывает обработку ответа): %v", dumpPath, werr)
		} else {
			logger.Printf("dorado: сырой ответ сохранён в %s", dumpPath)
		}
	}

	r := &Result{
		Data: out,
	}
	if err := json.Unmarshal(jb, r); err != nil {
		logger.Printf("Dorado response: %v", string(jb))
		return fmt.Errorf("failed to unmarshal response JSON: %w", err)
	}

	if r.Error.Error() != nil {
		logger.Printf("Dorado return error: %v", r.Error.Error())
		return r.Error.Error()
	}

	out = r.Data
	return nil
}

func (e ErrorResp) Error() error {
	switch e.Code {
	case 0:
		// no error
		return nil
	case ErrorCodeUnAuthorized, ErrorCodeUserIsOffline:
		// please retry
		return ErrUnAuthorized
	}

	return fmt.Errorf("Dorado Internal Error: %s (code: %d) Suggestion: %s", e.Description, e.Code, e.Suggestion)
}

// requestWithRetry do HTTP Request and retry if return UnAuthorized token.
// set false in retried when call from outer.
func (d *Device) requestWithRetry(req *http.Request, out interface{}, retryCount int) error {
	return d.requestWithRetryDump(req, out, retryCount, "")
}

// requestWithRetryDump - тот же requestWithRetry, но прокидывает dumpPath в
// decodeBodyDump (см. её комментарий) на каждую попытку, включая повтор
// после обновления токена - чтобы сырой ответ сохранялся и в том случае,
// если первая попытка упёрлась в истёкший iBaseToken и понадобился повтор.
func (d *Device) requestWithRetryDump(req *http.Request, out interface{}, retryCount int, dumpPath string) error {
	resp, err := d.request(req)
	if err != nil {
		return fmt.Errorf("failed to request: %w", err)
	}

	err = decodeBodyDump(resp, out, d.Logger, dumpPath)
	if err == ErrUnAuthorized && retryCount > 0 {
		// retry after refresh token
		// need update iBaseToken and ismsession in Cookie
		err = d.setToken()
		if err != nil {
			return fmt.Errorf("failed to setToken: %w", err)
		}

		spath := strings.TrimPrefix(req.URL.Path, d.URL.Path)
		var jb []byte
		if req.GetBody != nil {
			b, err := req.GetBody()
			if err != nil {
				return fmt.Errorf("failed to GetBody: %w", err)
			}

			jb, err = ioutil.ReadAll(b) // NOTE(whywaita): need to fix many memory allocation if occurred problem
			if err != nil {
				return fmt.Errorf("failed to ReadAll: %w", err)
			}
		}

		newReq, err := d.newRequest(req.Context(), req.Method, spath, bytes.NewBuffer(jb))
		if err != nil {
			return fmt.Errorf("failed to create new http request: %w", err)
		}

		return d.requestWithRetryDump(newReq, out, retryCount-1, dumpPath)
	}

	if err != nil {
		return fmt.Errorf(ErrDecodeBody+": %w", err)
	}

	return nil
}

func (d *Device) request(req *http.Request) (*http.Response, error) {
	d.Logger.Printf("Do Request %+v\n", req)

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(ErrHTTPRequestDo+": %w", err)
	}

	return resp, nil
}
