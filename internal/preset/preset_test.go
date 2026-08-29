package preset

import (
	"encoding/json"
	"testing"
)

func TestDefinitionsRenderAsEasyEffectsPreset(t *testing.T) {
	for _, d := range List() {
		data, err := render(d)
		if err != nil {
			t.Fatalf("%s: %v", d.ID, err)
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatalf("%s: %v", d.ID, err)
		}
		output := value["output"].(map[string]any)
		order := output["plugins_order"].([]any)
		if len(order) != 1 || order[0] != "equalizer#0" {
			t.Fatalf("%s: bad plugin order", d.ID)
		}
	}
}

func TestUnknownPreset(t *testing.T) {
	if _, ok := lookup("surprise"); ok {
		t.Fatal("unknown preset matched")
	}
}
