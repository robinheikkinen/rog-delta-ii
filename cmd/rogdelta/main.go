package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mbekkelund/rog-delta-ii/internal/audio"
	"github.com/mbekkelund/rog-delta-ii/internal/device"
	"github.com/mbekkelund/rog-delta-ii/internal/preset"
	"github.com/mbekkelund/rog-delta-ii/internal/settings"
	"github.com/mbekkelund/rog-delta-ii/internal/webui"
)

var version = "0.1.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rogdelta:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return serve(args)
	}
	switch args[0] {
	case "serve":
		return serve(args[1:])
	case "desktop":
		return desktop(args[1:])
	case "status":
		return printStatus()
	case "volume":
		if len(args) != 2 {
			return errors.New("usage: rogdelta volume <0-100>")
		}
		value, err := strconv.Atoi(args[1])
		if err != nil || value < 0 || value > 100 {
			return errors.New("volume must be 0-100")
		}
		return wpctl("set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%", value))
	case "mute":
		return wpctl("set-mute", "@DEFAULT_AUDIO_SINK@", "toggle")
	case "mic-mute":
		return wpctl("set-mute", "@DEFAULT_AUDIO_SOURCE@", "toggle")
	case "preset":
		if len(args) != 2 {
			return errors.New("usage: rogdelta preset <balanced|movie|gaming|communication|music|bass>")
		}
		return preset.Apply(args[1])
	case "record":
		return record(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (try rogdelta help)", args[0])
	}
}

func usage() {
	fmt.Print(`ROG Delta II control and protocol discovery for Linux

Usage:
  rogdelta [serve] [--listen 127.0.0.1:17890]
  rogdelta desktop
  rogdelta status
  rogdelta volume <0-100>
  rogdelta mute | mic-mute
  rogdelta preset <balanced|movie|gaming|communication|music|bass>
  rogdelta record [--device /dev/hidrawN] [--output capture.jsonl]
`)
}

func printStatus() error {
	s := device.Detect()
	b, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(b))
	if !s.Connected {
		return device.ErrNotFound
	}
	return nil
}

