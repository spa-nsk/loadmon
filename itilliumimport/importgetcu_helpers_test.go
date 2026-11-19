package itilliumimport

import "testing"

func TestParseBladeBucketName(t *testing.T) {
	if got := parseBladeBucketName("(IT-0000003911) C7000-15"); got != "C7000-15" {
		t.Fatalf("got=%q", got)
	}
	if got := parseBladeBucketName("C7000-01"); got != "C7000-01" {
		t.Fatalf("got=%q", got)
	}
	if got := parseBladeBucketName(""); got != "" {
		t.Fatalf("empty")
	}
	if got := parseBladeBucketName("  "); got != "" {
		t.Fatalf("spaces")
	}
}