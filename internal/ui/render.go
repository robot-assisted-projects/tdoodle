// Package ui renders the editor without modifying its document or operation.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/clipperhouse/displaywidth"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
	"tdoodle/internal/editor"
)

// Screen contains the display operations used by the renderer.  Keeping input
// outside this interface makes rendering testable without an actual terminal.
type Screen interface {
	Size() (int, int)
	Put(int, int, string, tcell.Style) (string, int)
	ShowCursor(int, int)
	HideCursor()
	Show()
	Colors() int
}

// Render redraws the viewport and its single status row. Uncommitted operations
// are composited over the document, so previews never become permanent here.
func Render(screen Screen, e *editor.Editor) {
	w, h := screen.Size()
	if w <= 0 || h <= 0 {
		screen.HideCursor()
		screen.Show()
		return
	}
	colors := screen.Colors()
	linux := os.Getenv("TERM") == "linux"
	blank := canvas.Blank()
	preview := e.Preview()
	cursor := e.DisplayCursor()
	for y := 0; y < h-1; y++ {
		for x := 0; x < w; x++ {
			point := canvas.Point{X: x + e.Offset.X, Y: y + e.Offset.Y}
			cell := blank
			if e.Doc != nil {
				cell = e.Doc.At(point)
			}
			// Reveal the source cell under the sampler, even beneath a preview.
			if overlay, ok := preview[point]; ok && !(e.Picker && point == cursor) {
				cell = overlay
			}
			glyph := cell.Rune
			if glyph == 0 {
				glyph = ' '
			}
			style := cellStyle(cell, colors, linux)
			if e.Picker && point == e.Cursor {
				style = style.Underline(true)
			}
			if point == cursor && !e.Help && e.Palette == 0 && !e.Toolbar {
				style = style.Reverse(true)
				if e.Picker {
					style = style.Bold(true)
				}
			}
			screen.Put(x, y, string(glyph), style)
		}
	}
	base := tcell.StyleDefault.Foreground(color.Silver).Background(color.Black)
	for x := 0; x < w; x++ {
		screen.Put(x, h-1, " ", base)
	}
	if e.Palette != 0 {
		renderPalette(screen, e, w, h-1, colors, linux)
	} else {
		renderStatus(screen, e, w, h-1, base, colors, linux)
	}
	if e.Help {
		renderHelp(screen, e.HelpScroll, w, h-1, base)
	}
	cx, cy := cursor.X-e.Offset.X, cursor.Y-e.Offset.Y
	if !e.Help && e.Palette == 0 && !e.Toolbar && cx >= 0 && cx < w && cy >= 0 && cy < h-1 {
		screen.ShowCursor(cx, cy)
	} else {
		screen.HideCursor()
	}
	screen.Show()
}

var toolNames = []string{"Text", "Oval", "Rectangle", "Line", "Pen", "Colors", "Picker", "Help"}
var toolShort = []string{"Text", "Oval", "Rect", "Line", "Pen", "Color", "Pick", "Help"}
var toolTiny = []string{"T", "O", "R", "L", "P", "C", "E", "?"}

func toolbarLabels(kind int) []string {
	labels := make([]string, len(toolNames))
	for i := range labels {
		switch kind {
		case 0:
			labels[i] = fmt.Sprintf("%s [F%d]", toolNames[i], i+1)
		case 1:
			labels[i] = fmt.Sprintf("%s F%d", toolShort[i], i+1)
		default:
			labels[i] = fmt.Sprintf("%s%d", toolTiny[i], i+1)
		}
	}
	return labels
}

