package ui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"

	"tdoodle/internal/canvas"
	"tdoodle/internal/palette"
)

// ColorFor maps a document's logical color to an automatically detected display.
// Rendering uses the editor's profile so an explicit -colors override applies to
// artwork, swatches, and previews alike. Saved logical colors remain intact.
func ColorFor(c canvas.Color, background bool, colors int, linux bool) color.Color {
	return palette.Detect(colors, linux).Map(c, background)
}

func cellStyle(c canvas.Cell, profile palette.Profile) tcell.Style {
	return tcell.StyleDefault.
		Foreground(profile.Map(c.FG, false)).
		Background(profile.Map(c.BG, true))
}
