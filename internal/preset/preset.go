package preset

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Definition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Gains       []float64 `json:"gains"`
	OutputGain  float64   `json:"outputGain"`
}

var frequencies = []float64{31, 62, 125, 250, 500, 1000, 2000, 4000, 8000, 16000}

var definitions = []Definition{
	{ID: "balanced", Name: "Balanced", Description: "Natural and neutral for everyday listening", Gains: []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, OutputGain: 0},
	{ID: "movie", Name: "Movie", Description: "Cinematic bass, clear dialogue and spacious detail", Gains: []float64{2, 3, 2, 0.5, -0.5, 0, 1.5, 2, 1, 0}, OutputGain: -3},
	{ID: "gaming", Name: "Gaming", Description: "Footsteps and positional cues come forward", Gains: []float64{-1, 0, 1, -1, -2, 1, 3, 4, 2, 0}, OutputGain: -4},
	{ID: "communication", Name: "Communication", Description: "Speech-focused for meetings, chat and podcasts", Gains: []float64{-6, -4, -2, -1, 1, 3, 4, 3, 0, -3}, OutputGain: -4},
	{ID: "music", Name: "Music", Description: "A gentle hi-fi curve with punch and clarity", Gains: []float64{1.5, 2, 1, 0.5, 0, 0.5, 1.5, 2, 1.5, 1}, OutputGain: -3},
	{ID: "bass", Name: "Bass Boost", Description: "More sub-bass and impact without overwhelming vocals", Gains: []float64{4, 5, 4, 2, 0, -1, 0, 1, 1, 0}, OutputGain: -5},
}

func List() []Definition {
	result := make([]Definition, len(definitions))
	copy(result, definitions)
	return result
}

func Apply(id string) error {
	d, ok := lookup(id)
	if !ok {
		return fmt.Errorf("unknown preset %q", id)
	}
	if _, err := exec.LookPath("easyeffects"); err != nil {
		return errors.New("EasyEffects is not installed")
	}
	if err := Install(); err != nil {
		return err
	}
	cmd := exec.Command("easyeffects", "--load-preset", fileName(d))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("load EasyEffects preset: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func Install() error {
	data := os.Getenv("XDG_DATA_HOME")
	// Snap-packaged IDEs may leak their private XDG path into launched terminals.
	// Native EasyEffects cannot see presets stored inside that IDE-specific tree.
	if data == "" || strings.Contains(filepath.Clean(data), string(filepath.Separator)+"snap"+string(filepath.Separator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		data = filepath.Join(home, ".local", "share")
	}
	dir := filepath.Join(data, "easyeffects", "output")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, d := range definitions {
		contents, err := render(d)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, fileName(d)+".json"), contents, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func lookup(id string) (Definition, bool) {
	for _, d := range definitions {
		if strings.EqualFold(d.ID, id) || strings.EqualFold(d.Name, id) {
			return d, true
		}
	}
	return Definition{}, false
}

func fileName(d Definition) string { return "ROG Delta II - " + d.Name }

func render(d Definition) ([]byte, error) {
	if len(d.Gains) != len(frequencies) {
		return nil, fmt.Errorf("preset %s has %d bands, want %d", d.ID, len(d.Gains), len(frequencies))
	}
	bands := map[string]any{}
	for i, frequency := range frequencies {
		bands[fmt.Sprintf("band%d", i)] = map[string]any{
			"frequency": frequency, "gain": d.Gains[i], "mode": "RLC (BT)", "mute": false,
			"q": 1.41, "slope": "x1", "solo": false, "type": "Bell", "width": 4.0,
		}
	}
	equalizer := map[string]any{
		"balance": 0.0, "bypass": false, "input-gain": 0.0, "left": bands,
		"mode": "IIR", "num-bands": len(frequencies), "output-gain": d.OutputGain,
		"pitch-left": 0.0, "pitch-right": 0.0, "right": bands, "split-channels": false,
	}
	value := map[string]any{"output": map[string]any{
		"blocklist": []string{}, "equalizer#0": equalizer, "plugins_order": []string{"equalizer#0"},
	}}
	return json.MarshalIndent(value, "", "    ")
}
