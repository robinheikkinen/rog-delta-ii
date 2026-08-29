package device

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventJSONUsesStableFieldNames(t *testing.T) {
	e := Event{Time: time.Unix(0, 0).UTC(), Device: "/dev/hidraw0", ReportID: 7, Data: "0701"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	if value["reportId"] != float64(7) || value["data"] != "0701" {
		t.Fatalf("unexpected event JSON: %s", b)
	}
}

func TestKnownUSBIdentity(t *testing.T) {
	if VendorID != "0b05" || ProductID != "1afa" {
		t.Fatalf("unexpected USB identity %s:%s", VendorID, ProductID)
	}
}
