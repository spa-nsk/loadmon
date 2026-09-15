package config

import "testing"

func TestResolveDomainPrivilege(t *testing.T) {
	cfg := JsonConfig{
		Auth: []AuthenticationGroup{
			{Privilege: 3, Name: "admins", ObjectSID: "S-1-5-21-1-2-3-1101"},
			{Privilege: 1, Name: "editors", ObjectSID: "S-1-5-21-1-2-3-1102"},
			{Privilege: 0, Name: "viewers", ObjectSID: "S-1-5-21-1-2-3-1103"},
		},
	}

	t.Run("single matching group", func(t *testing.T) {
		priv, found := cfg.ResolveDomainPrivilege([]string{"S-1-5-21-1-2-3-1102", "S-1-5-21-1-2-3-9999"})
		if !found || priv != 1 {
			t.Fatalf("got priv=%d found=%v, want priv=1 found=true", priv, found)
		}
	})

	t.Run("multiple matching groups take max privilege", func(t *testing.T) {
		priv, found := cfg.ResolveDomainPrivilege([]string{"S-1-5-21-1-2-3-1103", "S-1-5-21-1-2-3-1101"})
		if !found || priv != 3 {
			t.Fatalf("got priv=%d found=%v, want priv=3 found=true", priv, found)
		}
	})

	t.Run("no matching group", func(t *testing.T) {
		priv, found := cfg.ResolveDomainPrivilege([]string{"S-1-5-21-1-2-3-9999"})
		if found {
			t.Fatalf("got found=true priv=%d, want found=false", priv)
		}
	})

	t.Run("empty group list", func(t *testing.T) {
		priv, found := cfg.ResolveDomainPrivilege(nil)
		if found {
			t.Fatalf("got found=true priv=%d, want found=false", priv)
		}
	})

	t.Run("empty Auth config denies even with groups", func(t *testing.T) {
		empty := JsonConfig{}
		priv, found := empty.ResolveDomainPrivilege([]string{"S-1-5-21-1-2-3-1101"})
		if found {
			t.Fatalf("got found=true priv=%d, want found=false", priv)
		}
	})

	t.Run("entries with empty ObjectSID never match", func(t *testing.T) {
		withEmptySID := JsonConfig{
			Auth: []AuthenticationGroup{{Privilege: 3, Name: "broken-entry", ObjectSID: ""}},
		}
		priv, found := withEmptySID.ResolveDomainPrivilege([]string{""})
		if found {
			t.Fatalf("got found=true priv=%d, want found=false (empty ObjectSID must not match empty group SID)", priv)
		}
	})
}
