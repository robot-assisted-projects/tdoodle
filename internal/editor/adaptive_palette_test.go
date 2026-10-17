package editor

import (
	"reflect"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"

	"tdoodle/internal/canvas"
	"tdoodle/internal/palette"
)

func TestAdaptiveEditorUsesSeparateLinuxPalettes(t *testing.T) {
	e := newTestEditor(t)
	profile := palette.Detect(8, true)
	e.SetPaletteProfile(profile)
	if e.PaletteProfile() != profile {
		t.Fatal("editor lost its detected terminal profile")
	}
	wantFG := []canvas.Color{canvas.Red, canvas.Yellow, canvas.Green, canvas.Blue, canvas.Violet, canvas.Gray, canvas.Brown, canvas.White, canvas.Black, canvas.Cyan}
	wantBG := []canvas.Color{canvas.Red, canvas.Brown, canvas.Green, canvas.Blue, canvas.Violet, canvas.White, canvas.Black, canvas.Cyan}
	if got := e.PaletteColors(false); !reflect.DeepEqual(got, wantFG) {
		t.Fatalf("Linux foreground choices = %v, want %v", got, wantFG)
	}
	if got := e.PaletteColors(true); !reflect.DeepEqual(got, wantBG) {
		t.Fatalf("Linux background choices = %v, want %v", got, wantBG)
	}
	for _, reported := range []int{0, 2, 256, 1 << 24} {
		e.SetPaletteProfile(palette.Detect(reported, false))
		for _, bg := range []bool{false, true} {
			choices := e.PaletteColors(bg)
			if len(choices) == 0 || len(choices) > 11 {
				t.Fatalf("%d-color terminal offered %d knob choices", reported, len(choices))
			}
			if reported <= 2 && len(choices) > 2 {
				t.Fatalf("monochrome terminal offered %v", choices)
			}
		}
	}
}

func TestAdaptiveKnobAndArrowsCycleTheSameVisibleColors(t *testing.T) {
	for _, terminal := range []struct {
		name    string
		profile palette.Profile
	}{
		{"eight", palette.Detect(8, false)},
		{"linux", palette.Detect(8, true)},
		{"rich", palette.Detect(1<<24, false)},
		{"mono", palette.Detect(0, false)},
	} {
		for _, bg := range []bool{false, true} {
			for _, direction := range []int{-1, 1} {
				t.Run(terminal.name+map[bool]string{false: "/fg", true: "/bg"}[bg]+map[int]string{-1: "/reverse", 1: "/forward"}[direction], func(t *testing.T) {
					knob, arrows := newTestEditor(t), newTestEditor(t)
					for _, e := range []*Editor{knob, arrows} {
						e.Text.FG, e.Text.BG = canvas.Orange, canvas.Gray
						e.SetPaletteProfile(terminal.profile)
						e.TogglePalette()
						if bg {
							e.TogglePalette()
						}
					}
					choices := knob.PaletteColors(bg)
					initial := canvas.Orange
					if bg {
						initial = canvas.Gray
					}
					start := knob.PaletteIndexFor(initial, bg)
					for step := 1; step <= len(choices)+1; step++ {
						knob.AdjustKnobColor(direction, false)
						arrow := tcell.KeyRight
						if direction < 0 {
							arrow = tcell.KeyLeft
						}
						key(arrows, arrow)
						key(arrows, tcell.KeyEnter)
						index := ((start+step*direction)%len(choices) + len(choices)) % len(choices)
						want := choices[index]
						if knob.Text != arrows.Text {
							t.Fatalf("step %d: knob brush %+v differs from arrow brush %+v", step, knob.Text, arrows.Text)
						}
						got := knob.Text.FG
						if bg {
							got = knob.Text.BG
						}
						if got != want {
							t.Fatalf("step %d from alias %s: selected %s, want %s", step, initial, got, want)
						}
						arrows.TogglePalette()
						if bg {
							arrows.TogglePalette()
						}
					}
				})
			}
		}
	}
}

func TestAdaptiveAliasSelectionRemainsLogicalUntilApplied(t *testing.T) {
	e := newTestEditor(t)
	e.Text.FG, e.Text.BG = canvas.Orange, canvas.Gray
	profile := palette.Detect(8, false)
	e.SetPaletteProfile(profile)
	e.TogglePalette()
	if e.PaletteColors(false)[e.PaletteIndex] != canvas.Brown {
		t.Fatal("orange did not anchor at the matching brown swatch")
	}
	key(e, tcell.KeyRight)
	hover := e.PaletteIndex
	e.SetPaletteProfile(profile)
	if e.PaletteIndex != hover || e.Text.FG != canvas.Orange {
		t.Fatal("reapplying the same profile reset arrow selection or normalized the brush")
	}
	e.AdjustKnobColor(0, true)
	if e.Palette != 2 || e.PaletteColors(true)[e.PaletteIndex] != canvas.White || e.Text.BG != canvas.Gray {
		t.Fatal("switching knob focus did not preserve the logical gray background")
	}
	key(e, tcell.KeyLeft)
	hover = e.PaletteIndex
	e.SetPaletteProfile(profile)
	if e.PaletteIndex != hover {
		t.Fatal("repeated terminal detection discarded background arrow selection")
	}
	e.SetPaletteProfile(palette.Detect(256, false))
	if e.Palette != 2 || e.PaletteColors(true)[e.PaletteIndex] != canvas.Gray || e.Text.FG != canvas.Orange || e.Text.BG != canvas.Gray {
		t.Fatal("capability change did not reanchor an open palette without changing logical colors")
	}
}

