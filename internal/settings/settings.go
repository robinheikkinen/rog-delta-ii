package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Volume          int       `json:"volume"`
	ActivePreset    string    `json:"activePreset,omitempty"`
	CustomEQ        bool      `json:"customEQ"`
	Gains           []float64 `json:"gains"`
	SidetoneEnabled bool      `json:"sidetoneEnabled"`
	SidetoneLevel   int       `json:"sidetoneLevel"`
}

func Default() State {
	return State{Volume: 60, ActivePreset: "balanced", Gains: make([]float64, 10), SidetoneLevel: 25}
}

type Store struct {
	mu    sync.Mutex
	path  string
	state State
}

func Open() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return OpenAt(filepath.Join(dir, "rogdelta", "settings.json"))
}

func OpenAt(path string) (*Store, error) {
	s := &Store{path: path, state: Default()}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &s.state); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if len(s.state.Gains) != 10 {
		s.state.Gains = make([]float64, 10)
	}
	if s.state.SidetoneLevel < 0 || s.state.SidetoneLevel > 100 {
		s.state.SidetoneLevel = 25
	}
	if s.state.Volume < 0 || s.state.Volume > 100 {
		s.state.Volume = 60
	}
	return s, nil
}

func (s *Store) Get() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := s.state
	result.Gains = append([]float64(nil), s.state.Gains...)
	return result
}

func (s *Store) Update(change func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(&s.state)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
