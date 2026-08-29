package main

import "testing"

func TestRunRejectsInvalidVolume(t *testing.T) {
	for _, value := range []string{"-1", "101", "nope"} {
		if err := run([]string{"volume", value}); err == nil {
			t.Fatalf("volume %q should fail", value)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	if err := run([]string{"explode"}); err == nil {
		t.Fatal("unknown command should fail")
	}
}
