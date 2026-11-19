package locality

import "testing"

func isCity(s string) bool { return s == "Саяногорск" || s == "Екатеринбург" }

func TestParseLocation(t *testing.T) {
	l := ParseLocation("Саяногорск, ЦОД РТК, Менделеева 1 | Colocation | Зал 205 | Шкаф 2.4.2 | Unit 13", isCity)
	if l.Locality != "Саяногорск" || l.Street != "Менделеева" || l.House != "1" || l.Description != "ЦОД РТК" ||
		l.Cabinet != "2.4.2" || l.Pos == nil || l.Pos.Start != 13 || l.Pos.End != 13 || l.Room != "Colocation, Зал 205" {
		t.Fatalf("%+v", l)
	}
	l = ParseLocation("Сеть (Д), Екатеринбург, ул. Максима Горького, д.1 | 1 | Серверная", isCity)
	if l.Description != "Сеть (Д)" || l.Locality != "Екатеринбург" || l.Street != "ул. Максима Горького" || l.House != "д.1" || l.Floor != "1" || l.Room != "Серверная" {
		t.Fatalf("%+v", l)
	}
}

func TestParsePlacement(t *testing.T) {
	p := ParsePlacement("ЦОД Домофон / 3.11.0 | 20-21")
	if p.AltName != "ЦОД Домофон" || p.Cabinet != "3.11.0" || p.Pos.Start != 20 || p.Pos.End != 21 {
		t.Fatalf("%+v", p)
	}
	p = ParsePlacement("6.2-A-01 | 9-9")
	if p.AltName != "" || p.Cabinet != "6.2-A-01" || p.Pos.String() != "9" {
		t.Fatalf("%+v", p)
	}
	p = ParsePlacement("6.1-B-01 | 45-45")
	if p.AltName != "" || p.Cabinet != "6.1-B-01" || p.Pos == nil || p.Pos.Start != 45 || p.Pos.End != 45 {
		t.Fatalf("%+v", p)
	}
}
