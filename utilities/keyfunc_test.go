package utilities

import "testing"

func TestGetKeyDataKnownKeys(t *testing.T) {
	data := GetKey()

	cases := map[string]string{
		"usr-nsk-dati-02@global.bcs":     data[0],
		"svc-vmw-vc-ro-01@am-uralsib.ru": data[1],
		"work":                           data[2],
		"vnxview":                        data[3],
		"monitor":                        data[4],
		"spaview":                        data[5],
		"usr-nsk-dati-02@t-global.bcs":   data[6],
		"spausvc":                        data[7],
		"vcsim@local":                    "pass",
		"8":                              data[7],
		"9":                              data[7],
	}

	for key, want := range cases {
		if got := GetKeyData(key); got != want {
			t.Errorf("GetKeyData(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestGetKeyDataUnknownKey(t *testing.T) {
	if got := GetKeyData("no-such-account"); got != "key" {
		t.Errorf(`GetKeyData("no-such-account") = %q, want "key"`, got)
	}
}
