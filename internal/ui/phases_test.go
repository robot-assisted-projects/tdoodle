package ui

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
	"tdoodle/internal/editor"
)

func TestPhaseBarFollowsShapeForwardAndBack(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	for _, tc := range []struct {
		tool  editor.Tool
		steps []string
	}{
		{editor.ToolLine, []string{"start", "stroke", "draw"}},
		{editor.ToolRectangle, []string{"corner", "stroke", "fill", "draw"}},
		{editor.ToolOval, []string{"center", "stroke", "stretch", "fill", "draw"}},
	} {
		t.Run(tc.tool.String(), func(t *testing.T) {
			e := newEditor(t)
			e.SwitchTool(tc.tool)
			e.Cursor = canvas.Point{X: 5, Y: 2}
			s := newScreen(80, 12, 256)
			Render(s, e)
			if strings.Contains(s.row(10), "[stroke]") {
				t.Fatal("phase row appeared before starting a shape")
			}
			before := append([]canvas.Cell(nil), e.Doc.Cells...)
			check := func(phase int) {
				t.Helper()
				Render(s, e)
				row := s.row(10)
				if !strings.HasPrefix(row, "  Drawing "+tc.tool.String()+"  ") {
					t.Fatalf("missing shape name or left padding: %q", row)
				}
				want := "[" + strings.Join(tc.steps, "] --> [") + "]"
				if !strings.Contains(row, want) {
					t.Fatalf("phase %d lost full sequence %q: %q", phase, want, row)
				}
				for i, step := range tc.steps {
					start := strings.Index(row, "["+step+"]")
					for x := start; x < start+len(step)+2; x++ {
						_, style, _ := s.buffer.Get(x, 10)
						if style.HasReverse() || style.HasBold() != (i == phase) {
							t.Fatalf("phase %d: %s has incorrect highlighting", phase, step)
						}
						if i == phase && (style.GetBackground() != color.Lime || style.GetForeground() != color.Black) {
							t.Fatalf("phase %d: %s must show black text on lime", phase, step)
						}
						if i != phase && style.GetBackground() != color.Black {
							t.Fatalf("phase %d: %s lost the strip's black background", phase, step)
						}
					}
				}
				if s.outside || e.Phase != phase || !reflect.DeepEqual(before, e.Doc.Cells) {
					t.Fatal("phase rendering exceeded the screen or changed the pending drawing")
				}
			}
			for phase := 1; phase < len(tc.steps)-1; phase++ {
				e.Enter()
				check(phase)
				e.Move(2, 1)
			}
			for phase := len(tc.steps) - 3; phase > 0; phase-- {
				e.Backspace()
				check(phase)
			}
			for e.PendingShape() {
				e.Enter()
			}
			Render(s, e)
			if strings.Contains(s.row(10), "[stroke]") || e.StatusRows(12) != 1 {
				t.Fatal("committed shape retained the phase row")
			}
		})
	}
}

func TestPhaseBarKeepsCursorAboveStatusAndRestoresCanvas(t *testing.T) {
	e := newEditor(t)
	e.Cursor = canvas.Point{X: 5, Y: 4}
	for x := 0; x < 80; x++ {
		e.Doc.Set(canvas.Point{X: x, Y: 5}, canvas.Cell{Rune: '.', FG: canvas.White, BG: canvas.Blue})
	}
	e.Doc.Set(canvas.Point{X: 10, Y: 5}, canvas.Cell{Rune: '@', FG: canvas.Green, BG: canvas.Black})
	before := append([]canvas.Cell(nil), e.Doc.Cells...)
	s := newScreen(80, 6, 256)
	Render(s, e)
	e.SwitchTool(editor.ToolLine)
	e.Enter()
	Render(s, e)
	if s.cursor.Y != 3 || e.Cursor.Y != 4 || e.Offset.Y != 1 {
		t.Fatalf("phase row covered cursor instead of scrolling: cursor=%v offset=%v", s.cursor, e.Offset)
	}
	if !strings.Contains(s.row(4), "[stroke]") {
		t.Fatal("missing phase row above the main status")
	}
	end := strings.Index(s.row(4), "[draw]") + len("[draw]")
	for x := end; x < end+2; x++ {
		text, style, _ := s.buffer.Get(x, 4)
		if text != " " || style.GetBackground() != color.Black || style.HasReverse() {
			t.Fatal("phase strip must end with two spaces on black")
		}
	}
	text, style, _ := s.buffer.Get(end+2, 4)
	if text != "." || style.GetBackground() != ColorFor(canvas.Blue, true, 256, false) {
		t.Fatal("phase strip extended over the canvas beyond its padding")
	}
	e.HandleKey(tcell.NewEventKey(tcell.KeyEsc, "", tcell.ModNone), time.Now())
	Render(s, e)
	if text, _, _ := s.buffer.Get(10, 4); text != "@" {
		t.Fatalf("closing the phase row did not restore underlying canvas: %q", text)
	}
	if e.Doc.Width != 80 || e.Doc.Height != 40 || !reflect.DeepEqual(before, e.Doc.Cells) {
		t.Fatal("temporary phase row changed document dimensions or cells")
	}
	e.Enter()
	s = newScreen(24, 3, 0)
	Render(s, e)
	if s.cursor.Y != 0 || !strings.Contains(s.row(1), "[stroke]") || s.outside {
		t.Fatal("resizing an unfinished shape hid its cursor or phase")
	}
}

