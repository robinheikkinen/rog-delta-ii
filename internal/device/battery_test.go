package device

import (
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"
)

func mustHex(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseBattery(t *testing.T) {
	if p, ok := ParseBattery(mustHex(t, "cc120900005c")); !ok || p != 92 {
		t.Fatalf("got %d %v, want 92 true", p, ok)
	}
	for _, s := range []string{"cc120800000100", "cc12090000ff", "0c00", "cc1209"} {
		if _, ok := ParseBattery(mustHex(t, s)); ok {
			t.Fatalf("%s should not parse as battery", s)
		}
	}
}

func TestParseCharging(t *testing.T) {
	if c, ok := ParseCharging(mustHex(t, "cc1208000001")); !ok || !c {
		t.Fatal("expected charging")
	}
	if c, ok := ParseCharging(mustHex(t, "cc1208000000")); !ok || c {
		t.Fatal("expected not charging")
	}
	if _, ok := ParseCharging(mustHex(t, "cc120900005c")); ok {
		t.Fatal("battery report is not a charging report")
	}
}

func TestSaveAndLoad(t *testing.T) {
	old := statePath
	dir := t.TempDir()
	statePath = func() string { return filepath.Join(dir, "rogdelta", "battery.json") }
	defer func() { statePath = old; latest = batteryState{} }()
	latest = batteryState{}
	at := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	save(87, at)
	loadSaved()
	p, _, u := latest.snapshot()
	if p == nil || *p != 87 || !u.Equal(at) {
		t.Fatalf("got %v %v, want 87 %v", p, u, at)
	}
}
