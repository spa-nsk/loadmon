package itilium

import (
	"encoding/json"
	"strconv"
	"strings"
)

// FlexBool разбирает булевы значения ITILIUM, которые в примерах документации
// приходят и как JSON-булевы, и как строки "true"/"false".
type FlexBool bool

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if string(data) == "null" || len(data) == 0 {
		*b = false
		return nil
	}
	var v bool
	if err := json.Unmarshal(data, &v); err == nil {
		*b = FlexBool(v)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "да":
		*b = true
	default:
		*b = false
	}
	return nil
}

func (b FlexBool) Bool() bool { return bool(b) }

// FlexString принимает и JSON-строку, и число (PriorityCode в примерах —
// число, CategoryCode — строка).
type FlexString string

func (s *FlexString) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if string(data) == "null" || len(data) == 0 {
		*s = ""
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = FlexString(str)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*s = FlexString(n.String())
		return nil
	}
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		*s = FlexString(strconv.FormatBool(b))
		return nil
	}
	*s = FlexString(string(data))
	return nil
}

func (s FlexString) String() string { return string(s) }

func bytesTrimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\t' || b[j-1] == '\n' || b[j-1] == '\r') {
		j--
	}
	return b[i:j]
}