func TestPhaseBarFitsSmallScreensAndPausedOverlays(t *testing.T) {
	for _, colors := range []int{0, 8, 16, 256} {
		for _, mode := range []string{"drawing", "palette", "help", "toolbar", "picker", "message"} {
			for _, size := range [][2]int{{0, 0}, {0, 4}, {1, 1}, {8, 2}, {1, 3}, {8, 3}, {16, 4}, {24, 4}, {40, 4}, {80, 8}} {
				t.Run(fmt.Sprintf("%s/%d-colors/%dx%d", mode, colors, size[0], size[1]), func(t *testing.T) {
					e := newEditor(t)
					e.SwitchTool(editor.ToolOval)
					e.Enter()
					e.Move(4, 0)
					e.Enter()
					switch mode {
					case "palette":
						e.TogglePalette()
					case "help":
						e.Help = true
					case "toolbar":
						e.Toolbar = true
					case "picker":
						e.TogglePicker()
					case "message":
						e.Message = "Saved drawing"
					}
					s := newScreen(size[0], size[1], colors)
					Render(s, e)
					if s.outside || s.shown != 1 || e.Phase != 2 {
						t.Fatal("phase row exceeded bounds or changed the operation")
					}
					if s.cursor.X >= 0 && s.cursor.Y >= size[1]-e.StatusRows(size[1]) {
						t.Fatal("cursor overlapped a status row")
					}
					if size[0] >= 16 && size[1] >= 3 {
						row := s.row(size[1] - 2)
						x := strings.Index(row, "[stretch]")
						if x < 0 {
							t.Fatalf("overlay or clipping hid active step: %q", row)
						}
						_, style, _ := s.buffer.Get(x, size[1]-2)
						if !style.HasBold() || style.HasReverse() != (colors == 0) {
							t.Fatal("active step lost highlighting")
						}
					}
				})
			}
		}
	}
}

func TestKeyboardColorSelectionShowsBrushSampleWithoutKnob(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	for _, tc := range []struct {
		tool  editor.Tool
		phase int
	}{
		{editor.ToolText, 0}, {editor.ToolPen, 0}, {editor.ToolLine, 1},
		{editor.ToolRectangle, 1}, {editor.ToolRectangle, 2},
		{editor.ToolOval, 1}, {editor.ToolOval, 2}, {editor.ToolOval, 3},
	} {
		t.Run(fmt.Sprintf("%s/%d", tc.tool, tc.phase), func(t *testing.T) {
			e := newEditor(t)
			e.SwitchTool(tc.tool)
			for phase := 0; phase < tc.phase; phase++ {
				e.Enter()
			}
			e.TogglePalette()
			e.HandleKey(tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModNone), time.Now())
			e.TogglePalette()
			e.TogglePalette()
			e.HandleKey(tcell.NewEventKey(tcell.KeyRune, "5", tcell.ModNone), time.Now())
			for _, width := range []int{24, 40, 80, 160} {
				s := newScreen(width, 8, 256)
				Render(s, e)
				row := s.row(7)
				x := strings.Index(row, "[Aa]")
				if x < 0 {
					t.Fatalf("%d-column keyboard status lost color sample: %q", width, row)
				}
				for _, column := range []int{x + 1, x + 2} {
					_, style, _ := s.buffer.Get(column, 7)
					if style.GetForeground() != ColorFor(canvas.Red, false, 256, false) || style.GetBackground() != ColorFor(canvas.Blue, true, 256, false) {
						t.Fatal("keyboard-selected foreground/background did not appear in sample")
					}
				}
				if e.KnobUsed || e.Palette != 0 || e.Phase != tc.phase || s.outside {
					t.Fatal("sample rendering changed input state or exceeded bounds")
				}
			}
		})
	}
}
