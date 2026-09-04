package audio

import "testing"

func TestSidetoneNodeID(t *testing.T) {
	output := []byte("Audio\n ├─ Streams:\n │  87. rogdelta-sidetone  [active]\n")
	match := sidetoneID.FindSubmatch(output)
	if len(match) != 2 || string(match[1]) != "87" {
		t.Fatalf("did not find sidetone node: %q", match)
	}
}

func TestSidetoneRejectsInvalidLevel(t *testing.T) {
	s := NewSidetone()
	if err := s.Set(true, 101); err == nil {
		t.Fatal("invalid level was accepted")
	}
}
