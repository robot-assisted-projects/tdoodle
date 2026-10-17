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
	"tdoodle/internal/palette"
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
	profile, err := palette.ForMode(e.ColorMode, screen.Colors(), os.Getenv("TERM") == "linux")
	if err != nil {
		profile = palette.Detect(screen.Colors(), os.Getenv("TERM") == "linux")
	}
	e.SetPaletteProfile(profile)
	w, h := screen.Size()
	if w <= 0 || h <= 0 {
		screen.HideCursor()
		screen.Show()
		return
	}
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
			style := cellStyle(cell, profile)
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
		renderPalette(screen, e, w, h-1, profile)
	} else {
		renderStatus(screen, e, w, h-1, base, profile)
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

func renderStatus(screen Screen, e *editor.Editor, w, y int, base tcell.Style, profile palette.Profile) {
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
		statusStyle = base.Foreground(profile.Map(canvas.Yellow, false)).Bold(true)
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
			renderKnobSwatches(screen, e, right, w-rightWidth, y, rightWidth, profile)
		}
	}
}

// The focused name is a teal control label, independent of the chosen color.
// The other name is a swatch; Aa always previews the actual drawing colors.
func renderKnobSwatches(screen Screen, e *editor.Editor, text string, x, y, width int, profile palette.Profile) {
	paint := currentPaint(e)
	fg, bg := string(profile.Resolve(paint.FG, false)), string(profile.Resolve(paint.BG, true))
	if offset := strings.Index(text, fg+"/"+bg); offset >= 0 {
		fgStyle := profileSwatchStyle(paint.FG, 1, profile)
		bgStyle := profileSwatchStyle(paint.BG, 2, profile)
		focus := tcell.StyleDefault.Foreground(color.Teal).Background(color.Black)
		if profile.MaxColors(false) == 0 {
			focus = tcell.StyleDefault.Bold(true)
		}
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
		style := tcell.StyleDefault.Foreground(profile.Map(paint.FG, false)).Background(profile.Map(paint.BG, true))
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

func renderPalette(screen Screen, e *editor.Editor, w, y int, profile palette.Profile) {
	base := tcell.StyleDefault.Foreground(color.Silver).Background(color.Black)
	name, short := "Foreground", "FG"
	paint := currentPaint(e)
	background := e.Palette == 2
	selected := paint.FG
	if background {
		name, short = "Background", "BG"
		selected = paint.BG
	}
	selected = profile.Resolve(selected, background)
	approx := profile.MaxColors(background) < 256
	prefix := name + ": "
	if approx {
		prefix = name + " (approx): "
	}
	paletteColors := e.PaletteColors(background)
	labels := make([]string, len(paletteColors))
	for i, c := range paletteColors {
		name := string(c)
		labels[i] = paletteLabel(i, fmt.Sprintf("%s %s", paletteShortcuts(c, background, profile), strings.ToUpper(name[:1])+name[1:]), e.PaletteIndex)
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		prefix = short + ": "
		if approx {
			prefix = short + "~: "
		}
		for i, c := range paletteColors {
			labels[i] = paletteLabel(i, paletteShortcuts(c, background, profile)+paletteAbbreviation(c), e.PaletteIndex)
		}
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		for i, c := range paletteColors {
			labels[i] = paletteLabel(i, paletteShortcuts(c, background, profile), e.PaletteIndex)
		}
	}
	if len(prefix)+len(strings.Join(labels, " ")) > w {
		prefix = short
		if approx {
			prefix += "~"
		}
		prefix += " "
	}
	if len(paletteColors) == 0 {
		putText(screen, 0, y, prefix, w, base)
		return
	}
	i := max(0, min(e.PaletteIndex, len(paletteColors)-1))
	if w <= len(prefix) {
		putText(screen, 0, y, paletteShortcut(paletteColors[i]), w, profileSwatchStyle(paletteColors[i], e.Palette, profile).Underline(true).Bold(true))
		return
	}
	x := putText(screen, 0, y, prefix, w, base)
	// Very narrow displays show the current arrow selection and its name.
	if x+len(strings.Join(labels, " ")) > w {
		c := paletteColors[i]
		label := fmt.Sprintf("%s %s", paletteShortcuts(c, background, profile), string(c))
		if len(label) > w-x {
			label = paletteShortcut(c) + " " + string(c)
		}
		if len(label) > w-x {
			label = paletteShortcut(c) + paletteAbbreviation(c)
		}
		putText(screen, x, y, label, w-x, profileSwatchStyle(c, e.Palette, profile).Underline(true).Bold(true))
		return
	}
	for i, c := range paletteColors {
		if i > 0 {
			x += putText(screen, x, y, " ", w-x, base)
		}
		style := profileSwatchStyle(c, e.Palette, profile)
		if c == selected {
			style = style.Bold(true)
		}
		if i == e.PaletteIndex {
			style = style.Underline(true)
		}
		x += putText(screen, x, y, labels[i], w-x, style)
	}
}

// Shortcut identities are independent of palette order and terminal aliases.
func paletteShortcut(c canvas.Color) string {
	switch c {
	case canvas.Red:
		return "1"
	case canvas.Orange:
		return "2"
	case canvas.Yellow:
		return "3"
	case canvas.Green:
		return "4"
	case canvas.Blue:
		return "5"
	case canvas.Violet:
		return "6"
	case canvas.Gray:
		return "7"
	case canvas.Brown:
		return "8"
	case canvas.White:
		return "9"
	case canvas.Black:
		return "0"
	case canvas.Cyan:
		return "C"
	}
	return ""
}

// Show all existing shortcuts when several logical colors share one shade.
func paletteShortcuts(c canvas.Color, background bool, profile palette.Profile) string {
	var shortcuts []string
	for _, logical := range canvas.BackgroundColors {
		if profile.Resolve(logical, background) == c {
			shortcuts = append(shortcuts, paletteShortcut(logical))
		}
	}
	return strings.Join(shortcuts, "/")
}

func paletteAbbreviation(c canvas.Color) string {
	switch c {
	case canvas.Red:
		return "Red"
	case canvas.Orange:
		return "Org"
	case canvas.Yellow:
		return "Yel"
	case canvas.Green:
		return "Grn"
	case canvas.Blue:
		return "Blu"
	case canvas.Violet:
		return "Vio"
	case canvas.Gray:
		return "Gry"
	case canvas.Brown:
		return "Brn"
	case canvas.White:
		return "Wht"
	case canvas.Black:
		return "Blk"
	case canvas.Cyan:
		return "Cyn"
	}
	return string(c)
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

func swatchStyle(c canvas.Color, which, colors int, linux bool) tcell.Style {
	return profileSwatchStyle(c, which, palette.Detect(colors, linux))
}

func profileSwatchStyle(c canvas.Color, which int, profile palette.Profile) tcell.Style {
	if profile.MaxColors(false) == 0 {
		return tcell.StyleDefault
	}
	if which == 1 {
		fg := profile.Map(c, false)
		bg := color.Black
		if profile.Resolve(c, false) == canvas.Black {
			bg = color.Silver
		}
		return tcell.StyleDefault.Foreground(fg).Background(bg)
	}
	bg := profile.Map(c, true)
	r, g, b := bg.RGB()
	fg := profile.Map(canvas.White, false)
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
