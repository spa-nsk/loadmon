// anonymize - обезличивание xlsx-файлов.
//
// Что делает (детерминированно: одно и то же исходное значение всегда даёт
// одно и то же вымышленное, поэтому связи между листами и файлами сохраняются):
//   - ФИО                      -> "Иванов Иван Иванович" ("Иванов И.И." для инициалов)
//   - организации              -> вымышленные названия (форма ООО/АО/... сохраняется)
//   - адреса                   -> вымышленные города/улицы/районы, структура сохраняется
//   - серийные/инвентарные №   -> посимвольная замена с тем же форматом
//   - поля СЗ                  -> цифры заменяются на другие
//   - даты                     -> случайные в диапазоне 2011-01-01 .. 2026-10-20
//
// Тип колонки определяется по заголовку; кроме того, ФИО, организации
// с правовой формой и даты ищутся по шаблону в любых ячейках.
//
// Запуск:
//
//	go run ./tools/anonymize [-seed строка] [-o out.xlsx] [-report] in.xlsx [in2.xlsx ...]
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
)

type kind int

const (
	kText kind = iota // без спец. типа: только поиск шаблонов в тексте
	kFIO
	kOrg
	kParty // владелец/собственник: ФИО или организация - по содержимому
	kAddr
	kLoc
	kSerial
	kSZ
	kDate
)

var kindNames = map[kind]string{
	kText: "текст", kFIO: "ФИО", kOrg: "организация", kParty: "владелец",
	kAddr: "адрес", kLoc: "местоположение", kSerial: "серийный №", kSZ: "СЗ", kDate: "дата",
}

var (
	dateFrom = time.Date(2011, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo   = time.Date(2026, 10, 20, 23, 59, 59, 0, time.UTC)
	seed     = ""
)

const fakeFIO = "Иванов Иван Иванович"
const fakeInitials = "Иванов И.И."

// ---------------------------------------------------------------- классификация колонок

func normHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "ё", "е")
	return strings.Join(strings.Fields(s), " ")
}

func hasAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func classify(h string) kind {
	n := normHeader(h)
	switch {
	case n == "":
		return kText
	case n == "сз" || strings.HasPrefix(n, "сз ") || hasAny(n, "номера сз", "номер сз", "служебн"):
		return kSZ
	case n == "sn" || hasAny(n, "серийн", "serial", "s/n", "заводск", "инвентарн", "service tag", "сервис-тег", "сервисный тег"):
		return kSerial
	case hasAny(n, "дата", "date"):
		return kDate
	case hasAny(n, "фио", "ф.и.о", "фамили", "исполнител", "ответствен", "сотрудник", "автор",
		"администратор", "контактн", "пользовател", "руководител", "менеджер", "инженер",
		"согласовал", "утвердил", "создал", "изменил", "инициатор", "куратор"):
		return kFIO
	case hasAny(n, "владелец", "собственник", "балансодерж", "арендатор"):
		return kParty
	case hasAny(n, "организац", "заказчик", "контрагент", "поставщик", "подрядчик", "юрлиц", "юр. лиц", "компани"):
		return kParty
	case hasAny(n, "адрес", "address"):
		return kAddr
	case hasAny(n, "местоположен", "расположен", "площадк", "цод", "город", "локаци"):
		return kLoc
	}
	return kText
}

// ---------------------------------------------------------------- генераторы

func rngFor(k, val string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(seed + "|" + k + "|" + val))
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

func pick(r *rand.Rand, a []string) string { return a[r.Intn(len(a))] }

