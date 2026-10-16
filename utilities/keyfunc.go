package utilities

var keymap map[string]string

func init() {
	data := GetKey()
	keymap = make(map[string]string, 0)
	keymap["usr-nsk-dati-02@global.bcs"] = data[0]
	keymap["svc-vmw-vc-ro-01@am-uralsib.ru"] = data[1]
	keymap["work"] = data[2]
	keymap["vnxview"] = data[3]
	keymap["monitor"] = data[4]
	keymap["spaview"] = data[5]
	keymap["usr-nsk-dati-02@t-global.bcs"] = data[6]
	keymap["spausvc"] = data[7]
	keymap["vcsim@local"] = "pass"
	keymap["spausvc2"] = data[8]
	keymap["8"] = data[7]
	keymap["9"] = data[7]
}

// GetKeyData возвращает пароль/ключ, сопоставленный с указанной учётной записью key.
// Если ключ не найден, возвращается строка "key".
func GetKeyData(key string) string {
	m, ok := keymap[key]
	if ok {
		return m
	}
	return "key"
}
