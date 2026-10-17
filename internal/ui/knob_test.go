package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
)

func TestKnobStatusDisplaysSelectedColors(t *testing.T) {
	t.Setenv("TERM", "linux")
	e := newEditor(t)
	e.Text.FG, e.Text.BG = canvas.Red, canvas.Blue
	e.AdjustKnobColor(0, true)
	s := newScreen(160, 5, 16)
	Render(s, e)
	row := s.row(4)
	fg, bg, sample := strings.Index(row, "red/blue"), strings.Index(row, "blue"), strings.Index(row, "[Aa]")
	if fg < 0 || bg < 0 || sample < 0 {
		t.Fatalf("missing swatches: %q", row)
	}
	_, style, _ := s.buffer.Get(fg, 4)
	if style.GetForeground() != ColorFor(canvas.Red, false, 16, true) {
		t.Fatal("FG swatch lost its color")
	}
	_, style, _ = s.buffer.Get(bg, 4)
	if style.GetForeground() != color.Teal || style.GetBackground() != color.Black || style.HasUnderline() {
		t.Fatal("focused BG name must be plain teal on black")
	}
	_, style, _ = s.buffer.Get(sample+1, 4)
	if style.GetForeground() != ColorFor(canvas.Red, false, 16, true) || style.GetBackground() != ColorFor(canvas.Blue, true, 16, true) {
		t.Fatal("sample does not show actual drawing colors")
	}
	e.AdjustKnobColor(1, false)
	Render(s, e)
	_, style, _ = s.buffer.Get(strings.Index(s.row(4), "[Aa]")+1, 4)
	if style.GetBackground() != ColorFor(canvas.Violet, true, 16, true) {
		t.Fatal("sample did not update when knob changed BG")
	}
	for width := 1; width < 80; width++ {
		s := newScreen(width, 4, 16)
		Render(s, e)
		if s.outside {
			t.Fatalf("swatch exceeded %d-column screen", width)
		}
	}
	for _, selected := range canvas.BackgroundColors {
		e.Text.FG, e.Text.BG = selected, selected
		for _, background := range []bool{false, true} {
			e.KnobBackground = background
			Render(s, e)
			profile := e.PaletteProfile()
			fg, bg := string(profile.Resolve(selected, false)), string(profile.Resolve(selected, true))
			x := strings.Index(s.row(4), fg+"/"+bg)
			if x < 0 {
				t.Fatalf("missing canonical color names %s/%s in %q", fg, bg, s.row(4))
			}
			if background {
				x += len(fg) + 1
			}
			_, style, _ := s.buffer.Get(x, 4)
			if style.GetForeground() != color.Teal || style.GetBackground() != color.Black || style.HasUnderline() {
				t.Fatalf("focused %s, BG=%v must always be teal on black", selected, background)
			}
			// Switching focus restores the inactive name's normal color swatch.
			if background {
				x -= len(fg) + 1
				_, style, _ = s.buffer.Get(x, 4)
				if style != swatchStyle(selected, 1, 16, true) {
					t.Fatal("inactive FG swatch lost its color")
				}
			} else {
				x += len(fg) + 1
				_, style, _ = s.buffer.Get(x, 4)
				if style != swatchStyle(selected, 2, 16, true) {
					t.Fatal("inactive BG swatch lost its color")
				}
			}
		}
	}
}

func TestCyanBackgroundPaletteRendering(t *testing.T) {
	t.Setenv("TERM", "linux")
	e := newEditor(t)
	Render(newScreen(160, 4, 16), e)
	e.Palette, e.PaletteIndex = 2, e.PaletteIndexFor(canvas.Cyan, true)
	e.Text.BG = canvas.Cyan
	for width := 1; width <= 160; width++ {
		s := newScreen(width, 4, 16)
		Render(s, e)
		if s.outside {
			t.Fatalf("cyan palette exceeded %d columns", width)
		}
		if width == 160 {
			x := strings.Index(s.row(3), "[C Cyan]")
			if x < 0 {
				t.Fatalf("missing cyan shortcut: %s", s.row(3))
			}
			_, style, _ := s.buffer.Get(x, 3)
			if style.GetBackground() != color.Teal {
				t.Fatal("cyan palette entry lost its cyan background")
			}
		}
	}
	e.Palette, e.PaletteIndex = 1, 0
	s := newScreen(160, 4, 16)
	Render(s, e)
	if !strings.Contains(s.row(3), "C Cyan") {
		t.Fatal("cyan must also be available in the adaptive foreground palette")
	}
}