var (
	cityStems = []string{"Зареч", "Светлогор", "Новобор", "Липов", "Ясногор", "Тихомир", "Верхнекам", "Озёрн",
		"Сосновогор", "Красноборь", "Лугов", "Медвеж", "Солнечн", "Берёзов", "Каменск", "Дубров", "Ключев", "Рябин", "Звёздн", "Пригород"}
	citySuffixes = []string{"ск", "ово", "ино", "град", "ье", "ец", "горск", "реченск", "поле", "ский Посад", "ица"}
	adjStems     = []string{"Ясенев", "Лугов", "Ключев", "Рябинов", "Соснов", "Берёзов", "Луч", "Родников", "Заречн", "Тихоречн",
		"Липов", "Кедров", "Звёздн", "Лазурн", "Янтарн", "Мирн", "Садов", "Полев", "Речн", "Лесн", "Горн", "Солнечн", "Зелен", "Светл"}

	orgFirst = []string{"Вектор", "Альтаир", "Северный", "Орион", "Прогресс", "Горизонт", "Меридиан", "Спектр", "Эталон", "Квант",
		"Радуга", "Вершина", "Аргон", "Маяк", "Союз", "Фарватер", "Рубеж", "Импульс", "Полюс", "Зенит", "Атлант", "Лидер", "Старт", "Мираж"}
	orgSecond = []string{"Системы", "Технологии", "Сервис", "Инвест", "Трейд", "Строй", "Телеком", "Логистика", "Проект", "Софт",
		"Энерго", "Групп", "Холдинг", "Консалт", "Индустрия", "Финанс"}
)

func fakeCity(val string) string {
	r := rngFor("city", strings.ToLower(val))
	return pick(r, cityStems) + pick(r, citySuffixes)
}

// fakeAdj - вымышленное прилагательное; fem - женский род (улица, область) или мужской.
func fakeAdj(val string, fem bool) string {
	r := rngFor("adj", strings.ToLower(val))
	if fem {
		return pick(r, adjStems) + "ая"
	}
	return pick(r, adjStems) + "ый"
}

func fakeOrgName(val string) string {
	r := rngFor("org", strings.ToLower(strings.TrimSpace(val)))
	return pick(r, orgFirst) + " " + pick(r, orgSecond)
}

// differentRune возвращает символ того же класса, но не равный исходному.
func differentRune(r *rand.Rand, c rune) rune {
	switch {
	case c >= '0' && c <= '9':
		return '0' + (c-'0'+rune(1+r.Intn(9)))%10
	case c >= 'a' && c <= 'z':
		return 'a' + (c-'a'+rune(1+r.Intn(25)))%26
	case c >= 'A' && c <= 'Z':
		return 'A' + (c-'A'+rune(1+r.Intn(25)))%26
	case c >= 'а' && c <= 'я':
		return 'а' + (c-'а'+rune(1+r.Intn(31)))%32
	case c >= 'А' && c <= 'Я':
		return 'А' + (c-'А'+rune(1+r.Intn(31)))%32
	}
	return c
}

// scramble - посимвольная замена с сохранением формата (цифра->цифра, буква->буква того же регистра).
func scramble(s string, digitsOnly bool) string {
	r := rngFor("scr", s)
	var b strings.Builder
	for _, c := range s {
		if digitsOnly && !(c >= '0' && c <= '9') {
			b.WriteRune(c)
			continue
		}
		b.WriteRune(differentRune(r, c))
	}
	return b.String()
}

func anonSZ(s string) string {
	return scramble(s, false)
}

func anonSerial(s string) string { return scramble(s, false) }

// ---------------------------------------------------------------- даты

var (
	reDateDot = regexp.MustCompile(`(\d{1,2})\.(\d{1,2})\.(\d{4})(?:([ T])(\d{1,2}):(\d{2})(?::(\d{2}))?)?`)
	reDateISO = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})(?:([ T])(\d{1,2}):(\d{2})(?::(\d{2}))?)?`)
	reDateSl  = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`)
)

