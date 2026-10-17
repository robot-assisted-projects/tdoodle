package main

import (
	"bytes"
	"testing"
)

func TestScreenColorConfiguration(t *testing.T) {
	for _, tc := range []struct {
		mode, term string
		noColor    bool
		count      int
		force      bool
	}{
		{"auto", "linux", false, 16, true},
		{"auto", "linux", true, 0, true},
		{"auto", "xterm-256color", false, 0, false},
		{"8", "linux", false, 8, true},
		{"16", "linux", false, 16, true},
		{"256", "linux", false, 16, true},
		{"mono", "xterm-256color", false, 0, true},
		{"256", "xterm-256color", false, 256, true},
	} {
		got, force := screenColorCount(tc.mode, tc.term, tc.noColor)
		if got != tc.count || force != tc.force {
			t.Errorf("%+v: got %d, %v", tc, got, force)
		}
	}
}

func TestColorCapabilityOptions(t *testing.T) {
	for _, mode := range []string{"auto", "mono", "8", "16", "256"} {
		got, err := parseOptions([]string{"-colors=" + mode}, &bytes.Buffer{})
		if err != nil || got.colors != mode {
			t.Fatalf("color mode %s: %+v, %v", mode, got, err)
		}
	}
}
