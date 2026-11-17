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
	leftPad := min(2, max(0, width-1))
	rightPad := min(2, max(0, width-leftPad-1))
	contentWidth := width - leftPad - rightPad
	prefix, separator := "Drawing "+e.Tool.String()+"  ", " --> "
	if e.Help || e.Toolbar || e.Picker || e.Palette != 0 {
		prefix = "Drawing " + e.Tool.String() + " (paused)  "
	}
	if len(prefix)+len(strings.Join(labels, separator)) > contentWidth {
		separator = ">"
	}
	if len(prefix)+len(strings.Join(labels, separator)) > contentWidth {
		prefix = ""
	}
	first, last := 0, len(labels)
	if len(strings.Join(labels, separator)) > contentWidth {
		// Always keep the active step when the entire chain cannot fit.
		first, last = active, active+1
		if active+1 < len(labels) && len(labels[active])+len(separator)+len(labels[active+1]) <= contentWidth {
			last++
		}
	}
	x := putText(screen, 0, y, strings.Repeat(" ", leftPad), width, base)
	limit := leftPad + contentWidth
	x += putText(screen, x, y, prefix, limit-x, base.Bold(true))
	for i := first; i < last && x < limit; i++ {
		if i > first {
			x += putText(screen, x, y, separator, limit-x, base)
		}
		style := base
		if i == active {
			style = style.Bold(true).Reverse(true)
		}
		x += putText(screen, x, y, labels[i], limit-x, style)
	}
	putText(screen, x, y, strings.Repeat(" ", rightPad), width-x, base)
}
