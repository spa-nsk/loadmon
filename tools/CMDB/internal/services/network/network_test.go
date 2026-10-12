package network

import (
	"errors"
	"testing"

	"cmdb/internal/services/svcerr"
)

func TestNormalizeWWN(t *testing.T) {
	ok := map[string]string{
		"50:01:43:80:12:34:56:78":   "50:01:43:80:12:34:56:78",
		"50-01-43-80-12-34-56-78":   "50:01:43:80:12:34:56:78",
		"5001438012345678":          "50:01:43:80:12:34:56:78",
		"0x5001438012ABCDEF":        "50:01:43:80:12:ab:cd:ef",
		"  50:01:43:80:12:AB:CD:EF": "50:01:43:80:12:ab:cd:ef",
	}
	for in, want := range ok {
		got, err := NormalizeWWN(in)
		if err != nil || got != want {
			t.Errorf("NormalizeWWN(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "5001", "50:01:43:80:12:34:56:7g", "50:01:43:80:12:34:56:78:9a"} {
		if _, err := NormalizeWWN(bad); !errors.Is(err, svcerr.ErrInvalidInput) {
			t.Errorf("NormalizeWWN(%q): err = %v, want ErrInvalidInput", bad, err)
		}
	}
}

func TestNormalizeAddress(t *testing.T) {
	ok := map[string]string{
		"10.0.0.5":      "10.0.0.5/32",
		"10.0.0.5/24":   "10.0.0.5/24",
		" 192.168.1.1 ": "192.168.1.1/32",
		"2001:db8::1":   "2001:db8::1/128",
	}
	for in, want := range ok {
		got, err := NormalizeAddress(in)
		if err != nil || got != want {
			t.Errorf("NormalizeAddress(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "10.0.0.256", "10.0.0.5/33", "host", "10.0.0"} {
		if _, err := NormalizeAddress(bad); !errors.Is(err, svcerr.ErrInvalidInput) {
			t.Errorf("NormalizeAddress(%q): err = %v, want ErrInvalidInput", bad, err)
		}
	}
}
