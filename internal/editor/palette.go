package editor

import (
	"tdoodle/internal/canvas"
	"tdoodle/internal/palette"
)

// SetPaletteProfile updates available controls without recoloring the document
// or the brushes. Repeated renders keep the independent arrow selection intact.
func (e *Editor) SetPaletteProfile(profile palette.Profile) {
	if e.colorProfile == profile {
		return
	}
	e.colorProfile = profile
	if e.Palette != 0 {
		selected := e.brush().FG
		if e.Palette == 2 {
			selected = e.brush().BG
		}
		e.PaletteIndex = e.PaletteIndexFor(selected, e.Palette == 2)
	}
}

func (e *Editor) PaletteProfile() palette.Profile { return e.colorProfile }

func (e *Editor) PaletteColors(background bool) []canvas.Color {
	return e.colorProfile.Colors(background)
}

func (e *Editor) PaletteIndexFor(c canvas.Color, background bool) int {
	selected := e.colorProfile.Resolve(c, background)
	for i, candidate := range e.PaletteColors(background) {
		if candidate == selected {
			return i
		}
	}
	return 0
}

func (e *Editor) displayColorNames(fg, bg canvas.Color) string {
	return string(e.colorProfile.Resolve(fg, false)) + "/" + string(e.colorProfile.Resolve(bg, true))
}
