package naming

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const maxHostNameLen = 15 // лимит NetBIOS/AD для имени компьютера (intent §5.1 P3)

var siteRe = regexp.MustCompile(`^[A-Z0-9]{1,6}$`)

// BuildHostName строит основное имя КЕ по шаблону <SITE>-<CLS>-<SEQ> (intent §5.3).
// site — уже сжатый код площадки (location_code_c, без дефисов, ≤6 симв., напр. "OVBDC1").
// Возвращает каноническое имя в верхнем регистре; для DNS используйте DNSName().
func BuildHostName(reg *Registry, site, cls string, seq int64) (string, error) {
	site = strings.ToUpper(site)
	cls = strings.ToUpper(cls)

	if !siteRe.MatchString(site) {
		return "", fmt.Errorf("naming: недопустимый SITE %q: ожидается 1-6 символов A-Z/0-9", site)
	}

	d, ok := reg.Lookup(cls)
	if !ok {
		return "", fmt.Errorf("naming: неизвестный код класса %q", cls)
	}
	if !d.Active {
		if d.ReplacedBy != "" {
			return "", fmt.Errorf("naming: код класса %q устарел (deprecated/retired) и не выдаётся новым КЕ, используйте %q", cls, d.ReplacedBy)
		}
		return "", fmt.Errorf("naming: код класса %q устарел (deprecated/retired) и не выдаётся новым КЕ", cls)
	}

	if seq < 0 {
		return "", fmt.Errorf("naming: номер %d не может быть отрицательным", seq)
	}
	maxSeq := int64(1)
	for i := 0; i < d.SeqLength; i++ {
		maxSeq *= 10
	}
	if seq >= maxSeq {
		return "", fmt.Errorf("naming: номер %d не укладывается в %d цифр для класса %q", seq, d.SeqLength, cls)
	}

	name := fmt.Sprintf("%s-%s-%0*d", site, cls, d.SeqLength, seq)
	if len(name) > maxHostNameLen {
		return "", fmt.Errorf("naming: имя %q превышает лимит %d символов для хоста", name, maxHostNameLen)
	}
	return name, nil
}

// DNSName возвращает имя в нижнем регистре для использования в DNS (intent §5.1 P3: уникальность
// без учёта регистра, хранение — как есть, DNS — нижний регистр).
func DNSName(name string) string {
	return strings.ToLower(name)
}

// ParsedHost — результат разбора основного имени хоста по шаблону <SITE>-<CLS>-<SEQ>.
type ParsedHost struct {
	Site      string
	Cls       string
	Seq       int64
	SeqDigits int // фактическая ширина номера в разобранном имени (для сверки с designation.SeqLength)
}

// ParseHostName разбирает имя вида <SITE>-<CLS>-<SEQ> без проверки статуса кода (см. Validate
// для полной проверки соответствия профилю).
func ParseHostName(name string) (ParsedHost, error) {
	upper := strings.ToUpper(name)
	parts := strings.Split(upper, "-")
	if len(parts) != 3 {
		return ParsedHost{}, fmt.Errorf("naming: %q не соответствует шаблону <SITE>-<CLS>-<SEQ>", name)
	}
	site, cls, seqStr := parts[0], parts[1], parts[2]
	if !siteRe.MatchString(site) {
		return ParsedHost{}, fmt.Errorf("naming: недопустимый SITE %q", site)
	}
	if !regexp.MustCompile(`^[A-Z]{1,3}$`).MatchString(cls) {
		return ParsedHost{}, fmt.Errorf("naming: недопустимый код класса %q", cls)
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(seqStr) {
		return ParsedHost{}, fmt.Errorf("naming: недопустимый номер %q", seqStr)
	}
	seq, err := strconv.ParseInt(seqStr, 10, 64)
	if err != nil {
		return ParsedHost{}, fmt.Errorf("naming: не удалось разобрать номер %q: %w", seqStr, err)
	}
	return ParsedHost{Site: site, Cls: cls, Seq: seq, SeqDigits: len(seqStr)}, nil
}
