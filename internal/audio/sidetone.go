package audio

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

// Sidetone provides safe software microphone monitoring through PipeWire.
type Sidetone struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	level int
}

var sidetoneID = regexp.MustCompile(`(?m)^.*?([0-9]+)\.\s+rogdelta-sidetone\b`)

func NewSidetone() *Sidetone { return &Sidetone{level: 25} }

func (s *Sidetone) Set(enabled bool, level int) error {
	if level < 0 || level > 100 {
		return errors.New("sidetone level must be 0-100")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.level = level
	if !enabled {
		return s.stopLocked()
	}
	if s.cmd == nil {
		if _, err := exec.LookPath("pw-loopback"); err != nil {
			return errors.New("pw-loopback is not installed")
		}
		s.cmd = exec.Command("pw-loopback", "--name", "rogdelta-sidetone", "--latency", "20", "--playback-props", "node.name=rogdelta-sidetone node.description=ROG_Delta_II_Sidetone")
		if err := s.cmd.Start(); err != nil {
			s.cmd = nil
			return fmt.Errorf("start sidetone: %w", err)
		}
		go func(cmd *exec.Cmd) {
			_ = cmd.Wait()
			s.mu.Lock()
			if s.cmd == cmd {
				s.cmd = nil
			}
			s.mu.Unlock()
		}(s.cmd)
	}
	id := ""
	for attempt := 0; attempt < 10; attempt++ {
		output, _ := exec.Command("wpctl", "status", "-n").Output()
		if match := sidetoneID.FindSubmatch(output); len(match) == 2 {
			id = string(match[1])
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if id == "" || exec.Command("wpctl", "set-volume", id, fmt.Sprintf("%d%%", level)).Run() != nil {
		_ = s.stopLocked()
		return errors.New("could not set sidetone level (PipeWire did not expose the monitor stream)")
	}
	return nil
}

func (s *Sidetone) stopLocked() error {
	if s.cmd == nil || s.cmd.Process == nil {
		s.cmd = nil
		return nil
	}
	err := s.cmd.Process.Kill()
	s.cmd = nil
	return err
}

func (s *Sidetone) Close() error { s.mu.Lock(); defer s.mu.Unlock(); return s.stopLocked() }
