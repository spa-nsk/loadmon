// E7.4 через настоящий роутер и шаблоны: страница /quality рендерится и показывает свои разделы.
package ui_test

import (
	"testing"
)

func TestQuality_PageRenders(t *testing.T) {
	setup(t)
	w := send("GET", "/quality", nil, false)
	if w.Code != 200 {
		t.Fatalf("GET /quality: status=%d body=%.500s", w.Code, w.Body.String())
	}
	mustContain(t, "дашборд качества данных", w.Body.String(),
		"Качество данных", "Полнота обязательных атрибутов", "Соответствие профилю именования",
		"Устаревание", "Открытые конфликты импорта", "Динамика")
}