func TestAdaptivePalettesKeepExistingColorShortcuts(t *testing.T) {
	for _, bg := range []bool{false, true} {
		for _, shortcut := range []struct {
			key  string
			want canvas.Color
		}{
			{"1", canvas.Red}, {"2", canvas.Orange}, {"3", canvas.Yellow},
			{"4", canvas.Green}, {"5", canvas.Blue}, {"6", canvas.Violet},
			{"7", canvas.Gray}, {"8", canvas.Brown}, {"9", canvas.White},
			{"0", canvas.Black}, {"c", canvas.Cyan}, {"C", canvas.Cyan},
		} {
			e := newTestEditor(t)
			e.SetPaletteProfile(palette.Detect(8, false))
			e.TogglePalette()
			if bg {
				e.TogglePalette()
			}
			e.HandleKey(tcell.NewEventKey(tcell.KeyRune, shortcut.key, tcell.ModNone), time.Now())
			got := e.Text.FG
			if bg {
				got = e.Text.BG
			}
			if e.Palette != 0 || got != shortcut.want {
				t.Fatalf("background=%t shortcut %q selected %s, palette=%d; want logical %s", bg, shortcut.key, got, e.Palette, shortcut.want)
			}
		}
	}
}

func TestPaletteCapabilitiesAndFocusPreserveArtworkPreviewAndHistory(t *testing.T) {
	e := newTestEditor(t)
	e.Text.FG, e.Text.BG = canvas.Orange, canvas.Gray
	e.Type('A')
	e.Type('B')
	e.Undo() // Keep an undo entry and a redo entry while displaying a shape preview.
	e.SwitchTool(ToolRectangle)
	e.Cursor = canvas.Point{X: 2, Y: 2}
	e.Enter()
	e.Move(5, 4)
	e.Enter()
	e.Type('.')
	e.Outline.FG, e.Outline.BG = canvas.Yellow, canvas.White
	e.Fill.FG, e.Fill.BG = canvas.Gray, canvas.Orange
	e.Pen.FG, e.Pen.BG = canvas.Cyan, canvas.Yellow
	brushes := [4]canvas.Paint{e.Text, e.Outline, e.Fill, e.Pen}
	cells := append([]canvas.Cell(nil), e.Doc.Cells...)
	preview := e.Preview()
	revision, phase, cursor := e.Revision, e.Phase, e.Cursor
	for _, profile := range []palette.Profile{palette.Detect(8, false), palette.Detect(8, true), palette.Detect(256, false), palette.Detect(0, false)} {
		e.SetPaletteProfile(profile)
		e.TogglePalette()
		e.AdjustKnobColor(0, true)
		e.AdjustKnobColor(0, true)
		key(e, tcell.KeyEsc)
		if got := [4]canvas.Paint{e.Text, e.Outline, e.Fill, e.Pen}; got != brushes {
			t.Fatalf("capability or focus change rewrote brush colors: %+v", got)
		}
		if !reflect.DeepEqual(e.Doc.Cells, cells) || !reflect.DeepEqual(e.Preview(), preview) || e.Phase != phase || e.Cursor != cursor || e.Revision != revision {
			t.Fatal("capability or focus change altered the drawing or pending shape")
		}
		if len(e.undo) != 1 || len(e.redo) != 1 {
			t.Fatal("capability or focus change altered undo/redo history")
		}
	}
	key(e, tcell.KeyEsc) // Cancel the preserved preview, then use the original history.
	e.Redo()
	want := canvas.Cell{Rune: 'B', FG: canvas.Orange, BG: canvas.Gray}
	if got := e.Doc.At(canvas.Point{X: 1}); got != want {
		t.Fatalf("redo lost original drawing colors: %+v, want %+v", got, want)
	}
	e.Undo()
	e.Undo()
	if e.Doc.At(canvas.Point{}) != canvas.Blank() || e.Doc.At(canvas.Point{X: 1}) != canvas.Blank() {
		t.Fatal("original undo entries no longer restore the artwork")
	}
}
