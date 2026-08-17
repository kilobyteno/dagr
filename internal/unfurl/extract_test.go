package unfurl

import (
	"net"
	"testing"
)

func TestExtractURLs(t *testing.T) {
	t.Parallel()
	body := `See https://example.com/a and http://example.com/b, plus https://example.com/a again.
Also markdown [x](https://example.org/path) and bad ftp://example.com/x`
	got := ExtractURLs(body)
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
	if got[0] != "https://example.com/a" || got[1] != "http://example.com/b" || got[2] != "https://example.org/path" {
		t.Fatalf("unexpected order/values: %#v", got)
	}
}

func TestExtractURLsCapsAtMax(t *testing.T) {
	t.Parallel()
	body := "https://a.example/1 https://b.example/2 https://c.example/3 https://d.example/4"
	got := ExtractURLs(body)
	if len(got) != MaxURLsPerMessage {
		t.Fatalf("len=%d want %d", len(got), MaxURLsPerMessage)
	}
}

func TestIsPublicIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"127.0.0.1", false},
		{"10.0.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false},
		{"100.64.1.1", false},
		{"::1", false},
	}
	for _, tc := range cases {
		if got := isPublicIP(net.ParseIP(tc.ip)); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.ip, got, tc.want)
		}
	}
}

func TestValidateFetchURLBlocksPrivate(t *testing.T) {
	t.Parallel()
	if _, err := validateFetchURL("http://127.0.0.1/"); err == nil {
		t.Fatal("expected localhost block")
	}
	if _, err := validateFetchURL("http://192.168.0.10/path"); err == nil {
		t.Fatal("expected private block")
	}
	if _, err := validateFetchURL("ftp://example.com"); err == nil {
		t.Fatal("expected scheme block")
	}
}
