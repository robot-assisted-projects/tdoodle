package editor

import (
	"reflect"
	"strings"
	"testing"

	"tdoodle/internal/canvas"
)

func TestKnobColorsApplyWithoutEnterAndWrap(t *testing.T) {
	doc, _ := canvas.New(10, 10, 2)
	e := New(doc)
	e.AdjustKnobColor(1, false) // white -> black
	e.AdjustKnobColor(1, false) // black -> cyan
	e.AdjustKnobColor(1, false) // cyan -> red
	if e.Text.FG != canvas.Red || e.Text.BG != canvas.Black || e.Palette != 0 {
		t.Fatalf("foreground rotation: %+v", e.Text)
	}
	e.AdjustKnobColor(-1, false)
	if e.Text.FG != canvas.Cyan {
		t.Fatalf("reverse wrap to cyan: %s", e.Text.FG)
	}
	e.AdjustKnobColor(-1, false)
	if e.Text.FG != canvas.Black {
		t.Fatalf("reverse wrap: %s", e.Text.FG)
	}
	e.AdjustKnobColor(0, true)
	e.AdjustKnobColor(1, false) // black -> cyan, BG only
	if e.Text.BG != canvas.Cyan || e.Text.FG != canvas.Black {
		t.Fatalf("cyan background selection: %+v", e.Text)
	}
	e.AdjustKnobColor(1, false) // cyan wraps to red
	if e.Text.FG != canvas.Black || e.Text.BG != canvas.Red || !strings.Contains(e.Status().Text, "Knob: BG") {
		t.Fatalf("background rotation: %+v, %s", e.Text, e.Status().Text)
	}
	e.Type('X')
	cell := doc.At(canvas.Point{X: 0, Y: 0})
	if cell.Rune != 'X' || cell.FG != canvas.Black || cell.BG != canvas.Red {
		t.Fatalf("typing did not use knob colors: %+v", cell)
	}
}

func TestKnobKeepsShapePreviewAndChangesFillOnly(t *testing.T) {
	doc, _ := canvas.New(10, 10, 2)
	e := New(doc)
	e.SwitchTool(ToolRectangle)
	e.Enter()
	e.Move(4, 4)
	e.Enter() // Fill phase
	outline := e.Outline
	before := append([]canvas.Cell(nil), doc.Cells...)
	e.AdjustKnobColor(1, false)
	e.AdjustKnobColor(0, true)
	e.AdjustKnobColor(-1, false)
	if e.Phase != 2 || e.Outline != outline || !reflect.DeepEqual(before, doc.Cells) || e.Revision != 0 {
		t.Fatal("knob changed document, outline, or shape phase")
	}
	if e.Fill.FG != canvas.Black || e.Fill.BG != canvas.White {
		t.Fatalf("fill colors = %+v", e.Fill)
	}
	e.SwitchTool(ToolPen)
	if e.Pen.FG != canvas.Black || e.Pen.BG != canvas.White {
		t.Fatalf("next tool lost selected colors: %+v", e.Pen)
	}
}

func TestKnobAndPaletteShareTarget(t *testing.T) {
	doc, _ := canvas.New(10, 10, 2)
	e := New(doc)
	e.TogglePalette()
	e.TogglePalette() // BG
	e.AdjustKnobColor(1, false)
	if e.Palette != 2 || e.PaletteIndex != 10 || e.Text.BG != canvas.Cyan {
		t.Fatal("knob did not apply current palette target")
	}
	e.AdjustKnobColor(0, true)
	if e.Palette != 1 || e.PaletteIndex != 8 || e.Text.FG != canvas.White {
		t.Fatal("press did not switch palette to FG without changing the color")
	}
}
