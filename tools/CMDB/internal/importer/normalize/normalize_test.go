package normalize

import "testing"

func TestUUID(t *testing.T) {
	got, err := UUID("4C4C4544-0043-3410-8035-C7C04F435931")
	if err != nil {
		t.Fatalf("UUID: %v", err)
	}
	if want := "4c4c4544-0043-3410-8035-c7c04f435931"; got != want {
		t.Errorf("UUID = %q, want %q", got, want)
	}
	if got, err := UUID(""); got != "" || err != nil {
		t.Errorf("UUID(\"\") = %q, %v, want \"\", nil", got, err)
	}
	if _, err := UUID("not-a-uuid"); err == nil {
		t.Error("UUID(мусор) expected error")
	}
}

func TestGB(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"16", "16"},
		{"16 GB", "16"},
		{"16GB", "16"},
		{"16384MB", "16"},
		{"16384 MB", "16"},
		{"1 TB", "1024"},
		{"16 Гб", "16"},
		{"16384 Мб", "16"},
	}
	for _, c := range cases {
		got, err := GB(c.in)
		if err != nil {
			t.Errorf("GB(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("GB(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if _, err := GB("много"); err == nil {
		t.Error("GB(мусор) expected error")
	}
	if _, err := GB("16 фунтов"); err == nil {
		t.Error("GB(неизвестная единица) expected error")
	}
}

func TestTrim(t *testing.T) {
	if got := Trim("  Dell  "); got != "Dell" {
		t.Errorf("Trim = %q, want %q", got, "Dell")
	}
}
