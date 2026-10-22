package device

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Battery reports come from the vendor HID interface right after the headset connects:
//
//	cc 12 09 00 00 XX   XX = battery percent (0x5c = 92, confirmed against a phone reading)
//	cc 12 08 00 00 YY   YY = 1 while the charging cable is connected, 0 otherwise
//
// They are only read, never requested, so the level is the last one the headset announced.
const (
	reportBattery  = 0x09
	reportCharging = 0x08
)

// ParseBattery extracts a battery percentage from a raw HID report.
func ParseBattery(data []byte) (percent int, ok bool) {
	if len(data) < 6 || data[0] != 0xcc || data[1] != 0x12 || data[2] != reportBattery || data[5] > 100 {
		return 0, false
	}
	return int(data[5]), true
}

// ParseCharging extracts the charging flag from a raw HID report.
func ParseCharging(data []byte) (charging, ok bool) {
	if len(data) < 6 || data[0] != 0xcc || data[1] != 0x12 || data[2] != reportCharging || data[5] > 1 {
		return false, false
	}
	return data[5] == 1, true
}

type batteryState struct {
	mu       sync.Mutex
	percent  *int
	charging *bool
	updated  time.Time
}

var latest batteryState

// statePath is where the last known level is kept so it survives restarts.
var statePath = func() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "rogdelta", "battery.json")
}

type savedBattery struct {
	Percent int       `json:"percent"`
	Updated time.Time `json:"updated"`
}

func loadSaved() {
	path := statePath()
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var v savedBattery
	if json.Unmarshal(b, &v) != nil || v.Percent < 0 || v.Percent > 100 {
		return
	}
	latest.mu.Lock()
	defer latest.mu.Unlock()
	if latest.percent == nil {
		latest.percent, latest.updated = &v.Percent, v.Updated
	}
}

func save(percent int, at time.Time) {
	path := statePath()
	if path == "" {
		return
	}
	b, err := json.Marshal(savedBattery{Percent: percent, Updated: at})
	if err != nil || os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}

func (b *batteryState) snapshot() (*int, *bool, time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.percent, b.charging, b.updated
}

// StartMonitor reads the headset's HID reports in the background and remembers the latest battery
// level. It reopens the device after unplugging or reconnecting. It never writes to the device.
func StartMonitor(ctx context.Context) {
	loadSaved()
	go func() {
		for ctx.Err() == nil {
			s := Detect()
			if s.Connected && s.Readable {
				monitorOnce(ctx, s.HIDRaw)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
}

func monitorOnce(ctx context.Context, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	go func() { <-ctx.Done(); _ = f.Close() }()
	buf := make([]byte, 64)
	for {
		n, err := f.Read(buf)
		if err != nil {
			return
		}
		latest.mu.Lock()
		var saveLevel *int
		if p, ok := ParseBattery(buf[:n]); ok {
			if latest.percent == nil || *latest.percent != p {
				saveLevel = &p
			}
			latest.percent, latest.updated = &p, time.Now()
		}
		if c, ok := ParseCharging(buf[:n]); ok {
			latest.charging = &c
		}
		now := latest.updated
		latest.mu.Unlock()
		if saveLevel != nil {
			save(*saveLevel, now)
		}
	}
}