func renderStatus(screen Screen, e *editor.Editor, w, y int, base tcell.Style, colors int, linux bool) {
	active := int(e.Tool)
	if active < 0 || active > 4 {
		active = 0
	}
	if e.Picker {
		active = 6
	}
	if e.Help {
		active = 7
	}
	minimum := fmt.Sprintf("%s%d", toolTiny[active], active+1)
	status := e.Status()
	statusStyle := base
	if status.Warning {
		statusStyle = base.Foreground(ColorFor(canvas.Yellow, false, colors, linux)).Bold(true)
	} else if e.Message != "" {
		statusStyle = base.Bold(true)
	}
	// Reserve the active tool even when a long prompt needs most of the row.
	available := max(0, w-len(minimum)-1)
	right := statusText(status, available)
	rightWidth := min(displaywidth.String(right), available)
	if w <= len(minimum) {
		putText(screen, 0, y, minimum, w, statusStyle.Reverse(true).Bold(true))
		return
	}
	leftWidth := w
	if rightWidth > 0 {
		leftWidth -= rightWidth + 1
	}
	labels := toolbarLabels(0)
	if len(strings.Join(labels, " | ")) > leftWidth {
		labels = toolbarLabels(1)
	}
	if len(strings.Join(labels, " | ")) > leftWidth {
		labels = toolbarLabels(2)
	}
	if len(strings.Join(labels, " | ")) > leftWidth {
		index := active
		if e.Toolbar && e.ToolbarIndex >= 0 && e.ToolbarIndex < len(labels) {
			index = e.ToolbarIndex
		}
		label := fmt.Sprintf("%s F%d", toolShort[index], index+1)
		if len(label) > leftWidth {
			label = labels[index]
		}
		style := base.Reverse(true).Bold(true)
		putText(screen, 0, y, label, leftWidth, style)
	} else {
		x := 0
		for i, label := range labels {
			if i > 0 {
				x += putText(screen, x, y, " | ", leftWidth-x, base)
			}
			style := base
			if i == active {
				style = style.Bold(true).Reverse(true)
			}
			if e.Toolbar && i == e.ToolbarIndex {
				style = style.Underline(true).Reverse(true)
			}
			x += putText(screen, x, y, label, leftWidth-x, style)
		}
	}
	if rightWidth > 0 {
		putText(screen, w-rightWidth, y, right, rightWidth, statusStyle)
		if e.KnobUsed && e.Message == "" && !e.Picker && !e.Help {
			renderKnobSwatches(screen, e, right, w-rightWidth, y, rightWidth, colors, linux)
		}
	}
}

// The focused name is a teal control label, independent of the chosen color.
// The other name is a swatch; Aa always previews the actual drawing colors.
func renderKnobSwatches(screen Screen, e *editor.Editor, text string, x, y, width, colors int, linux bool) {
	paint := currentPaint(e)
	fg, bg := string(paint.FG), string(paint.BG)
	if offset := strings.Index(text, fg+"/"+bg); offset >= 0 {
		fgStyle := swatchStyle(paint.FG, 1, colors, linux)
		bgStyle := swatchStyle(paint.BG, 2, colors, linux)
		focus := tcell.StyleDefault.Foreground(color.Teal).Background(color.Black)
		if e.KnobBackground {
			bgStyle = focus
		} else {
			fgStyle = focus
		}
		putText(screen, x+offset, y, fg, max(0, width-offset), fgStyle)
		offset += len(fg) + 1
		putText(screen, x+offset, y, bg, max(0, width-offset), bgStyle)
	}
	if offset := strings.Index(text, "[Aa]"); offset >= 0 {
		offset++
		style := tcell.StyleDefault.Foreground(ColorFor(paint.FG, false, colors, linux)).Background(ColorFor(paint.BG, true, colors, linux))
		putText(screen, x+offset, y, "Aa", max(0, width-offset), style)
	}
}

func statusText(status editor.StatusInfo, width int) string {
	for _, text := range []string{status.Text, status.Compact, status.Minimal} {
		if text != "" && displaywidth.String(text) <= width {
			return text
		}
	}
	if status.Fallback != "" {
		return status.Fallback
	}
	if status.Warning {
		return "PREVIEW"
	}
	if status.Minimal != "" {
		return status.Minimal
	}
	return status.Text
}

