package main

import (
	"strings"
	"testing"
)

func TestAnonymize(t *testing.T) {
	n := 0
	for _, in := range []string{"12.03.2019", "2020-05-07 10:20:30", "ver 10.1.2024.5"} {
		t.Log(in, "->", replaceDates(in, &n))
	}
	for _, in := range []string{
		"123456, Свердловская обл., г. Екатеринбург, ул. Ленина, д. 15, оф. 301",
		"Мухосранск, пр-т Мира 7",
	} {
		out := anonAddress(in)
		t.Log(in, "->", out)
		if strings.Contains(out, "Ленина") || strings.Contains(out, "Екатеринбург") {
			t.Error("утечка:", out)
		}
	}
	t.Log(anonOrg(`ООО «Ромашка»; Рога и копыта`), anonSZ("сз 88888"), anonFIO("Петров Пётр Петрович; Сидоров А.В."))
	t.Log(replaceNames(`Договор с АО "Газпром" подписал Сидоров Сидор Сидорович`, &n))
}

func TestPhrases(t *testing.T) {
	got := replacePhrases("ЦОД Ростелеком, цод, Мегафон")
	if got != "БЦ Шаражка, бц, Домофон" {
		t.Error(got)
	}
}