func fakeTime(orig string) time.Time {
	r := rngFor("date", orig)
	span := dateTo.Unix() - dateFrom.Unix()
	return time.Unix(dateFrom.Unix()+r.Int63n(span), 0).UTC()
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// replaceDates меняет даты в тексте на вымышленные, сохраняя формат записи.
func replaceDates(s string, n *int) string {
	do := func(re *regexp.Regexp, yi, mi, di int, sep string, s string) string {
		var out strings.Builder
		last := 0
		for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
			a, b := m[0], m[1]
			if a > 0 && (isDigit(s[a-1]) || s[a-1] == '.' || s[a-1] == '-' || s[a-1] == '/') {
				continue
			}
			if b < len(s) && (isDigit(s[b]) || (s[b] == '.' && b+1 < len(s) && isDigit(s[b+1]))) {
				continue
			}
			g := func(i int) string {
				if m[2*i] < 0 {
					return ""
				}
				return s[m[2*i]:m[2*i+1]]
			}
			y, _ := strconv.Atoi(g(yi))
			mo, _ := strconv.Atoi(g(mi))
			d, _ := strconv.Atoi(g(di))
			if y < 1900 || y > 2100 || mo < 1 || mo > 12 || d < 1 || d > 31 {
				continue
			}
			t := fakeTime(s[a:b])
			dd, mm := strconv.Itoa(t.Day()), strconv.Itoa(int(t.Month()))
			if len(g(di)) == 2 {
				dd = fmt.Sprintf("%02d", t.Day())
			}
			if len(g(mi)) == 2 {
				mm = fmt.Sprintf("%02d", int(t.Month()))
			}
			var res string
			if sep == "-" {
				res = fmt.Sprintf("%04d-%s-%s", t.Year(), mm, dd)
			} else {
				res = dd + sep + mm + sep + strconv.Itoa(t.Year())
			}
			if re != reDateSl && g(4) != "" { // есть время
				res += g(4) + fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
				if g(7) != "" {
					res += fmt.Sprintf(":%02d", t.Second())
				}
			}
			out.WriteString(s[last:a])
			out.WriteString(res)
			last = b
			*n++
		}
		out.WriteString(s[last:])
		return out.String()
	}
	s = do(reDateDot, 3, 2, 1, ".", s)
	s = do(reDateISO, 1, 2, 3, "-", s)
	s = do(reDateSl, 3, 2, 1, "/", s)
	return s
}

const excelEpochOffset = 25569 // дней между 1899-12-30 и 1970-01-01

func fakeSerial(orig string, withTime bool) float64 {
	t := fakeTime(orig)
	days := float64(t.Unix()/86400 + excelEpochOffset)
	if withTime {
		days += float64(t.Unix()%86400) / 86400
	}
	return days
}

// ---------------------------------------------------------------- ФИО / организации

var (
	reFIO      = regexp.MustCompile(`[А-ЯЁ][а-яё]+(?:-[А-ЯЁ][а-яё]+)?\s+[А-ЯЁ][а-яё]+\s+[А-ЯЁ][а-яё]+(?:вич|вна|ична|ьич|оглы|кызы|кзы)`)
	reInit     = regexp.MustCompile(`[А-ЯЁ][а-яё]+(?:-[А-ЯЁ][а-яё]+)?\s+[А-ЯЁ]\.\s?[А-ЯЁ]\.`)
	reInitRev  = regexp.MustCompile(`[А-ЯЁ]\.\s?[А-ЯЁ]\.\s?[А-ЯЁ][а-яё]+(?:-[А-ЯЁ][а-яё]+)?`)
	reOrgQuote = regexp.MustCompile(`(ООО|ОАО|ЗАО|ПАО|НАО|АО|ФГУП|ГУП|МУП|ФГБУ|ФГАУ|ГБУ|ГАУ|МБУ|АНО|НКО|ИП)\s*[«"“„]([^»"”“]{1,80})[»"”“]`)
	reOrgForm  = regexp.MustCompile(`^(ООО|ОАО|ЗАО|ПАО|НАО|АО|ФГУП|ГУП|МУП|ФГБУ|ФГАУ|ГБУ|ГАУ|МБУ|АНО|НКО)\s*`)
	reQuotes   = regexp.MustCompile(`^[«"“„]?(.*?)[»"”“]?$`)
	reMulti    = regexp.MustCompile(`\s*[;\n]\s*`)
)

