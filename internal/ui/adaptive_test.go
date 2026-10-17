package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
)

func TestAdaptivePaletteLabelsAndSelection(t *testing.T) {
	for _, tc := range []struct {
		name, term, mode string
		reported, fg, bg int
		fgAlias, bgAlias string
	}{
		{"eight", "xterm", "auto", 8, 8, 8, "2/3/8 Brown", "2/3/8 Brown"},
		{"sixteen", "xterm", "auto", 16, 10, 10, "2/3 Yellow", "2/3 Yellow"},
		{"rich", "xterm-256color", "auto", 256, 11, 11, "2 Orange", "2 Orange"},
		{"linux", "linux", "auto", 8, 10, 8, "2/3 Yellow", "2/3/8 Brown"},
		{"override", "xterm-256color", "8", 256, 8, 8, "2/3/8 Brown", "2/3/8 Brown"},
		{"mono", "xterm-256color", "mono", 256, 1, 1, "1/2/3/4/5/6/7/8/9/0/C White", "1/2/3/4/5/6/7/8/9/0/C White"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TERM", tc.term)
			e := newEditor(t)
			e.ColorMode = tc.mode
			e.Text.FG, e.Text.BG = canvas.Orange, canvas.Orange
			for _, background := range []bool{false, true} {
				e.Palette = 1
				wantCount, alias := tc.fg, tc.fgAlias
				if background {
					e.Palette, wantCount, alias = 2, tc.bg, tc.bgAlias
				}
				s := newScreen(240, 4, tc.reported)
				Render(s, e)
				colors := e.PaletteColors(background)
				if len(colors) != wantCount {
					t.Fatalf("BG=%v: got %d choices, want %d", background, len(colors), wantCount)
				}
				row := s.row(3)
				x := strings.Index(row, alias)
				if x < 0 {
					t.Fatalf("BG=%v missing aliases %q in %q", background, alias, row)
				}
				_, style, _ := s.buffer.Get(x, 3)
				if !style.HasBold() {
					t.Fatalf("saved Orange must mark its visible alias as the current color: %q", row)
				}
				if wantCount > 1 && !strings.Contains(row, "C Cyan") {
					t.Fatalf("BG=%v missing cyan choice: %q", background, row)
				}
				if s.outside || e.Text.FG != canvas.Orange || e.Text.BG != canvas.Orange {
					t.Fatal("palette rendering exceeded bounds or rewrote logical colors")
				}
				// A capability-identical redraw must not move the arrow selection.
				e.PaletteIndex = len(colors) - 1
				Render(s, e)
				if e.PaletteIndex != len(colors)-1 {
					t.Fatal("redraw reset the palette arrow selection")
				}
			}
		})
	}
}

func TestAdaptivePalettesFitNarrowDisplays(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	for _, mode := range []string{"mono", "8", "16", "256"} {
		for _, background := range []bool{false, true} {
			for width := 1; width <= 130; width++ {
				e := newEditor(t)
				e.ColorMode = mode
				e.Palette = 1
				if background {
					e.Palette = 2
				}
				s := newScreen(width, 3, 256)
				Render(s, e)
				// Defend rendering even if a stale external palette index survives.
				e.PaletteIndex = 999
				Render(s, e)
				if s.outside {
					t.Fatalf("%s/BG=%v exceeded %d columns", mode, background, width)
				}
			}
		}
	}
}

func TestNarrowPaletteNamesMatchCanonicalShade(t *testing.T) {
	t.Setenv("TERM", "xterm")
	e := newEditor(t)
	e.ColorMode, e.Palette = "8", 2
	s := newScreen(8, 3, 256)
	Render(s, e)
	e.PaletteIndex = e.PaletteIndexFor(canvas.Brown, true)
	Render(s, e)
	if got := s.row(2); got != "BG~ 8Brn" {
		t.Fatalf("narrow alias must identify Brown: %q", got)
	}
	e.PaletteIndex = e.PaletteIndexFor(canvas.Cyan, true)
	Render(s, e)
	if got := s.row(2); got != "BG~ CCyn" {
		t.Fatalf("narrow palette must identify Cyan after deduplication: %q", got)
	}
}

func TestOverrideAppliesToArtworkAndKnobPreview(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	for _, mode := range []string{"mono", "8", "16", "256"} {
		t.Run(mode, func(t *testing.T) {
			e := newEditor(t)
			e.ColorMode = mode
			e.Text.FG, e.Text.BG = canvas.Orange, canvas.Gray
			e.KnobUsed = true
			point := canvas.Point{X: 1, Y: 1}
			cell := canvas.Cell{Rune: '@', FG: canvas.Orange, BG: canvas.Gray}
			e.Doc.Set(point, cell)
			before := append([]canvas.Cell(nil), e.Doc.Cells...)
			for _, background := range []bool{false, true} {
				e.KnobBackground = background
				s := newScreen(180, 5, 256)
				Render(s, e)
				profile := e.PaletteProfile()
				fg, bg := profile.Map(cell.FG, false), profile.Map(cell.BG, true)
				_, drawing, _ := s.buffer.Get(point.X, point.Y)
				if drawing.GetForeground() != fg || drawing.GetBackground() != bg {
					t.Fatal("artwork ignored the color override")
				}
				row := s.row(4)
				x := strings.Index(row, "[Aa]")
				if x < 0 {
					t.Fatalf("missing drawing preview in %q", row)
				}
				_, sample, _ := s.buffer.Get(x+1, 4)
				if sample.GetForeground() != fg || sample.GetBackground() != bg {
					t.Fatalf("Aa does not match artwork in %s mode", mode)
				}
				fgName, bgName := string(profile.Resolve(cell.FG, false)), string(profile.Resolve(cell.BG, true))
				x = strings.Index(row, fgName+"/"+bgName)
				if x < 0 {
					t.Fatalf("status lacks canonical names %s/%s: %q", fgName, bgName, row)
				}
				if background {
					x += len(fgName) + 1
				}
				_, focused, _ := s.buffer.Get(x, 4)
				if mode == "mono" {
					if focused.GetForeground() != color.Default || focused.GetBackground() != color.Default || !focused.HasBold() {
						t.Fatal("monochrome focus must use default colors and bold")
					}
				} else if focused.GetForeground() != color.Teal || focused.GetBackground() != color.Black || focused.HasUnderline() {
					t.Fatalf("focus in %s/BG=%v must remain plain teal on black", mode, background)
				}
				if !reflect.DeepEqual(before, e.Doc.Cells) {
					t.Fatal("display adaptation changed saved drawing colors")
				}
			}
		})
	}
}