func renderPalette(screen Screen, e *editor.Editor, w, y, colors int, linux bool) {
	base := tcell.StyleDefault.Foreground(color.Silver).Background(color.Black)
	name, short := "Foreground", "FG"
	paint := currentPaint(e)
	selected := paint.FG
	if e.Palette == 2 {
		name, short = "Background", "BG"
		selected = paint.BG
	}
	approx := colors < 256 || (linux && e.Palette == 2)
	prefix := name + ": "
	if approx {
		prefix = name + " (approx): "
	}
	paletteColors := canvas.PaletteColors(e.Palette == 2)
	labels := make([]string, len(paletteColors))
	for i, c := range paletteColors {
		name := string(c)
		labels[i] = paletteLabel(i, fmt.Sprintf("%s %s", paletteShortcut(i), strings.ToUpper(name[:1])+name[1:]), e.PaletteIndex)
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		prefix = short + ": "
		if approx {
			prefix = short + "~: "
		}
		abbreviated := []string{"Red", "Org", "Yel", "Grn", "Blu", "Vio", "Gry", "Brn", "Wht", "Blk", "Cyn"}
		for i := range labels {
			labels[i] = paletteLabel(i, paletteShortcut(i)+abbreviated[i], e.PaletteIndex)
		}
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		for i := range labels {
			labels[i] = paletteLabel(i, paletteShortcut(i), e.PaletteIndex)
		}
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		prefix = short
		if approx {
			prefix += "~"
		}
		prefix += " "
	}
	if w <= len(prefix) {
		i := max(0, min(e.PaletteIndex, len(paletteColors)-1))
		putText(screen, 0, y, paletteShortcut(i), w, swatchStyle(paletteColors[i], e.Palette, colors, linux).Underline(true).Bold(true))
		return
	}
	x := putText(screen, 0, y, prefix, w, base)
	// Very narrow displays show the current arrow selection and its name.
	if x+len(strings.Join(labels, " ")) > w {
		i := max(0, min(e.PaletteIndex, len(paletteColors)-1))
		label := fmt.Sprintf("%s %s", paletteShortcut(i), string(paletteColors[i]))
		putText(screen, x, y, label, w-x, swatchStyle(paletteColors[i], e.Palette, colors, linux).Underline(true).Bold(true))
		return
	}
	for i, c := range paletteColors {
		if i > 0 {
			x += putText(screen, x, y, " ", w-x, base)
		}
		style := swatchStyle(c, e.Palette, colors, linux)
		if c == selected {
			style = style.Bold(true)
		}
		if i == e.PaletteIndex {
			style = style.Underline(true)
		}
		x += putText(screen, x, y, labels[i], w-x, style)
	}
}

func paletteShortcut(index int) string {
	if index == 10 {
		return "C"
	}
	return fmt.Sprint((index + 1) % 10)
}

func paletteLabel(index int, label string, selected int) string {
	if index == selected {
		return "[" + label + "]"
	}
	return label
}

func currentPaint(e *editor.Editor) canvas.Paint {
	switch e.Tool {
	case editor.ToolText:
		return e.Text
	case editor.ToolPen:
		return e.Pen
	case editor.ToolOval:
		if e.Phase == 3 {
			return e.Fill
		}
	case editor.ToolRectangle:
		if e.Phase == 2 {
			return e.Fill
		}
	}
	return e.Outline
}

func swatchStyle(c canvas.Color, palette, colors int, linux bool) tcell.Style {
	if palette == 1 {
		fg := ColorFor(c, false, colors, linux)
		bg := color.Black
		if c == canvas.Black {
			bg = color.Silver
		}
		return tcell.StyleDefault.Foreground(fg).Background(bg)
	}
	bg := ColorFor(c, true, colors, linux)
	r, g, b := bg.RGB()
	fg := color.White
	if colors < 16 {
		fg = color.Silver
	}
	if 299*r+587*g+114*b > 100000 {
		fg = color.Black
	}
	return tcell.StyleDefault.Foreground(fg).Background(bg)
}

// putText writes only inside the requested span, advancing by displayed width.
func putText(screen Screen, x, y int, text string, width int, style tcell.Style) int {
	used := 0
	graphemes := displaywidth.StringGraphemes(text)
	for graphemes.Next() && used < width {
		cells := graphemes.Width()
		if cells <= 0 {
			continue
		}
		if cells > width-used {
			break
		}
		screen.Put(x+used, y, graphemes.Value(), style)
		used += cells
	}
	return used
}