func replaceNames(s string, n *int) string {
	cnt := func(f func(string) string) func(string) string {
		return func(m string) string { *n++; return f(m) }
	}
	s = reFIO.ReplaceAllStringFunc(s, cnt(func(string) string { return fakeFIO }))
	s = reInit.ReplaceAllStringFunc(s, cnt(func(string) string { return fakeInitials }))
	s = reInitRev.ReplaceAllStringFunc(s, cnt(func(string) string { return "И.И. Иванов" }))
	s = reOrgQuote.ReplaceAllStringFunc(s, cnt(func(m string) string {
		sm := reOrgQuote.FindStringSubmatch(m)
		return sm[1] + " «" + fakeOrgName(sm[2]) + "»"
	}))
	return s
}

// splitKeep разбивает по разделителям ; и перевод строки, сохраняя сами разделители.
func mapParts(s string, f func(string) string) string {
	seps := reMulti.FindAllString(s, -1)
	parts := reMulti.Split(s, -1)
	var b strings.Builder
	for i, p := range parts {
		if strings.TrimSpace(p) != "" {
			p = f(strings.TrimSpace(p))
		}
		b.WriteString(p)
		if i < len(seps) {
			b.WriteString(seps[i])
		}
	}
	return b.String()
}

func anonFIO(s string) string {
	return mapParts(s, func(p string) string {
		if reInit.MatchString(p) || reInitRev.MatchString(p) {
			return fakeInitials
		}
		return fakeFIO
	})
}

func anonOrg(s string) string {
	return mapParts(s, func(p string) string {
		form := ""
		if m := reOrgForm.FindString(p); m != "" {
			form = strings.TrimSpace(m)
			p = strings.TrimSpace(p[len(m):])
		}
		name := reQuotes.FindStringSubmatch(p)[1]
		if form == "" {
			return fakeOrgName(name)
		}
		return form + " «" + fakeOrgName(name) + "»"
	})
}

func anonParty(s string) string {
	return mapParts(s, func(p string) string {
		if reFIO.MatchString(p) || reInit.MatchString(p) || reInitRev.MatchString(p) {
			return anonFIO(p)
		}
		return anonOrg(p)
	})
}

// ---------------------------------------------------------------- адреса

var (
	reIndex    = regexp.MustCompile(`^\d{6}$`)
	reHouse    = regexp.MustCompile(`(?i)^(д|дом|корп|корпус|к|стр|строение|кв|квартира|оф|офис|пом|помещение|лит|литера|эт|этаж|ком|комната|подъезд)\.?\s*\d`)
	reSettle   = regexp.MustCompile(`(?i)^(г|город|п|пос|поселок|посёлок|пгт|рп|с|село|д|дер|деревня|ст|х|хутор|мкр|микрорайон|нп)\.?\s+\S`)
	reRegionKW = regexp.MustCompile(`(?i)^(обл|обл\.|область|край|респ|респ\.|республика|округ|р-н|район)$`)
	reStreetKW = regexp.MustCompile(`(?i)^(ул|ул\.|улица|пр-т|пр-т\.|просп|просп\.|проспект|пер|пер\.|переулок|пр-д|пр-д\.|проезд|б-р|бульвар|наб|наб\.|набережная|пл|пл\.|площадь|ш|ш\.|шоссе|тракт|аллея|туп|тупик)$`)
	reCountry  = regexp.MustCompile(`(?i)^(россия|рф|российская федерация)$`)
	reCyr      = regexp.MustCompile(`[А-Яа-яЁё]`)
	reDigitRun = regexp.MustCompile(`\d+`)
)

// rePlaceholder - заглушки вида "- n/a -", которые не нужно подменять.
var rePlaceholder = regexp.MustCompile(`(?i)^[\s\-–—]*(n/a|нет|н/д|не указан[оа]?)[\s\-–—]*$`)

func isCyrWord(t string) bool {
	return reCyr.MatchString(t) && !strings.ContainsAny(t, "0123456789")
}