func wpctl(args ...string) error {
	cmd := exec.Command("wpctl", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func record(args []string) error {
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	path := fs.String("device", "", "hidraw device (auto-detected by VID/PID when empty)")
	out := fs.String("output", "capture.jsonl", "JSON Lines output")
	seconds := fs.Int("seconds", 0, "stop after N seconds (0 waits for Ctrl-C)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *seconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*seconds)*time.Second)
		defer cancel()
	}
	return device.Record(ctx, *path, *out)
}

func serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listen := fs.String("listen", "127.0.0.1:17890", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sidetone := audio.NewSidetone()
	defer sidetone.Close()
	store, err := settings.Open()
	if err != nil {
		return fmt.Errorf("open settings: %w", err)
	}
	restore(store.Get(), sidetone)
	mux := routes(sidetone, store)
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	log.Printf("ROG Delta II UI: http://%s", listener.Addr())
	return runServer(listener, mux, nil)
}

func runServer(listener net.Listener, handler http.Handler, closed <-chan struct{}) error {
	server := &http.Server{Handler: securityHeaders(handler), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		select {
		case <-ctx.Done():
		case <-closed:
		}
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err := server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func desktop(args []string) error {
	fs := flag.NewFlagSet("desktop", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	browser, err := findAppBrowser()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	sidetone := audio.NewSidetone()
	defer sidetone.Close()
	store, err := settings.Open()
	if err != nil {
		listener.Close()
		return fmt.Errorf("open settings: %w", err)
	}
	restore(store.Get(), sidetone)
	mux := routes(sidetone, store)

	profile, err := os.MkdirTemp("", "rogdelta-desktop-")
	if err != nil {
		listener.Close()
		return err
	}
	defer os.RemoveAll(profile)
	url := "http://" + listener.Addr().String() + "/"
	cmd := exec.Command(browser, "--app="+url, "--user-data-dir="+profile, "--no-first-run", "--disable-sync")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	closed := make(chan struct{})
	serverErr := make(chan error, 1)
	go func() { serverErr <- runServer(listener, mux, closed) }()
	if err := cmd.Start(); err != nil {
		close(closed)
		<-serverErr
		return fmt.Errorf("start desktop window: %w", err)
	}
	log.Printf("desktop window started with %s", browser)
	waitErr := cmd.Wait()
	close(closed)
	serveErr := <-serverErr
	if waitErr != nil {
		return fmt.Errorf("desktop window: %w", waitErr)
	}
	return serveErr
}

func routes(sidetone *audio.Sidetone, store *settings.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusTemporaryRedirect)
	})
	mux.Handle("/", webui.Handler())
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) { jsonResponse(w, http.StatusOK, device.Detect()) })
	mux.HandleFunc("GET /api/presets", presetsListHandler)
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, _ *http.Request) { jsonResponse(w, 200, store.Get()) })
	mux.HandleFunc("POST /api/presets/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !presetApplyHandler(w, r) {
			return
		}
		_ = store.Update(func(s *settings.State) { s.ActivePreset = r.PathValue("id"); s.CustomEQ = false })
	})
	mux.HandleFunc("POST /api/equalizer", func(w http.ResponseWriter, r *http.Request) { equalizerHandler(w, r, store) })
	mux.HandleFunc("POST /api/sidetone", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Enabled bool `json:"enabled"`
			Level   int  `json:"level"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			jsonResponse(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		if err := sidetone.Set(body.Enabled, body.Level); err != nil {
			jsonResponse(w, 500, map[string]string{"error": err.Error()})
			return
		}
		_ = store.Update(func(s *settings.State) { s.SidetoneEnabled = body.Enabled; s.SidetoneLevel = body.Level })
		jsonResponse(w, 200, map[string]any{"ok": true, "enabled": body.Enabled, "level": body.Level})
	})
	mux.HandleFunc("POST /api/volume", func(w http.ResponseWriter, r *http.Request) { volumeHandler(w, r, store) })
	mux.HandleFunc("POST /api/mute/{kind}", muteHandler)
	return mux
}

func restore(state settings.State, sidetone *audio.Sidetone) {
	if err := wpctl("set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%", state.Volume)); err != nil {
		log.Printf("restore volume: %v", err)
	}
	var err error
	if state.CustomEQ {
		err = preset.ApplyCustom(state.Gains)
	} else if state.ActivePreset != "" {
		err = preset.Apply(state.ActivePreset)
	}
	if err != nil {
		log.Printf("restore equalizer: %v", err)
	}
	if state.SidetoneEnabled {
		if err := sidetone.Set(true, state.SidetoneLevel); err != nil {
			log.Printf("restore sidetone: %v", err)
		}
	}
}

func equalizerHandler(w http.ResponseWriter, r *http.Request, store *settings.Store) {
	var body struct {
		Gains []float64 `json:"gains"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		jsonResponse(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	if err := preset.ApplyCustom(body.Gains); err != nil {
		jsonResponse(w, 400, map[string]string{"error": err.Error()})
		return
	}
	_ = store.Update(func(s *settings.State) {
		s.Gains = append([]float64(nil), body.Gains...)
		s.CustomEQ = true
		s.ActivePreset = ""
	})
	jsonResponse(w, 200, map[string]any{"ok": true, "frequencies": preset.Frequencies()})
}

func findAppBrowser() (string, error) {
	for _, name := range []string{"vivaldi-stable", "vivaldi", "google-chrome-stable", "google-chrome", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no supported app-mode browser found (install Vivaldi, Chromium, or Chrome)")
}

func volumeHandler(w http.ResponseWriter, r *http.Request, store *settings.Store) {
	var body struct {
		Value int `json:"value"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil || body.Value < 0 || body.Value > 100 {
		jsonResponse(w, 400, map[string]string{"error": "value must be 0-100"})
		return
	}
	if err := wpctl("set-volume", "@DEFAULT_AUDIO_SINK@", fmt.Sprintf("%d%%", body.Value)); err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	_ = store.Update(func(s *settings.State) { s.Volume = body.Value })
	jsonResponse(w, 200, map[string]bool{"ok": true})
}

func muteHandler(w http.ResponseWriter, r *http.Request) {
	target := "@DEFAULT_AUDIO_SINK@"
	if strings.EqualFold(r.PathValue("kind"), "mic") {
		target = "@DEFAULT_AUDIO_SOURCE@"
	}
	if err := wpctl("set-mute", target, "toggle"); err != nil {
		jsonResponse(w, 500, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
}

func presetsListHandler(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, preset.List())
}

func presetApplyHandler(w http.ResponseWriter, r *http.Request) bool {
	id := r.PathValue("id")
	if err := preset.Apply(id); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return false
	}
	jsonResponse(w, http.StatusOK, map[string]any{"ok": true, "active": id})
	return true
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
