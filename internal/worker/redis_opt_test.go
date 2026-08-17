package worker

import (
	"testing"
)

func TestParseRedisOptPlaintext(t *testing.T) {
	opt, err := ParseRedisOpt("redis://:secret@redis:6379/0", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if opt.TLSConfig != nil {
		t.Fatal("expected no TLS for redis:// without forceTLS")
	}
	if opt.Addr != "redis:6379" || opt.Password != "secret" || opt.DB != 0 {
		t.Fatalf("unexpected opt: %+v", opt)
	}
}

func TestParseRedisOptForceTLS(t *testing.T) {
	opt, err := ParseRedisOpt("redis://:secret@redis:6379/0", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if opt.TLSConfig == nil {
		t.Fatal("expected TLS when forceTLS is set")
	}
}

func TestParseRedisOptRediss(t *testing.T) {
	opt, err := ParseRedisOpt("rediss://:secret@redis:6379/0", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if opt.TLSConfig == nil {
		t.Fatal("expected TLS for rediss://")
	}
	if !opt.TLSConfig.InsecureSkipVerify {
		t.Fatal("expected skip verify")
	}
}