func randNumber(orig string) string {
	r := rngFor("num", orig)
	out := []byte(orig)
	for i := range out {
		out[i] = byte('0' + r.Intn(10))
	}
	if len(out) > 1 && out[0] == '0' {
		out[0] = '1'
	}
	if string(out) == orig {
		out[len(out)-1] = '0' + (out[len(out)-1]-'0'+1)%10
	}
	return string(out)
}

func randDigits(s string) string {
	return reDigitRun.ReplaceAllStringFunc(s, randNumber)
}

// rewriteTokens: служебные слова (kw) остаются, цифры меняются, первое
// кириллическое слово заменяется gen(), остальные кириллические слова отбрасываются.
func rewriteTokens(t string, kw *regexp.Regexp, gen func(string) string) string {
	toks := strings.Fields(t)
	var out []string
	replaced := false
	var name []string
	for _, tk := range toks {
		switch {
		case kw != nil && kw.MatchString(tk):
			out = append(out, tk)
		case strings.ContainsAny(tk, "0123456789"):
			out = append(out, randDigits(tk))
		case isCyrWord(tk):
			name = append(name, tk)
			if !replaced {
				replaced = true
				out = append(out, "\x00")
			}
		default:
			out = append(out, tk)
		}
	}
	if replaced {
		g := gen(strings.Join(name, " "))
		for i, o := range out {
			if o == "\x00" {
				out[i] = g
			}
		}
	}
	return strings.Join(out, " ")
}

func anonAddrPart(t string) string {
	first := strings.Fields(t)
	switch {
	case reIndex.MatchString(t):
		return randNumber(t)
	case reHouse.MatchString(t):
		return randDigits(t)
	case reCountry.MatchString(t) || !reCyr.MatchString(t):
		if _, err := strconv.Atoi(t); err == nil {
			return randNumber(t)
		}
		return t
	case reSettle.MatchString(t):
		return rewriteTokens(t, reSettle2, fakeCity)
	}
	// район/область/край/республика
	for _, tk := range first {
		if reRegionKW.MatchString(tk) {
			low := strings.ToLower(tk)
			fem := strings.HasPrefix(low, "обл") || strings.HasPrefix(low, "респ")
			return rewriteTokens(t, reRegionKW, func(v string) string { return fakeAdj(v, fem) })
		}
	}
	for _, tk := range first {
		if reStreetKW.MatchString(tk) {
			low := strings.ToLower(tk)
			fem := !(strings.HasPrefix(low, "пр-т") || strings.HasPrefix(low, "просп") || strings.HasPrefix(low, "пер") ||
				strings.HasPrefix(low, "пр-д") || strings.HasPrefix(low, "проезд") || strings.HasPrefix(low, "б-р") ||
				strings.HasPrefix(low, "бульвар") || strings.HasPrefix(low, "ш") || strings.HasPrefix(low, "туп"))
			return rewriteTokens(t, reStreetKW, func(v string) string { return fakeAdj(v, fem) })
		}
	}
	if !strings.ContainsAny(t, "0123456789") {
		return fakeCity(t) // "Мухосранск" без префикса
	}
	return rewriteTokens(t, nil, fakeCity)
}

// reSettle2 - только само слово-префикс населённого пункта (для rewriteTokens).
var reSettle2 = regexp.MustCompile(`(?i)^(г|г\.|город|п|п\.|пос|пос\.|поселок|посёлок|пгт|пгт\.|рп|рп\.|с|с\.|д|д\.|дер|дер\.|деревня|ст|ст\.|х|х\.|хутор|мкр|мкр\.|микрорайон|нп)$`)

func anonAddress(s string) string {
	if strings.HasPrefix(strings.TrimSpace(s), "!") {
		return s
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		lead := p[:len(p)-len(strings.TrimLeft(p, " \t"))]
		parts[i] = lead + anonAddrPart(t)
	}
	return strings.Join(parts, ",")
}

