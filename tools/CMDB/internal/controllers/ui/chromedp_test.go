// E3.9: сквозные UI-тесты в настоящем headless-браузере (Chrome через chromedp) — проверяют то,
// что не видно из HTML/JS-кода: реально ли отрабатывает клиентский JS (переключение темы, живой
// разбор имени по HTMX, открытие/закрытие Bootstrap-модалки после HTMX-подмены #card-tabs — риск,
// явно отмеченный как непроверенный в E3.1: hx-swap="outerHTML" подменяет и саму модалку, Bootstrap
// не успевает вызвать hide(), без ручной подчистки в app.js модалка осталась бы "залипшей").
//
// Пропускается, если в PATH нет браузера (google-chrome/chromium) — как тесты БД пропускаются без
// CMDB_TEST_DB_DSN: отсутствие окружения не должно ронять обычный `go test`.
// CMDB_TEST_DB_DSN=... go test ./internal/controllers/ui/... -run TestChromeUI -v
package ui_test

import (
	"context"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	web "github.com/beego/beego/v2/server/web"
	"github.com/chromedp/chromedp"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

// chromeBinary возвращает путь к браузеру или "", если ни один из известных исполняемых файлов не найден.
func chromeBinary() string {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// newChromeServer поднимает настоящий HTTP-сервер поверх beego-роутера (тот же web.BeeApp.Handlers,
// что setup() уже готовит для send()) и настраивает StaticDir абсолютным путём — рабочая директория
// `go test` это каталог пакета, а не корень репозитория, относительный "static" по умолчанию не найдётся.
func newChromeServer(t *testing.T) *httptest.Server {
	t.Helper()
	staticDir, err := filepath.Abs(filepath.Join("..", "..", "..", "static"))
	if err != nil {
		t.Fatal(err)
	}
	web.BConfig.WebConfig.StaticDir["/static"] = staticDir
	srv := httptest.NewServer(web.BeeApp.Handlers)
	t.Cleanup(srv.Close)
	return srv
}

// newChromeCtx запускает headless-браузер; t.Skip, если браузера нет в PATH (см. package doc).
func newChromeCtx(t *testing.T) context.Context {
	t.Helper()
	bin := chromeBinary()
	if bin == "" {
		t.Skip("браузер (google-chrome/chromium) не найден в PATH — пропуск chromedp-теста")
	}
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(bin),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-gpu", true),
		)...)
	t.Cleanup(cancel)
	ctx, cancel := chromedp.NewContext(allocCtx)
	t.Cleanup(cancel)
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// TestChromeUI_ThemeToggle — клик по кнопке "Тема" реально переключает data-bs-theme (чистый
// клиентский JS, static/js/app.js, без обращения к серверу).
func TestChromeUI_ThemeToggle(t *testing.T) {
	setup(t)
	srv := newChromeServer(t)
	ctx := newChromeCtx(t)

	var theme string
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/cis"),
		chromedp.WaitVisible(`#theme-toggle`, chromedp.ByQuery),
		chromedp.Click(`#theme-toggle`, chromedp.ByQuery),
		chromedp.AttributeValue(`html`, "data-bs-theme", &theme, nil, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	if theme != "dark" {
		t.Errorf("data-bs-theme после клика = %q, want dark", theme)
	}
}

// TestChromeUI_NamingSandbox — живой разбор имени по HTMX (ввод -> keyup delay -> запрос ->
// подмена #naming-result) действительно происходит в браузере, не только по прямому curl.
func TestChromeUI_NamingSandbox(t *testing.T) {
	setup(t)
	srv := newChromeServer(t)
	ctx := newChromeCtx(t)

	var resultText string
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/admin/naming"),
		chromedp.WaitVisible(`#name`, chromedp.ByQuery),
		chromedp.SendKeys(`#name`, "AMSDC1-S-01000", chromedp.ByQuery),
		chromedp.WaitVisible(`#naming-result .badge`, chromedp.ByQuery),
		chromedp.Text(`#naming-result`, &resultText, chromedp.ByQuery, chromedp.NodeVisible),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	if !strings.Contains(resultText, "legacy") {
		t.Errorf("#naming-result после ввода имени = %q, want содержит legacy", resultText)
	}
}

// TestChromeUI_ComponentModalOpensSubmitsAndCloses — E3.1: открытие модалки, отправка формы,
// автозакрытие без залипшего backdrop после hx-swap="outerHTML" на #card-tabs. Именно это осталось
// непроверенным при первой реализации модальных форм (проверялось только чтением разметки).
func TestChromeUI_ComponentModalOpensSubmitsAndCloses(t *testing.T) {
	setup(t) // регистрирует маршруты (sync.Once) и отключает XSRF для тестов — testutil.DB(t) одной БД не хватает
	srv := newChromeServer(t)
	ctx := newChromeCtx(t)

	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура chromedp", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	cardURL := srv.URL + "/cis/" + strconv.FormatInt(res.CIID, 10)

	var backdropCountBefore, backdropCountAfterOpen, backdropCountAfterSubmit int
	var toastText string
	err = chromedp.Run(ctx,
		chromedp.Navigate(cardURL),
		chromedp.WaitVisible(`#card-tabs`, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(`document.querySelectorAll('.modal-backdrop').length`, &backdropCountBefore),

		// Перейти на вкладку "Компоненты" (HTMX-ссылка, не обычная навигация).
		chromedp.Click(`a[hx-get$="/tab/components"]`, chromedp.ByQuery),
		chromedp.WaitReady(`#modal-add-component`, chromedp.ByQuery), // модалка в DOM, но скрыта (display:none) до клика

		// Открыть модалку и дождаться, пока Bootstrap добавит класс "show" (анимация открытия).
		chromedp.Click(`button[data-bs-target="#modal-add-component"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`#modal-add-component.show`, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(`document.querySelectorAll('.modal-backdrop').length`, &backdropCountAfterOpen),

		// Заполнить и отправить форму — hx-post внутри модалки, ответ подменяет #card-tabs целиком.
		chromedp.SetValue(`#modal-add-component select[name="component_type"]`, "cpu", chromedp.ByQuery),
		chromedp.SendKeys(`#modal-add-component input[name="manufacturer"]`, "Dell", chromedp.ByQuery),
		chromedp.SendKeys(`#modal-add-component input[name="reason"]`, "chromedp-тест", chromedp.ByQuery),
		chromedp.Click(`#modal-add-component button[type="submit"]`, chromedp.ByQuery),

		// После успешной отправки: тост "success" и НИ ОДНОГО .modal-backdrop, оставшегося от
		// пересозданной (после outerHTML-подмены) закрытой по умолчанию модалки.
		chromedp.WaitVisible(`.toast`, chromedp.ByQuery),
		chromedp.Text(`.toast-body`, &toastText, chromedp.ByQuery, chromedp.NodeVisible),
		chromedp.EvaluateAsDevTools(`document.querySelectorAll('.modal-backdrop').length`, &backdropCountAfterSubmit),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}

	if backdropCountBefore != 0 {
		t.Errorf("backdrop до открытия модалки = %d, want 0", backdropCountBefore)
	}
	if backdropCountAfterOpen == 0 {
		t.Errorf("backdrop после открытия модалки = %d, want >0 (Bootstrap должен был его создать)", backdropCountAfterOpen)
	}
	if !strings.Contains(toastText, "Компонент добавлен") {
		t.Errorf("тост после отправки = %q, want содержит «Компонент добавлен»", toastText)
	}
	if backdropCountAfterSubmit != 0 {
		t.Errorf("backdrop после отправки формы (модалка должна закрыться) = %d, want 0 — см. htmx:afterSwap в app.js", backdropCountAfterSubmit)
	}
}

// TestChromeUI_AsOfSwitcher — E6.1: переключатель «На дату» в настоящем браузере ставит cookie и
// перезагружает страницу в режиме «только чтение» (баннер), «Вернуться к текущему» снимает его.
func TestChromeUI_AsOfSwitcher(t *testing.T) {
	setup(t)
	srv := newChromeServer(t)
	ctx := newChromeCtx(t)

	past := time.Now().UTC().Add(-48 * time.Hour).Format("2006-01-02T15:04")
	var bannerText, inputValue string
	var bannerAfterReset int
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/cis"),
		chromedp.WaitVisible(`#asof-input`, chromedp.ByQuery),
		chromedp.SetValue(`#asof-input`, past, chromedp.ByQuery),
		chromedp.Click(`#asof-apply`, chromedp.ByQuery),
		chromedp.WaitVisible(`#asof-banner`, chromedp.ByQuery),
		chromedp.Text(`#asof-banner`, &bannerText, chromedp.ByQuery),
		chromedp.Value(`#asof-input`, &inputValue, chromedp.ByQuery),
		chromedp.Click(`#asof-reset`, chromedp.ByQuery),
		chromedp.WaitVisible(`#asof-apply`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll('#asof-banner').length`, &bannerAfterReset),
	)
	if err != nil {
		t.Fatalf("chromedp: %v", err)
	}
	if !strings.Contains(bannerText, past[:10]) || !strings.Contains(bannerText, "только чтения") {
		t.Errorf("баннер = %q, want дата %s и «только чтения»", bannerText, past[:10])
	}
	if inputValue != past {
		t.Errorf("поле переключателя после применения = %q, want %q (момент сохраняется в cookie)", inputValue, past)
	}
	if bannerAfterReset != 0 {
		t.Errorf("после «Вернуться к текущему» баннеров = %d, want 0", bannerAfterReset)
	}
}
