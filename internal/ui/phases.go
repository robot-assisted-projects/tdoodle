package ui

import (
	"strings"

	"github.com/gdamore/tcell/v3"

	"tdoodle/internal/editor"
)

// The last label is the final Enter action, not another editing phase.
func shapeSteps(tool editor.Tool) []string {
	switch tool {
	case editor.ToolOval:
		return []string{"center", "stroke", "stretch", "fill", "draw"}
	case editor.ToolRectangle:
		return []string{"corner", "stroke", "fill", "draw"}
	case editor.ToolLine:
		return []string{"start", "stroke", "draw"}
	}
	return nil
}

func renderPhases(screen Screen, e *editor.Editor, width, y int, base tcell.Style) {
	steps := shapeSteps(e.Tool)
	active := e.Phase
	if active <= 0 || active >= len(steps)-1 {
		return
	}
	labels := make([]string, len(steps))
	for i, step := range steps {
		labels[i] = "[" + step + "]"
	}
	prefix, separator := "PREVIEW  ", " --> "
	if e.Help || e.Toolbar || e.Picker || e.Palette != 0 {
		prefix = "PREVIEW paused  "
	}
	if len(prefix)+len(strings.Join(labels, separator)) > width {
		prefix = ""
	}
	if len(strings.Join(labels, separator)) > width {
		separator = ">"
	}
	first, last := 0, len(labels)
	if len(strings.Join(labels, separator)) > width {
		// Always keep the active step when the entire chain cannot fit.
		first, last = active, active+1
		if active+1 < len(labels) && len(labels[active])+len(separator)+len(labels[active+1]) <= width {
			last++
		}
	}
	x := putText(screen, 0, y, prefix, width, base.Bold(true))
	for i := first; i < last && x < width; i++ {
		if i > first {
			x += putText(screen, x, y, separator, width-x, base)
		}
		style := base
		if i == active {
			style = style.Bold(true).Reverse(true)
		}
		x += putText(screen, x, y, labels[i], width-x, style)
	}
}