// anonLocation: "Город, ЦОД (улица) | 6 | 6.1 (624)". Меняются только короткие
// текстовые фрагменты (город, название площадки, улица в скобках); коды остаются.
func anonLocation(s string) string {
	if strings.HasPrefix(strings.TrimSpace(s), "!") {
		return s
	}
	reParen := regexp.MustCompile(`\(([^)]*)\)`)
	segs := strings.Split(s, "|")
	for si, seg := range segs {
		seg = reParen.ReplaceAllStringFunc(seg, func(m string) string {
			in := m[1 : len(m)-1]
			if !reCyr.MatchString(in) {
				return m
			}
			return "(" + rewriteLocWords(in) + ")"
		})
		parts := strings.Split(seg, ",")
		for i, p := range parts {
			t := strings.TrimSpace(p)
			if t == "" || !reCyr.MatchString(t) {
				continue
			}
			lead := p[:len(p)-len(strings.TrimLeft(p, " \t"))]
			// скобки уже обработаны: заменяем только текст вне скобок
			outer := reParen.ReplaceAllString(t, "\x01")
			if strings.TrimSpace(strings.ReplaceAll(outer, "\x01", "")) == "" {
				continue
			}
			words := strings.Fields(outer)
			if len(words) > 4 || len(t) > 50 || reCountry.MatchString(t) {
				continue
			}
			var res []string
			done := false
			for _, w := range words {
				switch {
				case strings.Contains(w, "\x01"):
					res = append(res, w)
				case isCyrWord(w) && w == strings.ToUpper(w) && len([]rune(w)) <= 4: // аббревиатура КЦ, ЦОД
					res = append(res, w)
				case isCyrWord(w) && unicode.IsUpper([]rune(w)[0]):
					if !done {
						res = append(res, fakeCity(t))
						done = true
					}
				default:
					res = append(res, w)
				}
			}
			joined := strings.Join(res, " ")
			// вернуть скобочные группы на место
			orig := reParen.FindAllString(t, -1)
			for _, g := range orig {
				joined = strings.Replace(joined, "\x01", g, 1)
			}
			trail := t2trail(p)
			parts[i] = lead + joined + trail
		}
		segs[si] = strings.Join(parts, ",")
	}
	return strings.Join(segs, "|")
}

func t2trail(p string) string { return p[len(strings.TrimRight(p, " 	")):] }

func rewriteLocWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if isCyrWord(w) && unicode.IsUpper([]rune(w)[0]) && !(w == strings.ToUpper(w) && len([]rune(w)) <= 4) {
			words[i] = fakeCity(w)
		}
	}
	return strings.Join(words, " ")
}

// ---------------------------------------------------------------- обработка файла

type stats struct{ byKind map[string]int }

func isDateStyle(f *excelize.File, sheet, cell string, cache map[int]bool) bool {
	id, err := f.GetCellStyle(sheet, cell)
	if err != nil || id == 0 {
		return false
	}
	if v, ok := cache[id]; ok {
		return v
	}
	res := false
	if st, err := f.GetStyle(id); err == nil && st != nil {
		if (st.NumFmt >= 14 && st.NumFmt <= 22) || (st.NumFmt >= 45 && st.NumFmt <= 47) {
			res = true
		} else if st.CustomNumFmt != nil {
			c := strings.ToLower(*st.CustomNumFmt)
			c = regexp.MustCompile(`"[^"]*"|\[[^\]]*\]`).ReplaceAllString(c, "")
			res = strings.ContainsAny(c, "dmy") && !strings.Contains(c, "0.0")
		}
	}
	cache[id] = res
	return res
}

