package naming

import "fmt"

// Status — итог проверки имени КЕ (intent §5.7).
type Status string

const (
	// StatusCompliant — имя построено строго по активным правилам профиля CMDB-NC-2026.
	StatusCompliant Status = "compliant"
	// StatusLegacy — имя распознано правилами Guide 2019 (реализуется в E1.4; до тех пор
	// Validate для таких имён возвращает StatusNoncompliant с пометкой "нет активного правила").
	StatusLegacy Status = "legacy"
	// StatusNoncompliant — имя не соответствует ни активным правилам, ни legacy-правилам Guide.
	StatusNoncompliant Status = "noncompliant"
	// StatusException — имя присвоено вручную в обход генератора, с обязательной причиной (UI, E3.5).
	StatusException Status = "exception"
)

// Result — результат Validate: статус, причина (для noncompliant) и предложенное имя (если есть).
type Result struct {
	Status     Status
	Reason     string
	Suggestion string
}

// legacyServerSeqDigits — задокументированные в Guide (§3.1.1/§3.1.4) ширины номера для серверов
// и ESX-хостов, отличные от 5-значной схемы CMDB-NC-2026: 3 цифры — схема до 1 мая 2004 года,
// 4 цифры — диапазон GI-H (6000-9999). Ширина у остальных классов (например CL) Guide не
// определял отдельно, поэтому её несовпадение с профилем 2026 — просто ошибка ввода, а не legacy.
var legacyServerSeqDigits = map[int]bool{3: true, 4: true}

// Validate проверяет основное имя хоста (<SITE>-<CLS>-<SEQ>) против активных правил профиля
// CMDB-NC-2026, а при несовпадении — против правил Guide 2019 (профиль GUIDE-2019, intent §5.6):
// перебор идёт "профиль 2026 → legacy Guide" (E1.5). Разбор суффиксных производных имён (BMC,
// порты, зоны и т.п.) выполняется отдельными функциями Parse*/Build* в этом пакете — Validate
// покрывает основной шаблон и (через matchLegacy) все схемы Guide, включая суффиксные (E1.4).
func Validate(reg *Registry, name string) Result {
	parsed, err := ParseHostName(name)
	if err != nil {
		if m, ok := matchLegacy(name); ok {
			return Result{Status: StatusLegacy, Reason: m.Note}
		}
		return Result{Status: StatusNoncompliant, Reason: err.Error()}
	}

	d, ok := reg.Lookup(parsed.Cls)
	if !ok {
		if m, ok := matchLegacy(name); ok {
			return Result{Status: StatusLegacy, Reason: m.Note}
		}
		return Result{
			Status: StatusNoncompliant,
			Reason: fmt.Sprintf("код класса %q не известен ни профилю CMDB-NC-2026, ни правилам Guide 2019", parsed.Cls),
		}
	}

	if !d.Active {
		reason := fmt.Sprintf("код класса %q — устаревшее правило Guide 2019 (deprecated/retired)", parsed.Cls)
		suggestion := ""
		if d.ReplacedBy != "" {
			reason += fmt.Sprintf(", в профиле CMDB-NC-2026 заменён на %q", d.ReplacedBy)
			if built, buildErr := BuildHostName(reg, parsed.Site, d.ReplacedBy, parsed.Seq); buildErr == nil {
				suggestion = built
			}
		}
		return Result{Status: StatusLegacy, Reason: reason, Suggestion: suggestion}
	}

	// Площадка встречается только в иллюстративных примерах самого Guide — форма имени совпадает
	// с профилем 2026 (совпадение конструкции неслучайно: §5.3 сознательно наследует основной
	// шаблон Guide), но принадлежит профилю GUIDE-2019, а не CMDB-NC-2026.
	if isLegacyGuideSite(parsed.Site) {
		return Result{
			Status: StatusLegacy,
			Reason: "площадка встречается в примерах Guide 2019 (GUIDE-2019), а не в профиле CMDB-NC-2026",
		}
	}

	if parsed.SeqDigits != d.SeqLength {
		suggestion, _ := BuildHostName(reg, parsed.Site, parsed.Cls, parsed.Seq)
		if (parsed.Cls == "S" || parsed.Cls == "H") && legacyServerSeqDigits[parsed.SeqDigits] {
			return Result{
				Status:     StatusLegacy,
				Reason:     fmt.Sprintf("номер из %d цифр — схема Guide 2019 (до 01.05.2004 — 3 цифры, диапазон GI-H — 4 цифры) вместо 5-значной схемы CMDB-NC-2026", parsed.SeqDigits),
				Suggestion: suggestion,
			}
		}
		return Result{
			Status:     StatusNoncompliant,
			Reason:     fmt.Sprintf("номер класса %q должен состоять из %d цифр, а не %d", parsed.Cls, d.SeqLength, parsed.SeqDigits),
			Suggestion: suggestion,
		}
	}

	if len(name) > maxHostNameLen {
		return Result{
			Status: StatusNoncompliant,
			Reason: fmt.Sprintf("имя длиннее лимита %d символов для хоста", maxHostNameLen),
		}
	}

	return Result{Status: StatusCompliant}
}
