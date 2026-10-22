package device

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	VendorID  = "0b05"
	ProductID = "1afa"
)

var ErrNotFound = errors.New("ROG Delta II dongle not found")

type Status struct {
	Connected      bool       `json:"connected"`
	Name           string     `json:"name"`
	VendorID       string     `json:"vendorId"`
	ProductID      string     `json:"productId"`
	HIDRaw         string     `json:"hidraw,omitempty"`
	Readable       bool       `json:"readable"`
	Protocol       string     `json:"protocol"`
	BatteryPercent *int       `json:"batteryPercent,omitempty"`
	Charging       *bool      `json:"charging,omitempty"`
	BatteryUpdated *time.Time `json:"batteryUpdated,omitempty"`
}

func Detect() Status {
	s := Status{Name: "ASUS ROG Delta II (2.4GHz)", VendorID: VendorID, ProductID: ProductID, Protocol: "discovery"}
	entries, _ := filepath.Glob("/sys/class/hidraw/hidraw*")
	for _, entry := range entries {
		uevent, err := os.ReadFile(filepath.Join(entry, "device", "uevent"))
		if err != nil {
			continue
		}
		upper := strings.ToUpper(string(uevent))
		if !strings.Contains(upper, "HID_ID=0003:00000B05:00001AFA") {
			continue
		}
		s.Connected = true
		s.HIDRaw = "/dev/" + filepath.Base(entry)
		if f, err := os.Open(s.HIDRaw); err == nil {
			s.Readable = true
			f.Close()
		}
		break
	}
	if s.Connected {
		if p, c, u := latest.snapshot(); p != nil || c != nil {
			s.BatteryPercent, s.Charging = p, c
			if p != nil && !u.IsZero() {
				s.BatteryUpdated = &u
			}
		}
	}
	return s
}

type Event struct {
	Time     time.Time `json:"time"`
	Device   string    `json:"device"`
	ReportID byte      `json:"reportId"`
	Data     string    `json:"data"`
}

// Record only reads interrupt reports. It deliberately never writes unverified vendor commands.
func Record(ctx context.Context, path, output string) error {
	if path == "" {
		s := Detect()
		if !s.Connected {
			return ErrNotFound
		}
		path = s.HIDRaw
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w (install the udev rule)", path, err)
	}
	defer f.Close()
	out, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := f.Read(buf)
			if err != nil {
				done <- err
				return
			}
			if n == 0 {
				continue
			}
			e := Event{Time: time.Now().UTC(), Device: path, ReportID: buf[0], Data: hex.EncodeToString(buf[:n])}
			if err := json.NewEncoder(w).Encode(e); err != nil {
				done <- err
				return
			}
			if err := w.Flush(); err != nil {
				done <- err
				return
			}
			fmt.Printf("%s report=0x%02x %s\n", e.Time.Format(time.RFC3339Nano), e.ReportID, e.Data)
		}
	}()
	select {
	case <-ctx.Done():
		_ = f.Close()
		return nil
	case err := <-done:
		if errors.Is(err, os.ErrClosed) || errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}