func processFile(in, out string, reportOnly bool) error {
	f, err := excelize.OpenFile(in)
	if err != nil {
		return err
	}
	defer f.Close()
	total := map[string]int{}
	styleCache := map[int]bool{}

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
		if err != nil {
			return err
		}
		// строка заголовка - первая строка с >=3 непустыми ячейками среди первых 15
		hdr := -1
		for i := 0; i < len(rows) && i < 15; i++ {
			c := 0
			for _, v := range rows[i] {
				if strings.TrimSpace(v) != "" {
					c++
				}
			}
			if c >= 3 {
				hdr = i
				break
			}
		}
		var kinds []kind
		if hdr >= 0 {
			for _, h := range rows[hdr] {
				kinds = append(kinds, classify(h))
			}
			fmt.Printf("  лист %q, заголовок в строке %d:\n", sheet, hdr+1)
			for i, h := range rows[hdr] {
				if kinds[i] != kText {
					fmt.Printf("    колонка %d %q -> %s\n", i+1, h, kindNames[kinds[i]])
				}
			}
		}
		for ri, row := range rows {
			for ci, raw := range row {
				if strings.TrimSpace(raw) == "" || ri == hdr {
					continue
				}
				k := kText
				if hdr >= 0 && ri > hdr && ci < len(kinds) {
					k = kinds[ci]
				}
				cell, _ := excelize.CoordinatesToCellName(ci+1, ri+1)
				var newVal any = raw
				changed := false

				// числовые даты (серийный номер Excel)
				if fv, perr := strconv.ParseFloat(raw, 64); perr == nil {
					if fv > 30000 && fv < 60000 && (k == kDate || isDateStyle(f, sheet, cell, styleCache)) {
						newVal = fakeSerial(raw+cell, fv != float64(int64(fv)))
						changed = true
						total["дата"]++
					} else if k == kSerial || k == kSZ {
						s := anonSerial(raw)
						if k == kSZ {
							s = anonSZ(raw)
						}
						newVal, changed = s, true
						total[kindNames[k]]++
					}
				} else {
					s := raw
					if rePlaceholder.MatchString(raw) {
						k = kText
					}
					switch k {
					case kFIO:
						s = anonFIO(raw)
					case kOrg:
						s = anonOrg(raw)
					case kParty:
						s = anonParty(raw)
					case kAddr:
						s = anonAddress(raw)
					case kLoc:
						s = anonLocation(raw)
					case kSerial:
						s = anonSerial(raw)
					case kSZ:
						s = anonSZ(raw)
					}
					if s != raw {
						total[kindNames[k]]++
					}
					d := 0
					s = replaceDates(s, &d)
					total["дата"] += d
					nn := 0
					if k != kSerial && k != kSZ {
						s = replaceNames(s, &nn)
						total["ФИО/орг. в тексте"] += nn
					}
					if s != raw {
						newVal, changed = s, true
					}
				}
				if changed && !reportOnly {
					if fm, _ := f.GetCellFormula(sheet, cell); fm != "" {
						continue
					}
					if err := f.SetCellValue(sheet, cell, newVal); err != nil {
						return err
					}
				}
			}
		}
	}
	fmt.Println("  заменено:", total)
	if reportOnly {
		return nil
	}
	_ = f.SetDocProps(&excelize.DocProperties{Creator: "user", LastModifiedBy: "user"})
	return f.SaveAs(out)
}

func main() {
	o := flag.String("o", "", "выходной файл (только для одного входного); по умолчанию <имя>_anon.xlsx")
	report := flag.Bool("report", false, "только показать, что будет заменено, ничего не записывать")
	flag.StringVar(&seed, "seed", "loadmon", "соль детерминированной подстановки (другая соль - другие вымышленные значения)")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "использование: anonymize [-seed s] [-o out.xlsx] [-report] in.xlsx [...]")
		os.Exit(2)
	}
	if *o != "" && flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "-o допустим только с одним входным файлом")
		os.Exit(2)
	}
	for _, in := range flag.Args() {
		out := *o
		if out == "" {
			ext := filepath.Ext(in)
			out = strings.TrimSuffix(in, ext) + "_anon.xlsx"
		}
		fmt.Println("Файл:", in)
		if err := processFile(in, out, *report); err != nil {
			fmt.Fprintln(os.Stderr, "ошибка:", err)
			os.Exit(1)
		}
		if !*report {
			fmt.Println("  записан:", out)
		}
	}
}
