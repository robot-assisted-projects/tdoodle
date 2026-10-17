package main

import "tdoodle/internal/palette"

// tcell detects Linux as eight colors and otherwise approximates bright
// foregrounds back to those eight. Configure its known sixteen-color support
// as well as the drawing palette, while keeping backgrounds limited to eight.
func screenColorCount(mode, term string, noColor bool) (int, bool) {
	if mode == "auto" {
		if noColor {
			return 0, true
		}
		if term == "linux" {
			return 16, true
		}
		return 0, false
	}
	profile, err := palette.ForMode(mode, 256, term == "linux")
	if err != nil {
		return 0, false // parseOptions rejects invalid modes before opening a terminal.
	}
	return profile.MaxColors(false), true
}
