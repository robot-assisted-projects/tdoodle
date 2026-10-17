package editor

// AdjustKnobColor changes the active brush without advancing a shape, painting
// a cell, or taking keyboard focus. A press selects FG/BG; rotation applies now.
func (e *Editor) AdjustKnobColor(step int, toggle bool) {
	e.KnobUsed = true
	e.quitKey = ""
	if e.Palette != 0 {
		e.KnobBackground = e.Palette == 2
	}
	if toggle {
		e.KnobBackground = !e.KnobBackground
	}
	palette := 1
	if e.KnobBackground {
		palette = 2
	}
	wasOpen := e.Palette != 0
	b := e.brush()
	selected := b.FG
	if e.KnobBackground {
		selected = b.BG
	}
	colors := e.PaletteColors(e.KnobBackground)
	index := e.PaletteIndexFor(selected, e.KnobBackground)
	index = (index + step%len(colors) + len(colors)) % len(colors)
	if step != 0 {
		e.Palette = palette
		e.SetColor(colors[index])
	}
	if wasOpen {
		e.Palette, e.PaletteIndex = palette, index
	}
}
