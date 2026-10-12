package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

// CmdbProxyController пересылает запросы /cmdb/... на бэкенд из конфигурации.
type CmdbProxyController struct {
	web.Controller
}

var proxy *httputil.ReverseProxy

func (c *CmdbProxyController) Proxy() {
	r := c.Ctx.Request

	// Роутер Beego перед вызовом контроллера сам читает тело запроса
	// (ParseForm/ParseMultipartForm) и закрывает его, поэтому восстанавливаем
	// тело из копии (см. web.BConfig.CopyRequestBody в main).
	if len(c.Ctx.Input.RequestBody) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(c.Ctx.Input.RequestBody))
		r.ContentLength = int64(len(c.Ctx.Input.RequestBody))
		// CopyBody хранит распакованные данные, заголовок больше не актуален
		r.Header.Del("Content-Encoding")
	} else if r.MultipartForm != nil {
		// для multipart CopyBody не вызывается — пересобираем тело из формы
		restoreMultipartBody(r)
	}

	proxy.ServeHTTP(c.Ctx.ResponseWriter, r)
}

// restoreMultipartBody пересобирает тело запроса из уже распарсенной
// multipart-формы, чтобы его можно было переслать на бэкенд.
func restoreMultipartBody(r *http.Request) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for key, values := range r.MultipartForm.Value {
		for _, v := range values {
			_ = w.WriteField(key, v)
		}
	}
	for key, files := range r.MultipartForm.File {
		for _, fh := range files {
			part, err := w.CreateFormFile(key, fh.Filename)
			if err != nil {
				continue
			}
			src, err := fh.Open()
			if err != nil {
				continue
			}
			_, _ = io.Copy(part, src)
			_ = src.Close()
		}
	}
	_ = w.Close()

	r.Body = io.NopCloser(&buf)
	r.ContentLength = int64(buf.Len())
	r.Header.Set("Content-Type", w.FormDataContentType())
}

func newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(target)
	baseDirector := rp.Director
	rp.Director = func(req *http.Request) {
		baseDirector(req)
		// /cmdb/... -> /...
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/cmdb")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, "/cmdb")
		// бэкенд видит свой собственный Host
		req.Host = target.Host
	}
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logs.Error("proxy error: %s %s -> %v", r.Method, r.URL.String(), err)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}
	return rp
}

func main() {
	// Нужно, чтобы роутер Beego сохранял копию тела запроса в ctx.Input.RequestBody,
	// из которой контроллер восстанавливает тело для пересылки.
	web.BConfig.CopyRequestBody = true

	scheme := web.AppConfig.DefaultString("backend_scheme", "http")
	host := web.AppConfig.DefaultString("backend_host", "127.0.0.1")
	port := web.AppConfig.DefaultInt("backend_port", 80)

	target, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, host, port))
	if err != nil {
		logs.Error("некорректный адрес бэкенда: %v", err)
		return
	}
	logs.Info("проксирование /cmdb/... -> %s", target)

	proxy = newReverseProxy(target)

	web.Router("/cmdb", &CmdbProxyController{}, "*:Proxy")
	web.Router("/cmdb/*", &CmdbProxyController{}, "*:Proxy")

	web.Run()
}
