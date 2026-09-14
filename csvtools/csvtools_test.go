package csvtools

import (
	"net/url"
	"testing"
	"time"
)

func TestSubstr(t *testing.T) {
	cases := []struct {
		name          string
		s             string
		start, length int
		want          string
	}{
		{"normal", "hello world", 0, 5, "hello"},
		{"middle", "hello world", 6, 5, "world"},
		{"negative start clamped to 0", "hello", -3, 3, "hel"},
		{"length beyond end clamped", "hello", 2, 100, "llo"},
		{"start beyond length wraps via modulo", "hello", 7, 2, "ll"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Substr(c.s, c.start, c.length)
			if got != c.want {
				t.Errorf("Substr(%q, %d, %d) = %q, want %q", c.s, c.start, c.length, got, c.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	if !Compare("1", 1) {
		t.Error(`Compare("1", 1) should be true (both stringify to "1")`)
	}
	if !Compare(" abc ", "abc") {
		t.Error(`Compare(" abc ", "abc") should be true (whitespace trimmed)`)
	}
	if Compare("abc", "def") {
		t.Error(`Compare("abc", "def") should be false`)
	}
	if !CompareNot("abc", "def") {
		t.Error(`CompareNot("abc", "def") should be true`)
	}
	if CompareNot("abc", "abc") {
		t.Error(`CompareNot("abc", "abc") should be false`)
	}
}

func TestNotNil(t *testing.T) {
	if NotNil(nil) {
		t.Error("NotNil(nil) should be false")
	}
	if !NotNil("something") {
		t.Error(`NotNil("something") should be true`)
	}
}

func TestDateParseAndDate(t *testing.T) {
	tm, err := DateParse("2026-08-10", "Y-m-d")
	if err != nil {
		t.Fatalf("DateParse returned error: %v", err)
	}
	if tm.Year() != 2026 || tm.Month() != time.August || tm.Day() != 10 {
		t.Errorf("DateParse got %v, want 2026-08-10", tm)
	}

	got := Date(tm, "d.m.Y")
	want := "10.08.2026"
	if got != want {
		t.Errorf("Date() = %q, want %q", got, want)
	}
}

func TestDateFormat(t *testing.T) {
	tm := time.Date(2026, time.August, 10, 15, 4, 5, 0, time.UTC)
	got := DateFormat(tm, "2006-01-02")
	if got != "2026-08-10" {
		t.Errorf("DateFormat() = %q, want %q", got, "2026-08-10")
	}
}

type parseFormTestStruct struct {
	Name    string    `csv:"name"`
	Age     int       `csv:"age"`
	Active  bool      `csv:"active"`
	Rate    float64   `csv:"rate"`
	Ignored string    `csv:"-"`
	When    time.Time `csv:"when"`
	Tags    []string  `csv:"tags"`
	Nums    []int     `csv:"nums"`
}

func TestParseForm(t *testing.T) {
	form := url.Values{
		"name":    {"server1"},
		"age":     {"7"},
		"active":  {"on"},
		"rate":    {"3.5"},
		"Ignored": {"should not be set"},
		"when":    {"2026-08-10"},
		"tags":    {"a", "b"},
		"nums":    {"1", "2", "3"},
	}

	var s parseFormTestStruct
	if err := ParseForm(form, &s); err != nil {
		t.Fatalf("ParseForm returned error: %v", err)
	}

	if s.Name != "server1" {
		t.Errorf("Name = %q, want %q", s.Name, "server1")
	}
	if s.Age != 7 {
		t.Errorf("Age = %d, want 7", s.Age)
	}
	if !s.Active {
		t.Errorf("Active = false, want true (\"on\" should parse as true)")
	}
	if s.Rate != 3.5 {
		t.Errorf("Rate = %v, want 3.5", s.Rate)
	}
	if s.Ignored != "" {
		t.Errorf("Ignored = %q, want empty (csv:\"-\" fields must not be set)", s.Ignored)
	}
	if s.When.Year() != 2026 || s.When.Month() != time.August || s.When.Day() != 10 {
		t.Errorf("When = %v, want 2026-08-10", s.When)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "a" || s.Tags[1] != "b" {
		t.Errorf("Tags = %v, want [a b]", s.Tags)
	}
	if len(s.Nums) != 3 || s.Nums[0] != 1 || s.Nums[1] != 2 || s.Nums[2] != 3 {
		t.Errorf("Nums = %v, want [1 2 3]", s.Nums)
	}
}

func TestParseFormRequiresStructPointer(t *testing.T) {
	var s parseFormTestStruct
	if err := ParseForm(url.Values{}, s); err == nil {
		t.Error("ParseForm should return an error when given a non-pointer struct")
	}
}

type csvHeaderTestStruct struct {
	Id   int64  `csv:"Идентификатор"`
	Name string `csv:"Наименование"`
}

func TestGetCSVHeaderAndData(t *testing.T) {
	rec := csvHeaderTestStruct{Id: 42, Name: "test"}

	header, err := GetCSVHeader(&rec)
	if err != nil {
		t.Fatalf("GetCSVHeader returned error: %v", err)
	}
	wantHeader := []string{"Идентификатор", "Наименование"}
	if len(header) != len(wantHeader) || header[0] != wantHeader[0] || header[1] != wantHeader[1] {
		t.Errorf("GetCSVHeader() = %v, want %v", header, wantHeader)
	}

	data, err := GetCSVData(&rec)
	if err != nil {
		t.Fatalf("GetCSVData returned error: %v", err)
	}
	if len(data) != 2 || data[0] != "42" || data[1] != "test" {
		t.Errorf("GetCSVData() = %v, want [42 test]", data)
	}
}
