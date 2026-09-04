package settings

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	store, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if err := store.Update(func(s *State) { s.Volume = 72; s.CustomEQ = true; s.Gains = want }); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get()
	if got.Volume != 72 || !got.CustomEQ || !reflect.DeepEqual(got.Gains, want) {
		t.Fatalf("state was not restored: %#v", got)
	}
}
