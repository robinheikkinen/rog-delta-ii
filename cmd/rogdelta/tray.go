package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"time"

	"fyne.io/systray"

	"github.com/mbekkelund/rog-delta-ii/internal/device"
)

// 3x5 pixel digits, one string per row.
var digits = map[rune][5]string{
	'0': {"111", "101", "101", "101", "111"},
	'1': {"010", "110", "010", "010", "111"},
	'2': {"111", "001", "111", "100", "111"},
	'3': {"111", "001", "111", "001", "111"},
	'4': {"101", "101", "111", "001", "001"},
	'5': {"111", "100", "111", "001", "111"},
	'6': {"111", "100", "111", "101", "111"},
	'7': {"111", "001", "010", "010", "010"},
	'8': {"111", "101", "111", "101", "111"},
	'9': {"111", "101", "111", "001", "111"},
	'-': {"000", "000", "111", "000", "000"},
}

// trayIcon draws the battery percent as pixel digits on a 32x32 PNG.
func trayIcon(percent *int, charging bool) []byte {
	text, col := "--", color.RGBA{150, 155, 165, 255}
	if percent != nil {
		text = fmt.Sprintf("%d", *percent)
		switch {
		case charging:
			col = color.RGBA{80, 160, 255, 255}
		case *percent <= 15:
			col = color.RGBA{255, 35, 79, 255}
		case *percent <= 30:
			col = color.RGBA{255, 176, 32, 255}
		default:
			col = color.RGBA{85, 229, 175, 255}
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	scale, gap := 4, 4
	if len(text) == 3 {
		scale, gap = 3, 1
	}
	width := len(text)*3*scale + (len(text)-1)*gap
	x0, y0 := (32-width)/2, (32-5*scale)/2
	for i, r := range text {
		glyph := digits[r]
		for y, row := range glyph {
			for x, c := range row {
				if c != '1' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.Set(x0+i*(3*scale+gap)+x*scale+dx, y0+y*scale+dy, col)
					}
				}
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func runTray() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	device.StartMonitor(ctx)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	systray.Run(func() { trayReady(ctx, cancel, self) }, cancel)
	return nil
}

func trayReady(ctx context.Context, cancel context.CancelFunc, self string) {
	systray.SetIcon(trayIcon(nil, false))
	systray.SetTitle("ROG Delta II")
	systray.SetTooltip("ROG Delta II: waiting for battery report")
	info := systray.AddMenuItem("Battery: --", "Last level reported by the headset")
	info.Disable()
	open := systray.AddMenuItem("Open window", "Open the ROG Delta II control center")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Close the tray icon")

	go func() {
		var shown *int
		var shownCharging bool
		first := true
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			s := device.Detect()
			charging := s.Charging != nil && *s.Charging
			changed := first || (shown == nil) != (s.BatteryPercent == nil) ||
				(shown != nil && s.BatteryPercent != nil && *shown != *s.BatteryPercent) || shownCharging != charging
			if changed {
				first, shown, shownCharging = false, s.BatteryPercent, charging
				systray.SetIcon(trayIcon(s.BatteryPercent, charging))
				text := "waiting for battery report"
				if s.BatteryPercent != nil {
					text = fmt.Sprintf("%d%%", *s.BatteryPercent)
					if charging {
						text += " (charging)"
					}
					if s.BatteryUpdated != nil {
						info.SetTitle("Battery: " + text + " · " + s.BatteryUpdated.Local().Format("Jan 2 15:04"))
					} else {
						info.SetTitle("Battery: " + text)
					}
				} else {
					info.SetTitle("Battery: --")
				}
				if !s.Connected {
					text = "headset dongle not found"
				}
				systray.SetTooltip("ROG Delta II: " + text)
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
	go func() {
		for {
			select {
			case <-open.ClickedCh:
				cmd := exec.Command(self, "desktop")
				if cmd.Start() == nil {
					go cmd.Wait()
				}
			case <-quit.ClickedCh:
				cancel()
				systray.Quit()
				return
			}
		}
	}()
}
